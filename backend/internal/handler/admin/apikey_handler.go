package admin

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// apiKeyManager captures the existing APIKeyService invariants needed by admin operations.
type apiKeyManager interface {
	Create(ctx context.Context, userID int64, req service.CreateAPIKeyRequest) (*service.APIKey, error)
	CreateQCanvasKeyPair(ctx context.Context, userID int64, req service.CreateQCanvasKeyPairRequest) (*service.QCanvasKeyPair, error)
	GetByID(ctx context.Context, id int64) (*service.APIKey, error)
	Update(ctx context.Context, id, userID int64, req service.UpdateAPIKeyRequest) (*service.APIKey, error)
	Delete(ctx context.Context, id, userID int64) error
}

type AdminCreateQCanvasKeyPairRequest struct {
	VideoGroupID     int64 `json:"video_group_id"`
	MediaGroupID     int64 `json:"media_group_id"`
	AllowAdminTarget bool  `json:"allow_admin_target"`
}

type adminQCanvasKeyPairResponse struct {
	Video *dto.APIKey `json:"video"`
	Media *dto.APIKey `json:"media"`
}

// CreateQCanvasKeyPair issues the two logical QCanvas credentials for one
// existing user. The service, not this handler, owns the atomic DB mutation.
// POST /api/v1/admin/users/:id/qcanvas-key-pair
func (h *AdminAPIKeyHandler) CreateQCanvasKeyPair(c *gin.Context) {
	userID, err := parsePositiveAdminID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	if h.apiKeys == nil {
		response.InternalError(c, "API key service not available")
		return
	}
	var req AdminCreateQCanvasKeyPairRequest
	if err := decodeAdminAPIKeyJSON(c, &req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.VideoGroupID <= 0 || req.MediaGroupID <= 0 {
		response.BadRequest(c, "video_group_id and media_group_id must be positive")
		return
	}
	idempotencyPayload := struct {
		UserID int64                            `json:"user_id"`
		Body   AdminCreateQCanvasKeyPairRequest `json:"body"`
	}{UserID: userID, Body: req}
	executeAdminIdempotentJSONWithStoredResponseSanitizer(c, "admin.users.qcanvas_key_pair.create", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), sanitizeQCanvasKeyPairForReplay, func(ctx context.Context) (any, error) {
		pair, createErr := h.apiKeys.CreateQCanvasKeyPair(ctx, userID, service.CreateQCanvasKeyPairRequest{
			VideoGroupID:     req.VideoGroupID,
			MediaGroupID:     req.MediaGroupID,
			AllowAdminTarget: req.AllowAdminTarget,
		})
		if createErr != nil {
			return nil, createErr
		}
		return &adminQCanvasKeyPairResponse{Video: dto.APIKeyFromService(pair.Video), Media: dto.APIKeyFromService(pair.Media)}, nil
	})
}

func sanitizeQCanvasKeyPairForReplay(data any) any {
	pair, ok := data.(*adminQCanvasKeyPairResponse)
	if !ok || pair == nil {
		return data
	}
	sanitized := *pair
	if pair.Video != nil {
		video := *pair.Video
		video.Key = ""
		sanitized.Video = &video
	}
	if pair.Media != nil {
		media := *pair.Media
		media.Key = ""
		sanitized.Media = &media
	}
	return &sanitized
}

// AdminAPIKeyHandler handles admin API key management.
type AdminAPIKeyHandler struct {
	adminService service.AdminService
	apiKeys      apiKeyManager
}

// NewAdminAPIKeyHandler creates a new admin API key handler.
func NewAdminAPIKeyHandler(adminService service.AdminService, apiKeyService *service.APIKeyService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{adminService: adminService, apiKeys: apiKeyService}
}

