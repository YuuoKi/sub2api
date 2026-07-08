package handler

import (
	"fmt"
	"strconv"
	"strings"
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
	ProviderAccountID int64                          `json:"provider_account_id" binding:"omitempty,min=0"`
	TaskType          string                         `json:"task_type" binding:"required,oneof=text_to_video image_to_video reference_to_video"`
	Model             string                         `json:"model" binding:"omitempty,max=200"`
	Prompt            string                         `json:"prompt" binding:"required,max=8000"`
	NegativePrompt    string                         `json:"negative_prompt" binding:"omitempty,max=4000"`
	ReferenceImageURL string                         `json:"reference_image_url" binding:"omitempty,max=1000"`
	ReferenceVideoURL string                         `json:"reference_video_url" binding:"omitempty,max=1000"`
	Content           []service.VideoTaskContentItem `json:"content" binding:"omitempty,dive"`
	AspectRatio       string                         `json:"aspect_ratio" binding:"omitempty,max=20"`
	Duration          int                            `json:"duration" binding:"omitempty"`
	Resolution        string                         `json:"resolution" binding:"omitempty,max=20"`
	GenerateAudio     *bool                          `json:"generate_audio" binding:"omitempty"`
	Watermark         *bool                          `json:"watermark" binding:"omitempty"`
	CameraFixed       *bool                          `json:"camera_fixed" binding:"omitempty"`
	ReturnLastFrame   *bool                          `json:"return_last_frame" binding:"omitempty"`
}

type apiKeyVideoTaskCreateRequest struct {
	Provider          string                         `json:"provider" binding:"omitempty,oneof=mock seedance kling"`
	TrialMode         string                         `json:"trial_mode" binding:"omitempty,oneof=tiny_real"`
	TaskType          string                         `json:"task_type" binding:"required,oneof=text_to_video image_to_video reference_to_video"`
	Model             string                         `json:"model" binding:"omitempty,max=200"`
	Prompt            string                         `json:"prompt" binding:"required,max=8000"`
	NegativePrompt    string                         `json:"negative_prompt" binding:"omitempty,max=4000"`
	ReferenceImageURL string                         `json:"reference_image_url" binding:"omitempty,max=1000"`
	ReferenceVideoURL string                         `json:"reference_video_url" binding:"omitempty,max=1000"`
	Content           []service.VideoTaskContentItem `json:"content" binding:"omitempty,dive"`
	AspectRatio       string                         `json:"aspect_ratio" binding:"omitempty,max=20"`
	Duration          int                            `json:"duration" binding:"omitempty"`
	Resolution        string                         `json:"resolution" binding:"omitempty,max=20"`
	GenerateAudio     *bool                          `json:"generate_audio" binding:"omitempty"`
	Watermark         *bool                          `json:"watermark" binding:"omitempty"`
	CameraFixed       *bool                          `json:"camera_fixed" binding:"omitempty"`
	ReturnLastFrame   *bool                          `json:"return_last_frame" binding:"omitempty"`
}

