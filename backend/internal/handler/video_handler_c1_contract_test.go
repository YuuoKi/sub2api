package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// TestC1ApiKeyVideoResponseMatchesQCanvasContract is the C1 skeleton contract guard.
//
// The end-to-end skeleton is QCanvas → Sub2API (mock provider) → result → candidate.
// The Sub2API service→mock→succeeded chain is already proven dynamically by the
// service-package tests (TestFormASeedanceHappyPathInMemory drives create→worker→poll
// →succeeded against an in-memory repo). The QCanvas side maps the Sub2API response in
// apps/hono-api/.../sub2api.video-mock-gateway.service.ts#normalizeTaskRecord, which
// reads these keys: taskId←(task_id|taskId|id), status, resultUrl←(result_url|resultUrl
// |url), errorMessage←(error_message|errorMessage|message), provider.
//
// The one link NOT covered by either side's tests is the HTTP RESPONSE SERIALIZATION:
// that the api-key video response Sub2API actually emits carries those exact JSON keys.
// This test marshals the real response builder for a representative succeeded MOCK task
// (the shape the always-on mock adapter produces) and asserts the wire keys QCanvas
// consumes are present and correct. If a json tag drifts, this fails before QCanvas's
// mapping silently degrades to "sub2api-mock-task-missing-id".
func TestC1ApiKeyVideoResponseMatchesQCanvasContract(t *testing.T) {
	// Mirror what the mock adapter yields on success (provider=mock, succeeded, a
	// playable mock URL, a mock upstream id).
	task := &service.VideoTask{
		ID:             12345,
		Provider:       service.VideoProviderMock,
		Model:          "mock-video",
		TaskType:       service.VideoTaskTypeTextToVideo,
		Prompt:         "qcanvas C1 skeleton clip",
		Status:         service.VideoStatusSucceeded,
		UpstreamTaskID: "mock-video-12345",
		ResultURL:      "/api/v1/video/mock-assets/12345.svg",
	}

	raw, err := json.Marshal(apiKeyVideoTaskToResponse(task, nil))
	if err != nil {
		t.Fatalf("marshal api-key video response: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	// --- keys QCanvas normalizeTaskRecord reads (the create/poll/result contract) ---
	if id, ok := m["id"].(float64); !ok || int64(id) != 12345 {
		t.Fatalf("response must carry numeric `id` (QCanvas taskId source), got %v (%T)", m["id"], m["id"])
	}
	if m["status"] != "succeeded" {
		t.Fatalf("response `status` = %v, want succeeded (QCanvas status source)", m["status"])
	}
	if m["result_url"] != "/api/v1/video/mock-assets/12345.svg" {
		t.Fatalf("response `result_url` = %v (QCanvas resultUrl source)", m["result_url"])
	}
	if _, ok := m["error_message"]; !ok {
		t.Fatal("response must carry `error_message` (QCanvas errorMessage source)")
	}
	if m["provider"] != "mock" {
		t.Fatalf("response `provider` = %v, want mock", m["provider"])
	}

	// --- mock-only boundary fields QCanvas diagnostics relies on ---
	if m["mock_only"] != true {
		t.Fatalf("mock task must report mock_only=true, got %v", m["mock_only"])
	}
	if cnt, ok := m["real_provider_dispatch_count"].(float64); !ok || cnt != 0 {
		t.Fatalf("mock-only response must report real_provider_dispatch_count=0, got %v", m["real_provider_dispatch_count"])
	}
	if m["provider_boundary"] != "api-key-video-mock-only" {
		t.Fatalf("mock-only boundary = %v, want api-key-video-mock-only", m["provider_boundary"])
	}
}

// TestC1ApiKeyVideoCreateRequestAcceptsQCanvasBody proves the inbound side: the JSON
// body QCanvas's createSub2ApiVideoMockTask sends ({provider, task_type, prompt, model,
// ...}) binds onto Sub2API's apiKeyVideoTaskCreateRequest with provider=mock accepted.
func TestC1ApiKeyVideoCreateRequestAcceptsQCanvasBody(t *testing.T) {
	// The exact shape QCanvas POSTs to /v1/video/tasks (mock path).
	body := []byte(`{"provider":"mock","task_type":"text_to_video","prompt":"hello","model":"mock-video","metadata":{"nodeId":"n1"}}`)
	var req apiKeyVideoTaskCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("QCanvas body must unmarshal onto apiKeyVideoTaskCreateRequest: %v", err)
	}
	if req.Provider != "mock" {
		t.Fatalf("provider = %q, want mock", req.Provider)
	}
	if req.TaskType != "text_to_video" {
		t.Fatalf("task_type = %q, want text_to_video", req.TaskType)
	}
	if req.Prompt != "hello" {
		t.Fatalf("prompt = %q, want hello", req.Prompt)
	}
}
