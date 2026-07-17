package admin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	service                   *service.VideoAdminService
	handoff                   service.AssetHandoffManager
	localAssets               adminVideoLocalAssetOpener
	trustDockerLoopbackBridge bool
}

type adminVideoLocalAssetOpener interface {
	OpenLocalAsset(context.Context, int64) (*service.VideoLocalAsset, error)
}

func NewVideoHandler(s *service.VideoAdminService) *VideoHandler {
	return &VideoHandler{
		service:                   s,
		handoff:                   s,
		localAssets:               s,
		trustDockerLoopbackBridge: strings.EqualFold(strings.TrimSpace(os.Getenv("ASSET_HANDOFF_TRUST_DOCKER_LOOPBACK_BRIDGE")), "true"),
	}
}

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
func (h *VideoHandler) Contract(c *gin.Context) {
	response.Success(c, gin.H{"provider": "seedance", "base_url": service.SeedanceBaseURL, "default_model": service.SeedanceModel, "duration_seconds": 4, "resolution": "720p"})
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
	if writeVideoAdminError(c, err) {
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
	if writeVideoAdminError(c, err) {
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
	if writeVideoAdminError(c, err) {
		return
	}
	response.Success(c, item)
}
func (h *VideoHandler) ListTasks(c *gin.Context) {
	page, size := response.ParsePagination(c)
	items, total, err := h.service.ListTasks(c.Request.Context(), service.VideoAdminTaskFilter{Page: page, PageSize: size, Status: c.Query("status")})
	if writeVideoAdminError(c, err) {
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
	if writeVideoAdminError(c, err) {
		return
	}
	response.Success(c, videoTaskAdminResponse(item))
}
func (h *VideoHandler) LocalAsset(c *gin.Context) {
	id, ok := videoID(c)
	if !ok {
		return
	}
	if h == nil || h.localAssets == nil {
		response.Error(c, http.StatusNotFound, "video local asset not found")
		return
	}
	asset, err := h.localAssets.OpenLocalAsset(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrVideoTaskNotFound) || errors.Is(err, service.ErrVideoLocalAssetNotFound) {
			response.Error(c, http.StatusNotFound, "video local asset not found")
		} else {
			response.InternalError(c, "failed to read local video asset")
		}
		return
	}
	defer asset.File.Close()
	filename := fmt.Sprintf("video-task-%d.mp4", id)
	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	c.Header("Accept-Ranges", "bytes")
	http.ServeContent(c.Writer, c.Request, filename, asset.ModTime, asset.File)
}
func (h *VideoHandler) CreateAssetHandoff(c *gin.Context) {
	id, ok := videoID(c)
	if !ok {
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req struct {
		AssetKind service.AssetHandoffKind `json:"asset_kind"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid asset handoff request")
		return
	}
	if h.handoff == nil {
		response.InternalError(c, "Asset handoff is not available")
		return
	}
	issued, err := h.handoff.Issue(c.Request.Context(), subject.UserID, id, req.AssetKind)
	if writeAssetHandoffError(c, err) {
		return
	}
	response.Success(c, issued)
}
func (h *VideoHandler) ConsumeAssetHandoff(c *gin.Context) {
	if !isTrustedAssetHandoffRequest(c.Request, h.trustDockerLoopbackBridge) {
		response.Error(c, http.StatusForbidden, "asset handoff consumption is loopback-only")
		return
	}
	if h.handoff == nil {
		response.InternalError(c, "Asset handoff is not available")
		return
	}
	var req struct {
		Ticket string `json:"ticket"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Ticket) == "" {
		response.BadRequest(c, "asset handoff ticket is required")
		return
	}
	asset, err := h.handoff.Consume(c.Request.Context(), req.Ticket)
	if writeAssetHandoffError(c, err) {
		return
	}
	response.Success(c, asset)
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
func writeVideoAdminError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrVideoAdminInvalidRequest), errors.Is(err, service.ErrVideoAdminInvalidGroup):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrVideoProviderNotFound), errors.Is(err, service.ErrVideoTaskNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrVideoAdminConflict), errors.Is(err, service.ErrVideoAdminAuthorizationConflict):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		response.ErrorFrom(c, err)
	}
	return true
}
func writeAssetHandoffError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrAssetHandoffNotFound), errors.Is(err, service.ErrVideoTaskNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrAssetHandoffExpired), errors.Is(err, service.ErrAssetHandoffConsumed):
		response.Error(c, http.StatusGone, err.Error())
	case errors.Is(err, service.ErrAssetHandoffInvalidIssuer), errors.Is(err, service.ErrAssetHandoffInvalidKind):
		response.BadRequest(c, err.Error())
	case errors.Is(err, service.ErrAssetHandoffTaskNotSucceeded), errors.Is(err, service.ErrAssetHandoffAssetMissing),
		errors.Is(err, service.ErrAssetHandoffInvalidMIME), errors.Is(err, service.ErrAssetHandoffTooLarge),
		errors.Is(err, service.ErrAssetHandoffUnverifiable):
		response.Error(c, http.StatusUnprocessableEntity, err.Error())
	default:
		response.ErrorFrom(c, err)
	}
	return true
}
func isRealLoopbackRemote(remoteAddr string) bool {
	ip := parseRemoteIP(remoteAddr)
	return ip != nil && ip.IsLoopback()
}
func parseRemoteIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.Trim(strings.TrimSpace(remoteAddr), "[]")
	}
	ip := net.ParseIP(host)
	return ip
}
func isTrustedAssetHandoffRequest(request *http.Request, trustDockerLoopbackBridge bool) bool {
	if request == nil {
		return false
	}
	remoteIP := parseRemoteIP(request.RemoteAddr)
	if remoteIP == nil {
		return false
	}
	if remoteIP.IsLoopback() {
		return true
	}
	if !trustDockerLoopbackBridge || !remoteIP.IsPrivate() {
		return false
	}
	host, port, err := net.SplitHostPort(strings.TrimSpace(request.Host))
	if err != nil || port != "8080" {
		return false
	}
	hostIP := net.ParseIP(strings.Trim(host, "[]"))
	return hostIP != nil && hostIP.IsLoopback()
}
func videoTaskAdminResponse(t *service.VideoTask) gin.H {
	if t == nil {
		return gin.H{}
	}
	localAssetAvailable := t.Status == service.VideoStatusSucceeded && t.ResultURL != "" && t.LocalAssetPath != "" && t.LocalAssetSavedAt != nil
	var localAssetURL any
	if localAssetAvailable {
		localAssetURL = fmt.Sprintf("/api/v1/admin/video/tasks/%d/local-asset", t.ID)
	}
	return gin.H{
		"id": t.ID, "api_key_id": t.APIKeyID, "group_id": t.GroupID, "provider_account_id": t.ProviderAccountID,
		"provider": t.Provider, "model": t.Model, "task_type": t.TaskType, "prompt": t.Prompt, "status": t.Status,
		"request_model": t.Model, "request_duration_seconds": t.DurationSeconds, "request_resolution": t.Resolution,
		"upstream_model": t.UpstreamModel, "upstream_duration_seconds": t.UpstreamDurationSeconds,
		"upstream_resolution": t.UpstreamResolution, "billing_model": t.BillingModel,
		"billing_duration_seconds": t.BillingDurationSeconds, "billing_resolution": t.BillingResolution,
		"upstream_task_id": t.UpstreamTaskID, "result_url": t.ResultURL, "last_frame_url": t.LastFrameURL,
		"duration_seconds": t.DurationSeconds, "resolution": t.Resolution, "usage_total_tokens": t.UsageTotalTokens,
		"error_message": t.ErrorMessage, "provider_error_code": t.ProviderErrorCode, "provider_error_message": t.ProviderErrorMessage,
		"reserved_cost_usd": t.ReservedCostUSD, "reservation_state": t.ReservationState, "reserved_at": t.ReservedAt,
		"reservation_window_5h_start": t.ReservationWindow5h, "reservation_window_1d_start": t.ReservationWindow1d,
		"reservation_window_7d_start": t.ReservationWindow7d, "cost_amount": t.CostAmount,
		"provider_actual_cost_usd": t.ProviderActualCostUSD, "currency": t.Currency,
		"pricing_source": nullableVideoPricingText(t.PricingSource), "pricing_version": nullableVideoPricingText(t.PricingVersion),
		"pricing_cny_per_million_completion_tokens": t.PricingCNYPerMillionCompletionTokens,
		"pricing_usd_cny_exchange_rate":             t.PricingUSDCNYExchangeRate, "pricing_maximum_cny": t.PricingMaximumCNY,
		"balance_before_usd": t.BalanceBeforeUSD, "balance_after_usd": t.BalanceAfterUSD,
		"balance_delta_usd": t.BalanceDeltaUSD, "authorization_consumed_at": t.AuthorizationConsumedAt,
		"authorization_consumed_by": t.AuthorizationConsumedBy,
		"real_dispatch_count":       t.RealDispatchCount, "dispatch_state": t.DispatchState, "created_by": t.CreatedBy,
		"created_at": t.CreatedAt, "updated_at": t.UpdatedAt, "completed_at": t.CompletedAt,
		"local_asset_available": localAssetAvailable, "local_asset_download_url": localAssetURL,
		"local_asset_saved_at": t.LocalAssetSavedAt,
	}
}

func nullableVideoPricingText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