type videoTaskResponse struct {
	ID                  int64                          `json:"id"`
	ProviderAccountID   int64                          `json:"provider_account_id"`
	ProviderAccountName string                         `json:"provider_account_name"`
	Provider            string                         `json:"provider"`
	Model               string                         `json:"model"`
	TaskType            string                         `json:"task_type"`
	Prompt              string                         `json:"prompt"`
	NegativePrompt      string                         `json:"negative_prompt"`
	ReferenceImageURL   string                         `json:"reference_image_url"`
	ReferenceVideoURL   string                         `json:"reference_video_url"`
	Content             []service.VideoTaskContentItem `json:"content,omitempty"`
	HasVideoInput       bool                           `json:"has_video_input"`
	AspectRatio         string                         `json:"aspect_ratio"`
	Duration            int                            `json:"duration"`
	Resolution          string                         `json:"resolution"`
	GenerateAudio       *bool                          `json:"generate_audio,omitempty"`
	Watermark           *bool                          `json:"watermark,omitempty"`
	CameraFixed         *bool                          `json:"camera_fixed,omitempty"`
	ReturnLastFrame     *bool                          `json:"return_last_frame,omitempty"`
	Status              string                         `json:"status"`
	UpstreamTaskID      string                         `json:"upstream_task_id"`
	ResultURL             string                         `json:"result_url"`
	ResultURLExpiresAt    *string                        `json:"result_url_expires_at,omitempty"`
	ResultURLExpirySource string                         `json:"result_url_expiry_source,omitempty"`
	Usage                 videoTaskUsageResponse         `json:"usage"`
	ActualResolution    string                         `json:"actual_resolution"`
	ActualDuration      *int                           `json:"actual_duration"`
	LastFrameURL        string                         `json:"last_frame_url"`
	ErrorMessage        string                         `json:"error_message"`
	CostEstimate        float64                        `json:"cost_estimate"`
	CreatedBy           int64                          `json:"created_by"`
	CreatedByEmail      string                         `json:"created_by_email"`
	CreatedByName       string                         `json:"created_by_name"`
	CreatedByLabel      string                         `json:"created_by_label"`
	RoutingStrategy     string                         `json:"routing_strategy"`
	RoutingReason       string                         `json:"routing_reason"`
	CreatedAt           string                         `json:"created_at"`
	UpdatedAt           string                         `json:"updated_at"`
	CompletedAt         *string                        `json:"completed_at"`
	Events              []videoTaskEventResponse       `json:"events,omitempty"`
}

type videoTaskUsageResponse struct {
	TotalTokens int64 `json:"total_tokens"`
}

type apiKeyVideoTaskResponse struct {
	videoTaskResponse
	MockOnly                  bool   `json:"mock_only"`
	ProviderBoundary          string `json:"provider_boundary"`
	RealProviderDispatchCount int    `json:"real_provider_dispatch_count"`
	TrialMode                 string `json:"trial_mode,omitempty"`
	BlockedReason             string `json:"blocked_reason,omitempty"`
	TrialGateResult           string `json:"trial_gate_result,omitempty"`
}

