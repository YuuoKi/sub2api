package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type VideoHandler struct{ service *service.VideoAdminService }

func NewVideoHandler(s *service.VideoAdminService) *VideoHandler { return &VideoHandler{service: s} }

type videoProviderRequest struct {
	GroupID      *int64  `json:"group_id"`
	Provider     string  `json:"provider"`
	DisplayName  *string `json:"display_name"`
	Enabled      *bool   `json:"enabled"`
	APIKey       *string `json:"api_key"`
	BaseURL      *string `json:"base_url"`
	DefaultModel *string `json:"default_model"`
}

func (h *VideoHandler) ListProviders(c *gin.Context) {
	items, err := h.service.ListProviders(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"items": items})
}
func (h *VideoHandler) CreateProvider(c *gin.Context) {
	var req videoProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid provider request")
		return
	}
	if req.GroupID == nil || req.DisplayName == nil || req.APIKey == nil {
		response.BadRequest(c, "group_id, display_name and api_key are required")
		return
	}
	enabled := false
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item, err := h.service.CreateProvider(c.Request.Context(), service.VideoProviderAdminCreate{GroupID: *req.GroupID, Provider: req.Provider, DisplayName: *req.DisplayName, APIKey: *req.APIKey, BaseURL: value(req.BaseURL), DefaultModel: value(req.DefaultModel), Enabled: enabled})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}
func (h *VideoHandler) UpdateProvider(c *gin.Context) {
	id, ok := videoID(c)
	if !ok {
		return
	}
	var req videoProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid provider request")
		return
	}
	item, err := h.service.UpdateProvider(c.Request.Context(), id, service.VideoProviderAdminUpdate{GroupID: req.GroupID, DisplayName: req.DisplayName, APIKey: req.APIKey, BaseURL: req.BaseURL, DefaultModel: req.DefaultModel, Enabled: req.Enabled})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}
func (h *VideoHandler) AuthorizeTinyReal(c *gin.Context) {
	id, ok := videoID(c)
	if !ok {
		return
	}
	var req struct {
		Confirmation string `json:"confirmation"`
	}
	if c.ShouldBindJSON(&req) != nil || req.Confirmation != "tiny_real" {
		response.BadRequest(c, "confirmation must equal tiny_real")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	item, err := h.service.AuthorizeTinyReal(c.Request.Context(), id, subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}
func (h *VideoHandler) ListTasks(c *gin.Context) {
	page, size := response.ParsePagination(c)
	items, total, err := h.service.ListTasks(c.Request.Context(), service.VideoAdminTaskFilter{Page: page, PageSize: size, Status: c.Query("status")})
	if response.ErrorFrom(c, err) {
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, videoTaskAdminResponse(&items[i]))
	}
	response.Paginated(c, out, total, page, size)
}
func (h *VideoHandler) GetTask(c *gin.Context) {
	id, ok := videoID(c)
	if !ok {
		return
	}
	item, err := h.service.GetTask(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, videoTaskAdminResponse(item))
}
func (h *VideoHandler) SystemCheck(c *gin.Context) {
	item, err := h.service.SystemCheck(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}
func videoID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid video id")
		return 0, false
	}
	return id, true
}
func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func videoTaskAdminResponse(t *service.VideoTask) gin.H {
	if t == nil {
		return gin.H{}
	}
	return gin.H{"id": t.ID, "provider_account_id": t.ProviderAccountID, "provider": t.Provider, "model": t.Model, "task_type": t.TaskType, "prompt": t.Prompt, "status": t.Status, "upstream_task_id": t.UpstreamTaskID, "result_url": t.ResultURL, "last_frame_url": t.LastFrameURL, "error_message": t.ErrorMessage, "provider_error_code": t.ProviderErrorCode, "provider_error_message": t.ProviderErrorMessage, "cost_amount": t.CostAmount, "provider_actual_cost_usd": t.ProviderActualCostUSD, "currency": t.Currency, "real_dispatch_count": t.RealDispatchCount, "dispatch_state": t.DispatchState, "created_by": t.CreatedBy, "created_at": t.CreatedAt, "updated_at": t.UpdatedAt, "completed_at": t.CompletedAt}
}
