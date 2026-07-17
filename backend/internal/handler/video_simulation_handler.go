package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type simulationResultOpener interface {
	OpenSimulationResult(ctx context.Context, taskID, userID int64) (*service.VideoSimulationResult, error)
}

type simulationAdminResultOpener interface {
	OpenSimulationResultAsAdmin(ctx context.Context, taskID int64) (*service.VideoSimulationResult, error)
}

type videoSimulationTaskService interface {
	SimulationContract() map[string]any
	CreateTask(ctx context.Context, cmd service.VideoSimulationCreateCommand) (*service.VideoTask, error)
	GetTask(ctx context.Context, taskID, userID int64) (*service.VideoTask, error)
	ListTasks(ctx context.Context, userID int64) ([]*service.VideoTask, error)
	CancelTask(ctx context.Context, taskID, userID int64) (*service.VideoTask, error)
	OpenSimulationResult(ctx context.Context, taskID, userID int64) (*service.VideoSimulationResult, error)
}

// VideoSimulationHandler serves JWT employee simulation routes.
type VideoSimulationHandler struct {
	service videoSimulationTaskService
	opener  simulationResultOpener
}

func NewVideoSimulationHandler(openerOrService any) *VideoSimulationHandler {
	h := &VideoSimulationHandler{}
	switch v := openerOrService.(type) {
	case videoSimulationTaskService:
		h.service = v
		h.opener = v
	case simulationResultOpener:
		h.opener = v
	}
	return h
}

type VideoSimulationAdminHandler struct {
	opener simulationAdminResultOpener
}

func NewVideoSimulationAdminHandler(opener simulationAdminResultOpener) *VideoSimulationAdminHandler {
	return &VideoSimulationAdminHandler{opener: opener}
}

type simulationCreateBody struct {
	APIKeyID    int64  `json:"api_key_id" binding:"required"`
	Prompt      string `json:"prompt" binding:"required"`
	CreationKey string `json:"creation_key"`
}

func (h *VideoSimulationHandler) Contract(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "video simulation unavailable")
		return
	}
	response.Success(c, h.service.SimulationContract())
}

func (h *VideoSimulationHandler) Create(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.service == nil {
		response.InternalError(c, "video simulation unavailable")
		return
	}
	var body simulationCreateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid simulation task request")
		return
	}
	task, err := h.service.CreateTask(c.Request.Context(), service.VideoSimulationCreateCommand{
		UserID: subject.UserID, APIKeyID: body.APIKeyID, Prompt: body.Prompt, CreationKey: body.CreationKey,
	})
	if err != nil {
		writeSimulationError(c, err)
		return
	}
	response.Accepted(c, simulationTaskResponse(task))
}

func (h *VideoSimulationHandler) List(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.service == nil {
		response.InternalError(c, "video simulation unavailable")
		return
	}
	tasks, err := h.service.ListTasks(c.Request.Context(), subject.UserID)
	if err != nil {
		writeSimulationError(c, err)
		return
	}
	items := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, simulationTaskResponse(task))
	}
	response.Success(c, gin.H{"items": items})
}

func (h *VideoSimulationHandler) Get(c *gin.Context) {
	h.withOwnedTask(c, false)
}

func (h *VideoSimulationHandler) Cancel(c *gin.Context) {
	h.withOwnedTask(c, true)
}

func (h *VideoSimulationHandler) withOwnedTask(c *gin.Context, cancel bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.service == nil {
		response.InternalError(c, "video simulation unavailable")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid video task id")
		return
	}
	var task *service.VideoTask
	if cancel {
		task, err = h.service.CancelTask(c.Request.Context(), id, subject.UserID)
	} else {
		task, err = h.service.GetTask(c.Request.Context(), id, subject.UserID)
	}
	if err != nil {
		writeSimulationError(c, err)
		return
	}
	response.Success(c, simulationTaskResponse(task))
}

func (h *VideoSimulationHandler) Result(c *gin.Context) {
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
	if h == nil || h.opener == nil {
		response.InternalError(c, "failed to open simulation result")
		return
	}
	result, err := h.opener.OpenSimulationResult(c.Request.Context(), id, subject.UserID)
	if err != nil {
		writeSimulationResultError(c, err)
		return
	}
	writeSimulationResult(c, result)
}

func (h *VideoSimulationAdminHandler) Result(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid video task id")
		return
	}
	if h == nil || h.opener == nil {
		response.InternalError(c, "failed to open simulation result")
		return
	}
	result, err := h.opener.OpenSimulationResultAsAdmin(c.Request.Context(), id)
	if err != nil {
		writeSimulationResultError(c, err)
		return
	}
	writeSimulationResult(c, result)
}

func writeSimulationResult(c *gin.Context, result *service.VideoSimulationResult) {
	if result == nil {
		response.InternalError(c, "failed to open simulation result")
		return
	}
	c.Header("Content-Type", result.ContentType)
	c.Header("X-Media-Kind", result.MediaKind)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	if c.Query("download") == "1" || strings.EqualFold(c.Query("download"), "true") {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", result.Filename))
	}
	c.Data(http.StatusOK, result.ContentType, result.Body)
}

func writeSimulationResultError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVideoTaskForbidden):
		response.Error(c, http.StatusForbidden, "video task is outside employee scope")
	case errors.Is(err, service.ErrVideoTaskNotFound):
		response.Error(c, http.StatusNotFound, "video task not found")
	case errors.Is(err, service.ErrVideoSimulationResultNotReady):
		response.Error(c, http.StatusConflict, "video simulation result is not ready")
	default:
		response.InternalError(c, "failed to open simulation result")
	}
}

func writeSimulationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAPIKeyNotFound):
		response.Error(c, http.StatusNotFound, "api key not found")
	case errors.Is(err, service.ErrVideoSimulationAPIKeyInactive):
		response.Error(c, http.StatusForbidden, "api key is inactive")
	case errors.Is(err, service.ErrVideoSimulationAPIKeyNotOwned):
		response.Error(c, http.StatusForbidden, "api key is not owned by caller")
	case errors.Is(err, service.ErrVideoTaskNotFound):
		response.Error(c, http.StatusNotFound, "video task not found")
	case errors.Is(err, service.ErrVideoTaskForbidden):
		response.Error(c, http.StatusForbidden, "video task is outside employee scope")
	case errors.Is(err, service.ErrVideoSimulationPromptTooLarge):
		response.BadRequest(c, err.Error())
	case errors.Is(err, service.ErrVideoCancelConflict),
		errors.Is(err, service.ErrVideoSimulationCreationKeyConflict),
		errors.Is(err, service.ErrVideoSimulationResultNotReady):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		response.InternalError(c, "video simulation request failed")
	}
}

func simulationTaskResponse(t *service.VideoTask) gin.H {
	if t == nil {
		return gin.H{}
	}
	return gin.H{
		"id": t.ID, "provider": t.Provider, "model": t.Model, "status": t.Status,
		"prompt": t.Prompt, "duration": t.DurationSeconds, "resolution": t.Resolution,
		"cost": t.CostAmount, "currency": t.Currency,
		"pricing_source": t.PricingSource, "pricing_version": t.PricingVersion,
		"error": t.ErrorMessage, "version": t.Version,
		"created_at": t.CreatedAt, "updated_at": t.UpdatedAt, "completed_at": t.CompletedAt,
	}
}
