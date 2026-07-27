package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ownedVideoLocalAssetOpener interface {
	OpenOwnedLocalAsset(context.Context, int64, int64) (*service.VideoLocalAsset, error)
}

type VideoGatewayHandler struct {
	service     *service.VideoGatewayService
	localAssets ownedVideoLocalAssetOpener
}

func NewVideoGatewayHandler(s *service.VideoGatewayService) *VideoGatewayHandler {
	return &VideoGatewayHandler{service: s, localAssets: s}
}

type videoCreateBody struct {
	ProviderAccountID int64                      `json:"provider_account_id" binding:"required"`
	Model             string                     `json:"model"`
	Prompt            string                     `json:"prompt"`
	Content           []service.VideoContentItem `json:"content"`
	Ratio             string                     `json:"ratio"`
	GenerateAudio     bool                       `json:"generate_audio"`
	ReturnLastFrame   bool                       `json:"return_last_frame"`
	Watermark         bool                       `json:"watermark"`
	Duration          int                        `json:"duration"`
	Resolution        string                     `json:"resolution"`
	CreationKey       string                     `json:"creation_key"`
}

func videoScope(c *gin.Context) (service.VideoTaskScope, bool) {
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key == nil || key.User == nil || key.GroupID == nil || *key.GroupID <= 0 {
		response.Error(c, http.StatusForbidden, "complete employee API-key scope is required")
		return service.VideoTaskScope{}, false
	}
	return service.VideoTaskScope{UserID: key.User.ID, APIKeyID: key.ID, GroupID: *key.GroupID}, true
}

func (h *VideoGatewayHandler) Providers(c *gin.Context) {
	scope, ok := videoScope(c)
	if !ok {
		return
	}
	items, err := h.service.ListProviders(c.Request.Context(), scope)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list video providers")
		return
	}
	response.Success(c, items)
}

func (h *VideoGatewayHandler) Create(c *gin.Context) {
	scope, ok := videoScope(c)
	if !ok {
		return
	}
	var body videoCreateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid video task request")
		return
	}
	task, err := h.service.CreateTask(c.Request.Context(), service.VideoTaskCreateCommand{Scope: scope, ProviderAccountID: body.ProviderAccountID,
		Model: body.Model, Prompt: body.Prompt, Content: body.Content, Ratio: body.Ratio, GenerateAudio: body.GenerateAudio,
		ReturnLastFrame: body.ReturnLastFrame, Watermark: body.Watermark, Duration: body.Duration, Resolution: body.Resolution,
		CreationKey: body.CreationKey})
	if err != nil {
		writeVideoError(c, err)
		return
	}
	response.Accepted(c, videoTaskResponse(task))
}

func (h *VideoGatewayHandler) Get(c *gin.Context)    { h.withTask(c, false) }
func (h *VideoGatewayHandler) Cancel(c *gin.Context) { h.withTask(c, true) }

func (h *VideoGatewayHandler) LocalAsset(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid video task id")
		return
	}
	if h == nil || h.localAssets == nil {
		response.Error(c, http.StatusNotFound, "video local asset not found")
		return
	}
	asset, err := h.localAssets.OpenOwnedLocalAsset(c.Request.Context(), id, subject.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrVideoTaskForbidden):
			response.Error(c, http.StatusForbidden, "video task is outside employee scope")
		case errors.Is(err, service.ErrVideoTaskNotFound), errors.Is(err, service.ErrVideoLocalAssetNotFound):
			response.Error(c, http.StatusNotFound, "video local asset not found")
		default:
			response.InternalError(c, "failed to read local video asset")
		}
		return
	}
	defer asset.File.Close()
	serveVideoLocalAsset(c, id, asset)
}

func serveVideoLocalAsset(c *gin.Context, taskID int64, asset *service.VideoLocalAsset) {
	filename := fmt.Sprintf("video-task-%d.mp4", taskID)
	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	c.Header("Accept-Ranges", "bytes")
	http.ServeContent(c.Writer, c.Request, filename, asset.ModTime, asset.File)
}

func (h *VideoGatewayHandler) withTask(c *gin.Context, cancel bool) {
	scope, ok := videoScope(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid video task id")
		return
	}
	var task *service.VideoTask
	if cancel {
		task, err = h.service.CancelTask(c.Request.Context(), id, scope)
	} else {
		task, err = h.service.GetTask(c.Request.Context(), id, scope)
	}
	if err != nil {
		writeVideoError(c, err)
		return
	}
	payload := videoTaskResponse(task)
	if cancel {
		payload["cancel_outcome"] = "local_pre_dispatch"
	}
	response.Success(c, payload)
}

func writeVideoError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVideoTaskNotFound):
		response.Error(c, http.StatusNotFound, "video task not found")
	case errors.Is(err, service.ErrVideoProviderNotFound):
		response.Error(c, http.StatusNotFound, "video provider not found")
	case errors.Is(err, service.ErrVideoRealDispatchDenied), errors.Is(err, service.ErrVideoRealDispatchConsumed), errors.Is(err, service.ErrVideoBudgetRejected):
		response.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrVideoCancelConflict):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		response.BadRequest(c, "video request failed")
	}
}

func videoTaskResponse(t *service.VideoTask) gin.H {
	localAssetAvailable := t.Status == service.VideoStatusSucceeded && t.ResultURL != "" && t.LocalAssetPath != "" && t.LocalAssetSavedAt != nil
	var localAssetURL any
	if localAssetAvailable {
		localAssetURL = fmt.Sprintf("/api/v1/video/tasks/%d/local-asset", t.ID)
	}
	return gin.H{"id": t.ID, "provider": t.Provider, "model": t.Model, "status": t.Status, "upstream_task_id": t.UpstreamTaskID,
		"result_url": t.ResultURL, "last_frame_url": t.LastFrameURL, "duration": t.DurationSeconds, "resolution": t.Resolution,
		"usage_total_tokens": t.UsageTotalTokens, "cost": t.CostAmount, "currency": t.Currency, "real_dispatch_count": t.RealDispatchCount,
		"pricing_source": nullableVideoPricingText(t.PricingSource), "pricing_version": nullableVideoPricingText(t.PricingVersion),
		"pricing_cny_per_million_completion_tokens": t.PricingCNYPerMillionCompletionTokens,
		"pricing_usd_cny_exchange_rate":             t.PricingUSDCNYExchangeRate, "pricing_maximum_cny": t.PricingMaximumCNY,
		"provider_error_code": t.ProviderErrorCode, "provider_error_message": t.ProviderErrorMessage, "error": t.ErrorMessage,
		"local_asset_available": localAssetAvailable, "local_asset_download_url": localAssetURL, "local_asset_saved_at": t.LocalAssetSavedAt}
}

func nullableVideoPricingText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