type videoProviderAccountResponse struct {
	ID                 int64  `json:"id"`
	Provider           string `json:"provider"`
	DisplayName        string `json:"display_name"`
	Enabled            bool   `json:"enabled"`
	APIKeyConfigured   bool   `json:"api_key_configured"`
	MaskedKey          string `json:"masked_key"`
	BaseURL            string `json:"base_url"`
	DefaultModel       string `json:"default_model"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	KeyStatus          string `json:"key_status"`
	HealthStatus       string `json:"health_status"`
	DiagnosticType     string `json:"diagnostic_type"`
	SuggestedAction    string `json:"suggested_action"`
	Priority           int    `json:"priority"`
	CurrentInflight    int64  `json:"current_inflight"`
	TodayTasks         int64  `json:"today_tasks"`
	TodayFailures      int64  `json:"today_failures"`
	LastError          string `json:"last_error"`
	LastTestAt         string `json:"last_test_at"`
	RouteAvailable     bool   `json:"route_available"`
	RouteSkipReason    string `json:"route_skip_reason"`
}

type videoTaskEventResponse struct {
	ID          int64          `json:"id"`
	VideoTaskID int64          `json:"video_task_id"`
	EventType   string         `json:"event_type"`
	Message     string         `json:"message"`
	Payload     map[string]any `json:"payload_json"`
	CreatedAt   string         `json:"created_at"`
}

func (h *VideoHandler) ListProviders(c *gin.Context) {
	items, err := h.video.ListProviderAccounts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]videoProviderAccountResponse, 0, len(items))
	for _, item := range items {
		out = append(out, videoProviderAccountToResponse(item))
	}
	response.Success(c, gin.H{"items": out})
}

func (h *VideoHandler) ListAPIKeyVideoProviders(c *gin.Context) {
	items, err := h.video.ListAPIKeyTrialProviders(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]videoProviderAccountResponse, 0, len(items))
	mockOnly := true
	for _, item := range items {
		out = append(out, videoProviderAccountToResponse(item))
		if item.Provider == service.VideoProviderSeedance && item.RouteSkipReason != "seedance provider account is not configured" {
			mockOnly = false
		}
	}
	response.Success(c, gin.H{
		"items":                        out,
		"mock_only":                    mockOnly,
		"real_provider_dispatch_count": 0,
		"trial_mode_supported":         true,
	})
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
	role, _ := middleware2.GetUserRoleFromContext(c)
	params := service.VideoTaskCreateParams{
		TaskType:                 req.TaskType,
		Model:                    req.Model,
		Prompt:                   req.Prompt,
		NegativePrompt:           req.NegativePrompt,
		ReferenceImageURL:        req.ReferenceImageURL,
		ReferenceVideoURL:        req.ReferenceVideoURL,
		Content:                  req.Content,
		AspectRatio:              req.AspectRatio,
		Duration:                 req.Duration,
		Resolution:               req.Resolution,
		GenerateAudio:            req.GenerateAudio,
		Watermark:                req.Watermark,
		CameraFixed:              req.CameraFixed,
		ReturnLastFrame:          req.ReturnLastFrame,
		CreatedBy:                subject.UserID,
		EnforceRealProviderTrial: true,
	}
	if role == "admin" {
		params.ProviderAccountID = req.ProviderAccountID
		params.EnforceRealProviderTrial = false
		params.RequireSeedanceProductionAuthorization = true
	}
	task, err := h.video.CreateTask(c.Request.Context(), params)
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

func (h *VideoHandler) CreateAPIKeyVideoTask(c *gin.Context) {
	var req apiKeyVideoTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	provider := strings.TrimSpace(req.Provider)
	var task *service.VideoTask
	var err error
	if provider == "" || provider == service.VideoProviderMock {
		task, err = h.video.CreateAPIKeyMockOnlyTask(c.Request.Context(), req.Provider, service.VideoTaskCreateParams{
			TaskType:          req.TaskType,
			Model:             req.Model,
			Prompt:            req.Prompt,
			NegativePrompt:    req.NegativePrompt,
			ReferenceImageURL: req.ReferenceImageURL,
			ReferenceVideoURL: req.ReferenceVideoURL,
			Content:           req.Content,
			AspectRatio:       req.AspectRatio,
			Duration:          req.Duration,
			Resolution:        req.Resolution,
			GenerateAudio:     req.GenerateAudio,
			Watermark:         req.Watermark,
			CameraFixed:       req.CameraFixed,
			ReturnLastFrame:   req.ReturnLastFrame,
			CreatedBy:         subject.UserID,
		})
	} else if provider == service.VideoProviderSeedance && req.TrialMode == "tiny_real" {
		task, err = h.video.CreateAPIKeySeedanceTinyTrialTask(c.Request.Context(), req.Provider, service.VideoTaskCreateParams{
			TaskType:          req.TaskType,
			Model:             req.Model,
			Prompt:            req.Prompt,
			NegativePrompt:    req.NegativePrompt,
			ReferenceImageURL: req.ReferenceImageURL,
			ReferenceVideoURL: req.ReferenceVideoURL,
			Content:           req.Content,
			AspectRatio:       req.AspectRatio,
			Duration:          req.Duration,
			Resolution:        req.Resolution,
			GenerateAudio:     req.GenerateAudio,
			Watermark:         req.Watermark,
			CameraFixed:       req.CameraFixed,
			ReturnLastFrame:   req.ReturnLastFrame,
			CreatedBy:         subject.UserID,
		})
	} else if provider == service.VideoProviderSeedance {
		task, err = h.video.CreateAPIKeySeedanceProductionTask(c.Request.Context(), req.Provider, service.VideoTaskCreateParams{
			TaskType:          req.TaskType,
			Model:             req.Model,
			Prompt:            req.Prompt,
			NegativePrompt:    req.NegativePrompt,
			ReferenceImageURL: req.ReferenceImageURL,
			ReferenceVideoURL: req.ReferenceVideoURL,
			Content:           req.Content,
			AspectRatio:       req.AspectRatio,
			Duration:          req.Duration,
			Resolution:        req.Resolution,
			GenerateAudio:     req.GenerateAudio,
			Watermark:         req.Watermark,
			CameraFixed:       req.CameraFixed,
			ReturnLastFrame:   req.ReturnLastFrame,
			CreatedBy:         subject.UserID,
		})
	} else {
		response.ErrorFrom(c, infraerrors.Forbidden(
			"VIDEO_PROVIDER_DISABLED",
			fmt.Sprintf("provider %s is disabled in API-key mock-only video gateway; use provider=mock", provider),
		).WithMetadata(map[string]string{
			"provider":                     provider,
			"mock_only":                    "true",
			"real_provider_dispatch_count": "0",
		}))
		return
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, apiKeyVideoTaskToResponse(task, nil))
}

func (h *VideoHandler) GetAPIKeyVideoTask(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}
	subject, authOK := middleware2.GetAuthSubjectFromContext(c)
	if !authOK || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	role, _ := middleware2.GetUserRoleFromContext(c)
	task, events, err := h.video.GetAPIKeyTrialTask(c.Request.Context(), id, subject.UserID, role == "admin")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, apiKeyVideoTaskToResponse(task, events))
}

func (h *VideoHandler) CancelAPIKeyVideoTask(c *gin.Context) {
	id, ok := parseTaskID(c)
	if !ok {
		return
	}
	subject, authOK := middleware2.GetAuthSubjectFromContext(c)
	if !authOK || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	role, _ := middleware2.GetUserRoleFromContext(c)
	task, err := h.video.CancelAPIKeyTrialTask(c.Request.Context(), id, subject.UserID, role == "admin")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, apiKeyVideoTaskToResponse(task, nil))
}

func (h *VideoHandler) CreateDramaTask(c *gin.Context) {
	var req service.DramaTaskCreateParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	req.CreatedBy = subject.UserID
	task, err := h.video.CreateDramaTask(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, task)
}

func (h *VideoHandler) ListDramaTasks(c *gin.Context) {
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	role, _ := middleware2.GetUserRoleFromContext(c)
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.video.ListDramaTasks(c.Request.Context(), service.VideoTaskListParams{
		Page:      page,
		PageSize:  pageSize,
		Status:    c.Query("status"),
		CreatedBy: subject.UserID,
		IsAdmin:   role == "admin",
	}, dramaTaskFilters(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *VideoHandler) GetDramaTask(c *gin.Context) {
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	role, _ := middleware2.GetUserRoleFromContext(c)
	task, err := h.video.GetDramaTask(c.Request.Context(), c.Param("id"), subject.UserID, role == "admin")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, task)
}

func (h *VideoHandler) RecordDramaShotDecision(c *gin.Context) {
	var req service.DramaShotDecisionParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	role, _ := middleware2.GetUserRoleFromContext(c)
	req.CreatedBy = subject.UserID
	record, err := h.video.RecordDramaShotDecision(c.Request.Context(), req, subject.UserID, role == "admin")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, record)
}

func (h *VideoHandler) RecordDramaPromptArtifact(c *gin.Context) {
	var req service.DramaPromptArtifactParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	role, _ := middleware2.GetUserRoleFromContext(c)
	req.CreatedBy = subject.UserID
	record, err := h.video.RecordDramaPromptArtifact(c.Request.Context(), req, subject.UserID, role == "admin")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, record)
}

func (h *VideoHandler) RecommendDramaProvider(c *gin.Context) {
	recommendation, err := h.video.RecommendDramaProvider(c.Request.Context(), service.DramaProviderRecommendParams{
		DramaType:          c.Query("drama_type"),
		SceneType:          c.Query("scene_type"),
		ShotRole:           c.Query("shot_role"),
		ReferenceAssetType: c.Query("reference_asset_type"),
		RequestedEngineCapabilities: service.DramaEngineCapabilityRequest{
			SupportsRealPerson:      queryBool(c, "supports_real_person"),
			SupportsImageToVideo:    queryBool(c, "supports_image_to_video"),
			SupportsMotionControl:   queryBool(c, "supports_motion_control"),
			SupportsLipsync:         queryBool(c, "supports_lipsync"),
			SupportsNativeAudio:     queryBool(c, "supports_native_audio"),
			SupportsGlobalReference: queryBool(c, "supports_global_reference"),
			SupportsAnimeDrama:      queryBool(c, "supports_anime_drama"),
		},
		DurationSeconds: queryInt(c, "duration_seconds"),
		AspectRatio:     c.Query("aspect_ratio"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, recommendation)
}

func (h *VideoHandler) DramaEngineCapabilityMatrix(c *gin.Context) {
	response.Success(c, gin.H{"items": h.video.DramaEngineCapabilityMatrix()})
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
	var resultExpiresAt *string
	resultExpirySource := ""
	if task.ResultURL != "" {
		if expires, source := service.ParseResultURLExpiry(task.ResultURL, task.CompletedAt); expires != nil {
			v := formatTaskTime(*expires)
			resultExpiresAt = &v
			resultExpirySource = source
		} else {
			resultExpirySource = source
		}
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
	routingStrategy := videoTaskEventPayloadString(events, "routed", "strategy")
	routingReason := videoTaskEventPayloadString(events, "routed", "reason")
	totalTokens := int64(0)
	if task.UsageTotalTokens != nil {
		totalTokens = *task.UsageTotalTokens
	}
	var actualDuration *int
	if task.ActualDuration != nil {
		v := *task.ActualDuration
		actualDuration = &v
	}
	return videoTaskResponse{
		ID:                  task.ID,
		ProviderAccountID:   task.ProviderAccountID,
		ProviderAccountName: task.ProviderAccountName,
		Provider:            task.Provider,
		Model:               task.Model,
		TaskType:            task.TaskType,
		Prompt:              task.Prompt,
		NegativePrompt:      task.NegativePrompt,
		ReferenceImageURL:   task.ReferenceImageURL,
		ReferenceVideoURL:   task.ReferenceVideoURL,
		Content:             task.Content,
		HasVideoInput:       task.HasVideoInput,
		AspectRatio:         task.AspectRatio,
		Duration:            task.Duration,
		Resolution:          task.Resolution,
		GenerateAudio:       task.GenerateAudio,
		Watermark:           task.Watermark,
		CameraFixed:         task.CameraFixed,
		ReturnLastFrame:     task.ReturnLastFrame,
		Status:              task.Status,
		UpstreamTaskID:        task.UpstreamTaskID,
		ResultURL:             task.ResultURL,
		ResultURLExpiresAt:    resultExpiresAt,
		ResultURLExpirySource: resultExpirySource,
		Usage:                 videoTaskUsageResponse{TotalTokens: totalTokens},
		ActualResolution:      task.ActualResolution,
		ActualDuration:      actualDuration,
		LastFrameURL:        task.LastFrameURL,
		ErrorMessage:        task.ErrorMessage,
		CostEstimate:        task.CostEstimate,
		CreatedBy:           task.CreatedBy,
		CreatedByEmail:      task.CreatedByEmail,
		CreatedByName:       task.CreatedByName,
		CreatedByLabel:      videoCreatedByLabel(task),
		RoutingStrategy:     routingStrategy,
		RoutingReason:       routingReason,
		CreatedAt:           formatTaskTime(task.CreatedAt),
		UpdatedAt:           formatTaskTime(task.UpdatedAt),
		CompletedAt:         completed,
		Events:              eventResponses,
	}
}

func apiKeyVideoTaskToResponse(task *service.VideoTask, events []*service.VideoTaskEvent) apiKeyVideoTaskResponse {
	mockOnly := task != nil && task.Provider == service.VideoProviderMock
	boundary := "api-key-video-mock-only"
	realDispatchCount := 0
	trialMode := ""
	blockedReason := ""
	trialGateResult := ""

	if task != nil && task.Provider == service.VideoProviderSeedance {
		boundary = "api-key-video-seedance-tiny-trial"
	}

	for _, ev := range events {
		if task != nil && task.Provider != service.VideoProviderMock && ev != nil && ev.EventType == service.VideoStatusSubmitted {
			realDispatchCount++
		}
		if ev != nil && ev.EventType == "trial_gate" && ev.Payload != nil {
			if v, ok := ev.Payload["trial_mode"]; ok {
				trialMode = fmt.Sprint(v)
			}
			if v, ok := ev.Payload["blocked_reasons"]; ok {
				blockedReason = fmt.Sprint(v)
			}
			if v, ok := ev.Payload["gate_result"]; ok {
				trialGateResult = fmt.Sprint(v)
			}
		}
	}

	return apiKeyVideoTaskResponse{
		videoTaskResponse:         videoTaskToResponse(task, events),
		MockOnly:                  mockOnly,
		ProviderBoundary:          boundary,
		RealProviderDispatchCount: realDispatchCount,
		TrialMode:                 trialMode,
		BlockedReason:             blockedReason,
		TrialGateResult:           trialGateResult,
	}
}

func videoProviderAccountToResponse(item *service.VideoProviderAccount) videoProviderAccountResponse {
	if item == nil {
		return videoProviderAccountResponse{}
	}
	return videoProviderAccountResponse{
		ID:                 item.ID,
		Provider:           item.Provider,
		DisplayName:        item.DisplayName,
		Enabled:            item.Enabled,
		APIKeyConfigured:   item.APIKeyConfigured,
		MaskedKey:          item.MaskedKey,
		BaseURL:            item.BaseURL,
		DefaultModel:       item.DefaultModel,
		RateLimitPerMinute: item.RateLimitPerMinute,
		KeyStatus:          item.KeyStatus,
		HealthStatus:       item.HealthStatus,
		DiagnosticType:     item.DiagnosticType,
		SuggestedAction:    item.SuggestedAction,
		Priority:           item.Priority,
		CurrentInflight:    item.CurrentInflight,
		TodayTasks:         item.TodayTasks,
		TodayFailures:      item.TodayFailures,
		LastError:          item.LastError,
		LastTestAt:         formatOptionalTaskTime(item.LastTestAt),
		RouteAvailable:     item.RouteAvailable,
		RouteSkipReason:    item.RouteSkipReason,
	}
}

func videoCreatedByLabel(task *service.VideoTask) string {
	if task == nil {
		return ""
	}
	name := strings.TrimSpace(task.CreatedByName)
	email := strings.TrimSpace(task.CreatedByEmail)
	switch {
	case name != "" && email != "":
		return name + " / " + email
	case name != "":
		return name
	case email != "":
		return email
	default:
		return fmt.Sprintf("用户 #%d", task.CreatedBy)
	}
}

func videoTaskEventPayloadString(events []*service.VideoTaskEvent, eventType, key string) string {
	for _, event := range events {
		if event == nil || event.EventType != eventType || event.Payload == nil {
			continue
		}
		value, ok := event.Payload[key]
		if !ok || value == nil {
			continue
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func parseTaskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_VIDEO_TASK_ID", "invalid video task id"))
		return 0, false
	}
	return id, true
}

func dramaTaskFilters(c *gin.Context) map[string]string {
	return map[string]string{
		"employee_alias": c.Query("employee_alias"),
		"api_client_id":  c.Query("api_client_id"),
		"project_id":     c.Query("project_id"),
		"drama_type":     c.Query("drama_type"),
		"genre":          c.Query("genre"),
		"scene_type":     c.Query("scene_type"),
		"engine":         c.Query("engine"),
		"model":          c.Query("model"),
		"mode":           c.Query("mode"),
		"status":         c.Query("status"),
	}
}

func queryBool(c *gin.Context, key string) bool {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return false
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return raw == "1"
	}
	return value
}

func queryInt(c *gin.Context, key string) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func formatTaskTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func formatOptionalTaskTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTaskTime(*t)
}
