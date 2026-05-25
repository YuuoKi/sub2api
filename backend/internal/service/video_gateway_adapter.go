package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type VideoAdapter interface {
	Provider() string
	CreateTask(ctx context.Context, account *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error)
	PollTask(ctx context.Context, account *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error)
	CancelTask(ctx context.Context, account *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error)
	NormalizeStatus(upstream string) string
	BuildCreatePayload(account *VideoProviderAccount, task *VideoTask) map[string]any
}

type VideoAdapterResult struct {
	UpstreamTaskID string
	Status         string
	ResultURL      string
	ErrorMessage   string
	CostEstimate   float64
	Payload        map[string]any
}

func NewVideoAdapterRegistry() map[string]VideoAdapter {
	adapters := []VideoAdapter{
		&mockVideoAdapter{},
		&seedanceVideoAdapter{},
		&klingVideoAdapter{},
	}
	out := make(map[string]VideoAdapter, len(adapters))
	for _, adapter := range adapters {
		out[adapter.Provider()] = adapter
	}
	return out
}

type mockVideoAdapter struct{}

func (a *mockVideoAdapter) Provider() string { return VideoProviderMock }

func (a *mockVideoAdapter) CreateTask(_ context.Context, _ *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error) {
	return &VideoAdapterResult{
		UpstreamTaskID: fmt.Sprintf("mock-video-%d", task.ID),
		Status:         VideoStatusSubmitted,
		Payload: map[string]any{
			"provider":         VideoProviderMock,
			"mock_should_fail": mockShouldFail(task.Prompt),
			"created_at":       time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

func (a *mockVideoAdapter) PollTask(_ context.Context, _ *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error) {
	if task.Status == VideoStatusSubmitted {
		return &VideoAdapterResult{
			Status: VideoStatusRunning,
			Payload: map[string]any{
				"stage": "mock_render_started",
			},
		}, nil
	}
	if mockShouldFail(task.Prompt) {
		return &VideoAdapterResult{
			Status:       VideoStatusFailed,
			ErrorMessage: "mock provider forced failure for P0 validation",
			Payload: map[string]any{
				"stage":  "mock_render_failed",
				"reason": "prompt contains failure trigger",
			},
		}, nil
	}
	return &VideoAdapterResult{
		Status:       VideoStatusSucceeded,
		ResultURL:    fmt.Sprintf("https://mock.sub2api.local/video/%d.mp4", task.ID),
		CostEstimate: 0,
		Payload: map[string]any{
			"stage":      "mock_render_completed",
			"result_url": fmt.Sprintf("https://mock.sub2api.local/video/%d.mp4", task.ID),
		},
	}, nil
}

func (a *mockVideoAdapter) CancelTask(_ context.Context, _ *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error) {
	return &VideoAdapterResult{
		Status: VideoStatusCancelled,
		Payload: map[string]any{
			"upstream_task_id": task.UpstreamTaskID,
			"stage":            "mock_cancelled",
		},
	}, nil
}

func (a *mockVideoAdapter) NormalizeStatus(upstream string) string {
	return normalizeVideoStatus(upstream)
}

func (a *mockVideoAdapter) BuildCreatePayload(_ *VideoProviderAccount, task *VideoTask) map[string]any {
	return map[string]any{
		"model":               task.Model,
		"task_type":           task.TaskType,
		"prompt":              task.Prompt,
		"negative_prompt":     task.NegativePrompt,
		"reference_image_url": task.ReferenceImageURL,
		"aspect_ratio":        task.AspectRatio,
		"duration":            task.Duration,
		"resolution":          task.Resolution,
	}
}

func mockShouldFail(prompt string) bool {
	lower := strings.ToLower(prompt)
	return strings.Contains(lower, "[fail]") ||
		strings.Contains(lower, "mock:fail") ||
		strings.Contains(prompt, "失败")
}

type seedanceVideoAdapter struct{}

func (a *seedanceVideoAdapter) Provider() string { return VideoProviderSeedance }

func (a *seedanceVideoAdapter) CreateTask(_ context.Context, account *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error) {
	if !account.APIKeyConfigured || strings.TrimSpace(account.PlainAPIKey) == "" {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"provider": "seedance",
			"reason":   "api key is not configured; real call skipped",
		})
	}

	baseURL := account.BaseURL
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}

	model := task.Model
	if model == "" {
		model = "doubao-seedance-2-0-260128"
	}

	content := []map[string]any{{"type": "text", "text": task.Prompt}}
	if task.ReferenceImageURL != "" {
		content = append(content, map[string]any{"type": "image_url", "image_url": map[string]string{"url": task.ReferenceImageURL}})
	}

	payload := map[string]any{
		"model":   model,
		"content": content,
	}
	if task.NegativePrompt != "" {
		payload["negative_prompt"] = task.NegativePrompt
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("seedance: marshal create payload: %w", err)
	}

	url := strings.TrimRight(baseURL, "/") + "/contents/generations/tasks"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("seedance: build create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+account.PlainAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, infraerrorsUnavailable("SEEDANCE_CREATE_HTTP_ERROR", "Seedance create task HTTP error: "+err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("seedance: read create response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, infraerrorsUnavailable("SEEDANCE_CREATE_UPSTREAM_ERROR",
			fmt.Sprintf("Seedance create task returned %d: %s", resp.StatusCode, truncate(string(respBody), 500)))
	}

	var parsed struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Error  struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("seedance: parse create response: %w", err)
	}

	if parsed.Error.Message != "" {
		return nil, infraerrorsUnavailable("SEEDANCE_CREATE_BUSINESS_ERROR", "Seedance: "+parsed.Error.Message)
	}

	return &VideoAdapterResult{
		UpstreamTaskID: parsed.ID,
		Status:         a.NormalizeStatus(parsed.Status),
		Payload: map[string]any{
			"provider":    "seedance",
			"upstream_id": parsed.ID,
			"model":       model,
			"created_at":  time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

func (a *seedanceVideoAdapter) PollTask(_ context.Context, account *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error) {
	if !account.APIKeyConfigured || strings.TrimSpace(account.PlainAPIKey) == "" {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"provider": "seedance",
			"reason":   "api key is not configured; poll skipped",
		})
	}

	baseURL := account.BaseURL
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3"
	}

	url := strings.TrimRight(baseURL, "/") + "/contents/generations/tasks/" + task.UpstreamTaskID
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("seedance: build poll request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+account.PlainAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, infraerrorsUnavailable("SEEDANCE_POLL_HTTP_ERROR", "Seedance poll task HTTP error: "+err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("seedance: read poll response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, infraerrorsUnavailable("SEEDANCE_POLL_UPSTREAM_ERROR",
			fmt.Sprintf("Seedance poll task returned %d: %s", resp.StatusCode, truncate(string(respBody), 500)))
	}

	var parsed struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Content struct {
			VideoURL string `json:"video_url"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("seedance: parse poll response: %w", err)
	}

	result := &VideoAdapterResult{
		UpstreamTaskID: parsed.ID,
		Status:         a.NormalizeStatus(parsed.Status),
		ResultURL:      parsed.Content.VideoURL,
		Payload: map[string]any{
			"provider": "seedance",
			"polled_at": time.Now().UTC().Format(time.RFC3339),
		},
	}

	if parsed.Error.Message != "" {
		result.ErrorMessage = parsed.Error.Message
		result.Status = VideoStatusFailed
	}

	return result, nil
}

func (a *seedanceVideoAdapter) CancelTask(_ context.Context, _ *VideoProviderAccount, _ *VideoTask) (*VideoAdapterResult, error) {
	return &VideoAdapterResult{Status: VideoStatusCancelled, Payload: map[string]any{"provider": "seedance", "mode": "skeleton"}}, nil
}

func (a *seedanceVideoAdapter) NormalizeStatus(upstream string) string {
	switch strings.ToLower(strings.TrimSpace(upstream)) {
	case "queued", "pending", "created":
		return VideoStatusQueued
	case "submitted", "in_queue":
		return VideoStatusSubmitted
	case "running", "processing":
		return VideoStatusRunning
	case "succeeded", "success", "completed":
		return VideoStatusSucceeded
	case "failed", "error":
		return VideoStatusFailed
	case "cancelled", "canceled", "deleted":
		return VideoStatusCancelled
	default:
		return VideoStatusRunning
	}
}

func (a *seedanceVideoAdapter) BuildCreatePayload(account *VideoProviderAccount, task *VideoTask) map[string]any {
	content := []map[string]any{{"type": "text", "text": task.Prompt}}
	if task.ReferenceImageURL != "" {
		content = append(content, map[string]any{"type": "image_url", "image_url": task.ReferenceImageURL})
	}
	return map[string]any{
		"base_url":        account.BaseURL,
		"model":           task.Model,
		"content":         content,
		"negative_prompt": task.NegativePrompt,
		"aspect_ratio":    task.AspectRatio,
		"duration":        task.Duration,
		"resolution":      task.Resolution,
		"source_docs":     "https://www.volcengine.com/docs/82379/1520757?lang=zh",
	}
}

type klingVideoAdapter struct{}

func (a *klingVideoAdapter) Provider() string { return VideoProviderKling }

func (a *klingVideoAdapter) CreateTask(_ context.Context, account *VideoProviderAccount, _ *VideoTask) (*VideoAdapterResult, error) {
	if !account.APIKeyConfigured || strings.TrimSpace(account.PlainAPIKey) == "" {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"provider": "kling",
			"reason":   "api key is not configured; real call skipped",
		})
	}
	return nil, infraerrorsUnavailable("KLING_REAL_CALL_DISABLED", "Kling adapter skeleton is mapped but real upstream calls are disabled in P0")
}

func (a *klingVideoAdapter) PollTask(_ context.Context, account *VideoProviderAccount, _ *VideoTask) (*VideoAdapterResult, error) {
	if !account.APIKeyConfigured || strings.TrimSpace(account.PlainAPIKey) == "" {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"provider": "kling",
			"reason":   "api key is not configured; poll skipped",
		})
	}
	return nil, infraerrorsUnavailable("KLING_REAL_POLL_DISABLED", "Kling poll skeleton is mapped but real upstream calls are disabled in P0")
}

func (a *klingVideoAdapter) CancelTask(_ context.Context, _ *VideoProviderAccount, _ *VideoTask) (*VideoAdapterResult, error) {
	return &VideoAdapterResult{Status: VideoStatusCancelled, Payload: map[string]any{"provider": "kling", "mode": "skeleton"}}, nil
}

func (a *klingVideoAdapter) NormalizeStatus(upstream string) string {
	switch strings.ToLower(strings.TrimSpace(upstream)) {
	case "submitted", "created", "queued", "pending":
		return VideoStatusSubmitted
	case "processing", "running":
		return VideoStatusRunning
	case "succeed", "succeeded", "success", "completed":
		return VideoStatusSucceeded
	case "failed", "fail", "error":
		return VideoStatusFailed
	case "cancelled", "canceled":
		return VideoStatusCancelled
	default:
		return VideoStatusRunning
	}
}

func (a *klingVideoAdapter) BuildCreatePayload(account *VideoProviderAccount, task *VideoTask) map[string]any {
	payload := map[string]any{
		"base_url":        account.BaseURL,
		"model_name":      task.Model,
		"prompt":          task.Prompt,
		"negative_prompt": task.NegativePrompt,
		"aspect_ratio":    task.AspectRatio,
		"duration":        task.Duration,
		"resolution":      task.Resolution,
		"source_docs":     "https://app.klingai.com/cn/dev/document-api/apiReference/updateNotice",
	}
	if task.ReferenceImageURL != "" {
		payload["image"] = task.ReferenceImageURL
	}
	if task.ReferenceVideoURL != "" {
		payload["reference_video"] = task.ReferenceVideoURL
	}
	return payload
}

func normalizeVideoStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case VideoStatusQueued:
		return VideoStatusQueued
	case VideoStatusSubmitted:
		return VideoStatusSubmitted
	case VideoStatusRunning:
		return VideoStatusRunning
	case VideoStatusSucceeded, "success", "completed", "complete":
		return VideoStatusSucceeded
	case VideoStatusFailed, "error":
		return VideoStatusFailed
	case VideoStatusCancelled, "canceled":
		return VideoStatusCancelled
	default:
		return VideoStatusRunning
	}
}

func infraerrorsUnavailable(reason, message string) error {
	return infraerrors.ServiceUnavailable(reason, message)
}
