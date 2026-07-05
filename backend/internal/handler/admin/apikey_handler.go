package admin

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// apiKeyManager captures the APIKeyService operations the admin handler needs.
// Declared as an interface so handler tests can stub it.
type apiKeyManager interface {
	Create(ctx context.Context, userID int64, req service.CreateAPIKeyRequest) (*service.APIKey, error)
	Update(ctx context.Context, id int64, userID int64, req service.UpdateAPIKeyRequest) (*service.APIKey, error)
	Delete(ctx context.Context, id int64, userID int64) error
	GetByID(ctx context.Context, id int64) (*service.APIKey, error)
}

// AdminAPIKeyHandler handles admin API key management
type AdminAPIKeyHandler struct {
	adminService service.AdminService
	apiKeys      apiKeyManager
}

// NewAdminAPIKeyHandler creates a new admin API key handler
func NewAdminAPIKeyHandler(adminService service.AdminService, apiKeyService *service.APIKeyService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		adminService: adminService,
		apiKeys:      apiKeyService,
	}
}

// AdminCreateAPIKeyRequest represents the admin request to issue an API key for a user.
type AdminCreateAPIKeyRequest struct {
	Name          string   `json:"name" binding:"required"`
	GroupID       *int64   `json:"group_id"`
	CustomKey     *string  `json:"custom_key"`
	IPWhitelist   []string `json:"ip_whitelist"`
	IPBlacklist   []string `json:"ip_blacklist"`
	Quota         *float64 `json:"quota"`
	ExpiresInDays *int     `json:"expires_in_days"`
	RateLimit5h   *float64 `json:"rate_limit_5h"`
	RateLimit1d   *float64 `json:"rate_limit_1d"`
	RateLimit7d   *float64 `json:"rate_limit_7d"`
}

// CreateForUser issues a new API key bound to the specified user (员工开卡).
// POST /api/v1/admin/users/:id/api-keys
func (h *AdminAPIKeyHandler) CreateForUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req AdminCreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.CreateAPIKeyRequest{
		Name:          req.Name,
		GroupID:       req.GroupID,
		CustomKey:     req.CustomKey,
		IPWhitelist:   req.IPWhitelist,
		IPBlacklist:   req.IPBlacklist,
		ExpiresInDays: req.ExpiresInDays,
	}
	if req.Quota != nil {
		svcReq.Quota = *req.Quota
	}
	if req.RateLimit5h != nil {
		svcReq.RateLimit5h = *req.RateLimit5h
	}
	if req.RateLimit1d != nil {
		svcReq.RateLimit1d = *req.RateLimit1d
	}
	if req.RateLimit7d != nil {
		svcReq.RateLimit7d = *req.RateLimit7d
	}

	idempotencyPayload := struct {
		UserID int64                    `json:"user_id"`
		Req    AdminCreateAPIKeyRequest `json:"req"`
	}{UserID: userID, Req: req}

	executeAdminIdempotentJSON(c, "admin.users.api_keys.create", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		key, err := h.apiKeys.Create(ctx, userID, svcReq)
		if err != nil {
			return nil, err
		}
		return dto.APIKeyFromService(key), nil
	})
}

// AdminUpdateAPIKeyRequest represents the request to update an API key's admin-managed fields.
type AdminUpdateAPIKeyRequest struct {
	GroupID             *int64 `json:"group_id"`               // nil=不修改, 0=解绑, >0=绑定到目标分组
	ResetRateLimitUsage *bool  `json:"reset_rate_limit_usage"` // true=重置 5h/1d/7d 限速用量

	Name        *string  `json:"name"`
	Status      *string  `json:"status" binding:"omitempty,oneof=active disabled"`
	Quota       *float64 `json:"quota"`       // nil=不修改, 0=无限制
	ResetQuota  *bool    `json:"reset_quota"` // true=已用配额清零
	ExpiresAt   *string  `json:"expires_at"`  // ISO 8601；空字符串=清除过期时间
	RateLimit5h *float64 `json:"rate_limit_5h"`
	RateLimit1d *float64 `json:"rate_limit_1d"`
	RateLimit7d *float64 `json:"rate_limit_7d"`
}