type AdminCreateAPIKeyRequest struct {
	Name          string   `json:"name"`
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

// CreateForUser issues a new API key for a member. The full key is returned only here.
// POST /api/v1/admin/users/:id/api-keys
func (h *AdminAPIKeyHandler) CreateForUser(c *gin.Context) {
	userID, err := parsePositiveAdminID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	if h.apiKeys == nil {
		response.InternalError(c, "API key service not available")
		return
	}

	var req AdminCreateAPIKeyRequest
	if err := decodeAdminAPIKeyJSON(c, &req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := validateAdminCreateAPIKeyRequest(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	serviceRequest := service.CreateAPIKeyRequest{
		Name:          strings.TrimSpace(req.Name),
		GroupID:       req.GroupID,
		CustomKey:     req.CustomKey,
		IPWhitelist:   req.IPWhitelist,
		IPBlacklist:   req.IPBlacklist,
		ExpiresInDays: req.ExpiresInDays,
	}
	if req.Quota != nil {
		serviceRequest.Quota = *req.Quota
	}
	if req.RateLimit5h != nil {
		serviceRequest.RateLimit5h = *req.RateLimit5h
	}
	if req.RateLimit1d != nil {
		serviceRequest.RateLimit1d = *req.RateLimit1d
	}
	if req.RateLimit7d != nil {
		serviceRequest.RateLimit7d = *req.RateLimit7d
	}

	idempotencyPayload := struct {
		UserID int64                    `json:"user_id"`
		Body   AdminCreateAPIKeyRequest `json:"body"`
	}{UserID: userID, Body: req}
	executeAdminIdempotentJSONWithStoredResponseSanitizer(c, "admin.users.api_keys.create", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), sanitizeCreatedAPIKeyForReplay, func(ctx context.Context) (any, error) {
		key, createErr := h.apiKeys.Create(ctx, userID, serviceRequest)
		if createErr != nil {
			return nil, createErr
		}
		return dto.APIKeyFromService(key), nil
	})
}

func sanitizeCreatedAPIKeyForReplay(data any) any {
	key, ok := data.(*dto.APIKey)
	if !ok || key == nil {
		return data
	}
	sanitized := *key
	sanitized.Key = ""
	return &sanitized
}

type AdminUpdateAPIKeyRequest struct {
	GroupID             *int64   `json:"group_id"`
	ResetRateLimitUsage *bool    `json:"reset_rate_limit_usage"`
	Name                *string  `json:"name"`
	Status              *string  `json:"status"`
	Quota               *float64 `json:"quota"`
	ResetQuota          *bool    `json:"reset_quota"`
	ExpiresAt           *string  `json:"expires_at"`
	RateLimit5h         *float64 `json:"rate_limit_5h"`
	RateLimit1d         *float64 `json:"rate_limit_1d"`
	RateLimit7d         *float64 `json:"rate_limit_7d"`
}

func (r *AdminUpdateAPIKeyRequest) hasFieldUpdates() bool {
	return r.Name != nil || r.Status != nil || r.Quota != nil || r.ResetQuota != nil ||
		r.ExpiresAt != nil || r.RateLimit5h != nil || r.RateLimit1d != nil || r.RateLimit7d != nil
}

type adminAPIKeyMutationKind int

const (
	adminAPIKeyMutationFields adminAPIKeyMutationKind = iota + 1
	adminAPIKeyMutationGroup
	adminAPIKeyMutationRateLimitReset
)

func (r *AdminUpdateAPIKeyRequest) mutationKind() (adminAPIKeyMutationKind, error) {
	fieldMutation := r.hasFieldUpdates()
	groupMutation := r.GroupID != nil
	rateLimitResetMutation := r.ResetRateLimitUsage != nil && *r.ResetRateLimitUsage

	count := 0
	for _, selected := range []bool{fieldMutation, groupMutation, rateLimitResetMutation} {
		if selected {
			count++
		}
	}
	if count != 1 {
		return 0, errAdminAPIKeyValidation("request must select exactly one mutation category: fields, group, or rate-limit reset")
	}
	if fieldMutation {
		return adminAPIKeyMutationFields, nil
	}
	if groupMutation {
		return adminAPIKeyMutationGroup, nil
	}
	return adminAPIKeyMutationRateLimitReset, nil
}

// UpdateGroup handles all administrator-managed API key fields.
// PUT /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	keyID, err := parsePositiveAdminID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	if h.apiKeys == nil {
		response.InternalError(c, "API key service not available")
		return
	}

	var req AdminUpdateAPIKeyRequest
	if err := decodeAdminAPIKeyJSON(c, &req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := validateAdminUpdateAPIKeyRequest(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	mutationKind, err := req.mutationKind()
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result := &service.AdminUpdateAPIKeyGroupIDResult{}
	switch mutationKind {
	case adminAPIKeyMutationFields:
		result.APIKey, err = h.applyFieldUpdates(c.Request.Context(), keyID, &req)
	case adminAPIKeyMutationGroup:
		result, err = h.adminService.AdminUpdateAPIKeyGroupID(c.Request.Context(), keyID, req.GroupID)
	case adminAPIKeyMutationRateLimitReset:
		result.APIKey, err = h.adminService.AdminResetAPIKeyRateLimitUsage(c.Request.Context(), keyID)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, struct {
		APIKey                 *dto.APIKey `json:"api_key"`
		AutoGrantedGroupAccess bool        `json:"auto_granted_group_access"`
		GrantedGroupID         *int64      `json:"granted_group_id,omitempty"`
		GrantedGroupName       string      `json:"granted_group_name,omitempty"`
	}{
		APIKey:                 apiKeyDTOWithoutSecret(result.APIKey),
		AutoGrantedGroupAccess: result.AutoGrantedGroupAccess,
		GrantedGroupID:         result.GrantedGroupID,
		GrantedGroupName:       result.GrantedGroupName,
	})
}

func (h *AdminAPIKeyHandler) applyFieldUpdates(ctx context.Context, keyID int64, req *AdminUpdateAPIKeyRequest) (*service.APIKey, error) {
	existing, err := h.apiKeys.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	serviceRequest := service.UpdateAPIKeyRequest{
		Name:                req.Name,
		Status:              req.Status,
		IPWhitelist:         append([]string(nil), existing.IPWhitelist...),
		IPBlacklist:         append([]string(nil), existing.IPBlacklist...),
		Quota:               req.Quota,
		ResetQuota:          req.ResetQuota,
		RateLimit5h:         req.RateLimit5h,
		RateLimit1d:         req.RateLimit1d,
		RateLimit7d:         req.RateLimit7d,
		ResetRateLimitUsage: nil,
	}
	if req.ExpiresAt != nil {
		if strings.TrimSpace(*req.ExpiresAt) == "" {
			serviceRequest.ClearExpiration = true
		} else {
			expiresAt, parseErr := time.Parse(time.RFC3339, *req.ExpiresAt)
			if parseErr != nil {
				return nil, parseErr
			}
			serviceRequest.ExpiresAt = &expiresAt
		}
	}
	return h.apiKeys.Update(ctx, keyID, existing.UserID, serviceRequest)
}

// Reveal returns the full plaintext API key for an administrator and writes an audit log.
// GET /api/v1/admin/api-keys/:id/reveal
func (h *AdminAPIKeyHandler) Reveal(c *gin.Context) {
	keyID, err := parsePositiveAdminID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	if h.apiKeys == nil {
		response.InternalError(c, "API key service not available")
		return
	}
	existing, err := h.apiKeys.GetByID(c.Request.Context(), keyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := dto.APIKeyFromService(existing)
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	role, _ := middleware.GetUserRoleFromContext(c)
	slog.Info("admin.api_key.reveal",
		"audit", true,
		"admin_user_id", subject.UserID,
		"role", role,
		"api_key_id", existing.ID,
		"target_user_id", existing.UserID,
		"client_ip", c.ClientIP(),
	)
	response.Success(c, out)
}

// Delete resolves the true owner before delegating to the audited service deletion path.
// DELETE /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) Delete(c *gin.Context) {
	keyID, err := parsePositiveAdminID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	if h.apiKeys == nil {
		response.InternalError(c, "API key service not available")
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

func decodeAdminAPIKeyJSON(c *gin.Context, target any) error {
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureJSONEOF(decoder)
}

func validateAdminCreateAPIKeyRequest(req *AdminCreateAPIKeyRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errAdminAPIKeyValidation("name is required")
	}
	if req.GroupID != nil && *req.GroupID <= 0 {
		return errAdminAPIKeyValidation("group_id must be positive")
	}
	if req.ExpiresInDays != nil && *req.ExpiresInDays <= 0 {
		return errAdminAPIKeyValidation("expires_in_days must be positive")
	}
	return validateAdminAPIKeyAmounts(req.Quota, req.RateLimit5h, req.RateLimit1d, req.RateLimit7d)
}

func validateAdminUpdateAPIKeyRequest(req *AdminUpdateAPIKeyRequest) error {
	if req.Status != nil && *req.Status != service.StatusAPIKeyActive && *req.Status != service.StatusAPIKeyDisabled {
		return errAdminAPIKeyValidation("status must be active or disabled")
	}
	if req.GroupID != nil && *req.GroupID < 0 {
		return errAdminAPIKeyValidation("group_id must be non-negative")
	}
	if err := validateAdminAPIKeyAmounts(req.Quota, req.RateLimit5h, req.RateLimit1d, req.RateLimit7d); err != nil {
		return err
	}
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		if _, err := time.Parse(time.RFC3339, *req.ExpiresAt); err != nil {
			return errAdminAPIKeyValidation("expires_at must be RFC3339")
		}
	}
	return nil
}

func validateAdminAPIKeyAmounts(values ...*float64) error {
	for _, value := range values {
		if value != nil && (*value < 0 || math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return errAdminAPIKeyValidation("quota and rate limits must be finite non-negative numbers")
		}
	}
	return nil
}

type adminAPIKeyValidationError string

func (e adminAPIKeyValidationError) Error() string  { return string(e) }
func errAdminAPIKeyValidation(message string) error { return adminAPIKeyValidationError(message) }

func parsePositiveAdminID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errAdminAPIKeyValidation("id must be positive")
	}
	return id, nil
}

func apiKeyDTOWithoutSecret(key *service.APIKey) *dto.APIKey {
	out := dto.APIKeyFromService(key)
	if out != nil {
		out.KeyHint = apiKeyHint(out.Key)
		out.Key = ""
	}
	return out
}

func apiKeyHint(full string) string {
	trimmed := strings.TrimSpace(full)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return trimmed
	}
	return trimmed[len(trimmed)-4:]
}
