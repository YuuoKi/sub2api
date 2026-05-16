package handler

import (
	"strconv"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	video *service.VideoGatewayService
}

func NewVideoHandler(video *service.VideoGatewayService) *VideoHandler {
	return &VideoHandler{video: video}
}

type videoTaskCreateRequest struct {
	ProviderAccountID int64  `json:"provider_account_id" binding:"required,min=1"`
	TaskType          string `json:"task_type" binding:"required,oneof=text_to_video image_to_video reference_to_video"`
	Model             string `json:"model" binding:"omitempty,max=200"`
	Prompt            string `json:"prompt" binding:"required,max=8000"`
	NegativePrompt    string `json:"negative_prompt" binding:"omitempty,max=4000"`
	ReferenceImageURL string `json:"reference_image_url" binding:"omitempty,max=1000"`
	ReferenceVideoURL string `json:"reference_video_url" binding:"omitempty,max=1000"`
	AspectRatio       string `json:"aspect_ratio" binding:"omitempty,max=20"`
	Duration          int    `json:"duration" binding:"omitempty,min=1,max=60"`
	Resolution        string `json:"resolution" binding:"omitempty,max=20"`
}

type videoTaskResponse struct {
	ID                int64                    `json:"id"`
	ProviderAccountID int64                    `json:"provider_account_id"`
	Provider          string                   `json:"provider"`
	Model             string                   `json:"model"`
	TaskType          string                   `json:"task_type"`
	Prompt            string                   `json:"prompt"`
	NegativePrompt    string                   `json:"negative_prompt"`
	ReferenceImageURL string                   `json:"reference_image_url"`
	ReferenceVideoURL string                   `json:"reference_video_url"`
	AspectRatio       string                   `json:"aspect_ratio"`
	Duration          int                      `json:"duration"`
	Resolution        string                   `json:"resolution"`
	Status            string                   `json:"status"`
	UpstreamTaskID    string                   `json:"upstream_task_id"`
	ResultURL         string                   `json:"result_url"`
	ErrorMessage      string                   `json:"error_message"`
	CostEstimate      float64                  `json:"cost_estimate"`
	CreatedBy         int64                    `json:"created_by"`
	CreatedAt         string                   `json:"created_at"`
	UpdatedAt         string                   `json:"updated_at"`
	CompletedAt       *string                  `json:"completed_at"`
	Events            []videoTaskEventResponse `json:"events,omitempty"`
}

type videoTaskEventResponse struct {
	ID          int64          `json:"id"`
	VideoTaskID int64          `json:"video_task_id"`
	EventType   string         `json:"event_type"`
	Message     string         `json:"message"`
	Payload     map[string]any `json:"payload_json"`
	CreatedAt   string         `json:"created_at"`
}

func (h *VideoHandler) ListTasks(c *gin.Context) {
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	role, _ := middleware2.GetUserRoleFromContext(c)
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.video.ListTasks(c.Request.Context(), service.VideoTaskListParams{
		Page:      page,
		PageSize:  pageSize,
		Status:    c.Query("status"),
		Provider:  c.Query("provider"),
		CreatedBy: subject.UserID,
		IsAdmin:   role == "admin",
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]videoTaskResponse, 0, len(items))
	for _, item := range items {
		out = append(out, videoTaskToResponse(item, nil))
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *VideoHandler) CreateTask(c *gin.Context) {
	var req videoTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	task, err := h.video.CreateTask(c.Request.Context(), service.VideoTaskCreateParams{
		ProviderAccountID: req.ProviderAccountID,
		TaskType:          req.TaskType,
		Model:             req.Model,
		Prompt:            req.Prompt,
		NegativePrompt:    req.NegativePrompt,
		ReferenceImageURL: req.ReferenceImageURL,
		ReferenceVideoURL: req.ReferenceVideoURL,
		AspectRatio:       req.AspectRatio,
		Duration:          req.Duration,
		Resolution:        req.Resolution,
		CreatedBy:         subject.UserID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, videoTaskToResponse(task, nil))
}

func (h *VideoHandler) GetTask(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	role, _ := middleware2.GetUserRoleFromContext(c)
	task, events, err := h.video.GetTask(c.Request.Context(), id, subject.UserID, role == "admin")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, videoTaskToResponse(task, events))
}

func (h *VideoHandler) CancelTask(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	role, _ := middleware2.GetUserRoleFromContext(c)
	task, err := h.video.CancelTask(c.Request.Context(), id, subject.UserID, role == "admin")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, videoTaskToResponse(task, nil))
}

func videoTaskToResponse(task *service.VideoTask, events []*service.VideoTaskEvent) videoTaskResponse {
	if task == nil {
		return videoTaskResponse{}
	}
	var completed *string
	if task.CompletedAt != nil {
		v := formatTaskTime(*task.CompletedAt)
		completed = &v
	}
	eventResponses := make([]videoTaskEventResponse, 0, len(events))
	for _, event := range events {
		eventResponses = append(eventResponses, videoTaskEventResponse{
			ID:          event.ID,
			VideoTaskID: event.VideoTaskID,
			EventType:   event.EventType,
			Message:     event.Message,
			Payload:     event.Payload,
			CreatedAt:   formatTaskTime(event.CreatedAt),
		})
	}
	return videoTaskResponse{
		ID:                task.ID,
		ProviderAccountID: task.ProviderAccountID,
		Provider:          task.Provider,
		Model:             task.Model,
		TaskType:          task.TaskType,
		Prompt:            task.Prompt,
		NegativePrompt:    task.NegativePrompt,
		ReferenceImageURL: task.ReferenceImageURL,
		ReferenceVideoURL: task.ReferenceVideoURL,
		AspectRatio:       task.AspectRatio,
		Duration:          task.Duration,
		Resolution:        task.Resolution,
		Status:            task.Status,
		UpstreamTaskID:    task.UpstreamTaskID,
		ResultURL:         task.ResultURL,
		ErrorMessage:      task.ErrorMessage,
		CostEstimate:      task.CostEstimate,
		CreatedBy:         task.CreatedBy,
		CreatedAt:         formatTaskTime(task.CreatedAt),
		UpdatedAt:         formatTaskTime(task.UpdatedAt),
		CompletedAt:       completed,
		Events:            eventResponses,
	}
}

func parseTaskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_VIDEO_TASK_ID", "invalid video task id"))
		return 0, false
	}
	return id, true
}

func formatTaskTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