func (r *AdminUpdateAPIKeyRequest) hasFieldUpdates() bool {
	return r.Name != nil || r.Status != nil || r.Quota != nil || r.ResetQuota != nil ||
		r.ExpiresAt != nil || r.RateLimit5h != nil || r.RateLimit1d != nil || r.RateLimit7d != nil
}

// UpdateGroup handles updating an API key's admin-managed fields.
// PUT /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}

	var req AdminUpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	var resetKey *service.APIKey
	if req.ResetRateLimitUsage != nil && *req.ResetRateLimitUsage {
		resetKey, err = h.adminService.AdminResetAPIKeyRateLimitUsage(c.Request.Context(), keyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	var fieldUpdatedKey *service.APIKey
	if req.hasFieldUpdates() {
		fieldUpdatedKey, err = h.applyFieldUpdates(c, keyID, &req)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if fieldUpdatedKey == nil {
			// applyFieldUpdates already wrote a 400 response
			return
		}
	}

	result, err := h.adminService.AdminUpdateAPIKeyGroupID(c.Request.Context(), keyID, req.GroupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.GroupID == nil {
		if fieldUpdatedKey != nil {
			result.APIKey = fieldUpdatedKey
		} else if resetKey != nil {
			result.APIKey = resetKey
		}
	}

	resp := struct {
		APIKey                 *dto.APIKey `json:"api_key"`
		AutoGrantedGroupAccess bool        `json:"auto_granted_group_access"`
		GrantedGroupID         *int64      `json:"granted_group_id,omitempty"`
		GrantedGroupName       string      `json:"granted_group_name,omitempty"`
	}{
		APIKey:                 dto.APIKeyFromService(result.APIKey),
		AutoGrantedGroupAccess: result.AutoGrantedGroupAccess,
		GrantedGroupID:         result.GrantedGroupID,
		GrantedGroupName:       result.GrantedGroupName,
	}
	response.Success(c, resp)
}

// applyFieldUpdates updates non-group fields (name/status/quota/expiry/rate limits) on behalf
// of the key owner. Returns (nil, nil) when a validation response has already been written.
func (h *AdminAPIKeyHandler) applyFieldUpdates(c *gin.Context, keyID int64, req *AdminUpdateAPIKeyRequest) (*service.APIKey, error) {
	existing, err := h.apiKeys.GetByID(c.Request.Context(), keyID)
	if err != nil {
		return nil, err
	}

	svcReq := service.UpdateAPIKeyRequest{
		Name:   req.Name,
		Status: req.Status,
		// APIKeyService.Update overwrites IP lists unconditionally; pass through
		// the existing values so an admin field update never clears them.
		IPWhitelist: existing.IPWhitelist,
		IPBlacklist: existing.IPBlacklist,
		Quota:       req.Quota,
		ResetQuota:  req.ResetQuota,
		RateLimit5h: req.RateLimit5h,
		RateLimit1d: req.RateLimit1d,
		RateLimit7d: req.RateLimit7d,
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			svcReq.ClearExpiration = true
		} else {
			t, parseErr := time.Parse(time.RFC3339, *req.ExpiresAt)
			if parseErr != nil {
				response.BadRequest(c, "Invalid expires_at format: "+parseErr.Error())
				return nil, nil
			}
			svcReq.ExpiresAt = &t
		}
	}

	return h.apiKeys.Update(c.Request.Context(), keyID, existing.UserID, svcReq)
}

// Delete removes an API key regardless of its owner.
// DELETE /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) Delete(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}

	existing, err := h.apiKeys.GetByID(c.Request.Context(), keyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if err := h.apiKeys.Delete(c.Request.Context(), keyID, existing.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "API key deleted successfully"})
}
