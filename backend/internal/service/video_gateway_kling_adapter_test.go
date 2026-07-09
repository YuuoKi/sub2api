package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

const (
	klingTestAccessKey = "ak-kling-fixture-001122334455"
	klingTestSecretKey = "sk-kling-fixture-556677889900"
)

func newSmokeGatedKlingFixture(t *testing.T, handler http.HandlerFunc) (*klingVideoAdapter, *VideoProviderAccount, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	t.Setenv("SUB2API_VIDEO_REAL_SMOKE_ENABLED", "1")
	t.Setenv("SUB2API_VIDEO_REDACTED_EVENT_LOG", t.TempDir()+"/kling-redacted-events.log")
	t.Setenv("SUB2API_VIDEO_URL_ALLOWLIST", "example.com,klingai.com,volces.com")

	acc := &VideoProviderAccount{
		ID:               42,
		Provider:         VideoProviderKling,
		Enabled:          true,
		APIKeyConfigured: true,
		PlainAccessKey:   klingTestAccessKey,
		PlainSecretKey:   klingTestSecretKey,
		BaseURL:          srv.URL,
		Metadata:         map[string]any{"single_smoke_authorized": true},
	}
	return &klingVideoAdapter{}, acc, srv
}

func TestKlingCreateTextToVideoPayloadAndJWT(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotAuth   string
		gotBody   map[string]any
		called    bool
	)
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"task_id":"kt-t2v-1","task_status":"submitted"}}`))
	})

	audio := true
	res, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:          "kling-2.6-pro",
		TaskType:       VideoTaskTypeTextToVideo,
		Prompt:         "a fox runs",
		NegativePrompt: "blur",
		AspectRatio:    "16:9",
		Duration:       5,
		GenerateAudio:  &audio,
		Resolution:     "1080p",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if !called {
		t.Fatal("expected upstream HTTP call")
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/videos/text2video" {
		t.Fatalf("request = %s %s, want POST /v1/videos/text2video", gotMethod, gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") || len(strings.TrimPrefix(gotAuth, "Bearer ")) < 20 {
		t.Fatalf("Authorization must be Bearer JWT, got %q", gotAuth)
	}
	if strings.Contains(gotAuth, klingTestAccessKey) || strings.Contains(gotAuth, klingTestSecretKey) {
		t.Fatalf("Authorization must not contain raw AK/SK: %q", gotAuth)
	}
	if gotBody["model_name"] != "kling-v2-6" {
		t.Fatalf("model_name=%v want kling-v2-6; body=%+v", gotBody["model_name"], gotBody)
	}
	if gotBody["prompt"] != "a fox runs" || gotBody["negative_prompt"] != "blur" {
		t.Fatalf("prompt fields mismatch: %+v", gotBody)
	}
	if gotBody["duration"] != "5" {
		t.Fatalf("duration=%v want string \"5\"", gotBody["duration"])
	}
	if gotBody["aspect_ratio"] != "16:9" {
		t.Fatalf("aspect_ratio=%v", gotBody["aspect_ratio"])
	}
	if gotBody["mode"] != "pro" {
		t.Fatalf("mode=%v want pro for 1080p", gotBody["mode"])
	}
	if gotBody["sound"] != "on" {
		t.Fatalf("sound=%v want on", gotBody["sound"])
	}
	if res.UpstreamTaskID != "kt-t2v-1" {
		t.Fatalf("UpstreamTaskID=%q", res.UpstreamTaskID)
	}
	if res.Status != VideoStatusSubmitted {
		t.Fatalf("status=%q want submitted", res.Status)
	}
}

func TestKlingCreateImageToVideoPayload(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-i2v-1","task_status":"submitted"}}`))
	})

	res, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v1",
		TaskType: VideoTaskTypeImageToVideo,
		Prompt:   "animate",
		Duration: 5,
		Content: []VideoTaskContentItem{
			{Type: VideoContentTypeText, Text: "animate"},
			{Type: VideoContentTypeImageURL, Role: VideoContentRoleFirstFrame, URL: "https://cdn.example.com/first.png"},
			{Type: VideoContentTypeImageURL, Role: VideoContentRoleLastFrame, URL: "https://cdn.example.com/last.png"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if gotPath != "/v1/videos/image2video" {
		t.Fatalf("path=%q", gotPath)
	}
	if gotBody["model_name"] != "kling-v1" {
		t.Fatalf("model_name=%v", gotBody["model_name"])
	}
	if gotBody["image"] != "https://cdn.example.com/first.png" {
		t.Fatalf("image=%v", gotBody["image"])
	}
	if gotBody["image_tail"] != "https://cdn.example.com/last.png" {
		t.Fatalf("image_tail=%v", gotBody["image_tail"])
	}
	if res.UpstreamTaskID != "kt-i2v-1" {
		t.Fatalf("UpstreamTaskID=%q", res.UpstreamTaskID)
	}
}

func TestKlingCreateMultiImageReferencePayload(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-multi-1","task_status":"submitted"}}`))
	})

	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v2-6",
		TaskType: VideoTaskTypeReferenceToVideo,
		Prompt:   "combine refs",
		Duration: 5,
		Content: []VideoTaskContentItem{
			{Type: VideoContentTypeText, Text: "combine refs"},
			{Type: VideoContentTypeImageURL, Role: VideoContentRoleReferenceImage, URL: "https://cdn.example.com/a.png"},
			{Type: VideoContentTypeImageURL, Role: VideoContentRoleReferenceImage, URL: "https://cdn.example.com/b.png"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if gotPath != "/v1/videos/multi-image2video" {
		t.Fatalf("path=%q want multi-image2video", gotPath)
	}
	if gotBody["model_name"] != "kling-v2-6" {
		t.Fatalf("model_name=%v", gotBody["model_name"])
	}
	list, ok := gotBody["image_list"].([]any)
	if !ok || len(list) != 2 {
		t.Fatalf("image_list=%v", gotBody["image_list"])
	}
}

func TestKlingCreateOmniReferencePayload(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-omni-1","task_status":"submitted"}}`))
	})

	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-3.0-omni",
		TaskType: VideoTaskTypeReferenceToVideo,
		Prompt:   "omni scene",
		Duration: 5,
		Content: []VideoTaskContentItem{
			{Type: VideoContentTypeText, Text: "omni scene"},
			{Type: VideoContentTypeImageURL, Role: VideoContentRoleReferenceImage, URL: "https://cdn.example.com/ref.png"},
			{Type: VideoContentTypeVideoURL, Role: VideoContentRoleReferenceVideo, URL: "https://cdn.example.com/ref.mp4"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if gotPath != "/v1/videos/omni-video" {
		t.Fatalf("path=%q want omni-video", gotPath)
	}
	if gotBody["model_name"] != "kling-v3-omni" {
		t.Fatalf("model_name=%v want kling-v3-omni", gotBody["model_name"])
	}
	imgList, ok := gotBody["image_list"].([]any)
	if !ok || len(imgList) != 1 {
		t.Fatalf("image_list=%v", gotBody["image_list"])
	}
	vidList, ok := gotBody["video_list"].([]any)
	if !ok || len(vidList) != 1 {
		t.Fatalf("video_list=%v", gotBody["video_list"])
	}
	vid0, _ := vidList[0].(map[string]any)
	if vid0["video_url"] != "https://cdn.example.com/ref.mp4" {
		t.Fatalf("video_list[0]=%v", vidList[0])
	}
}

func TestKlingPollSucceedReturnsResultURL(t *testing.T) {
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("poll method=%s", r.Method)
		}
		if r.URL.Path != "/v1/videos/text2video/kt-poll-1" {
			t.Fatalf("poll path=%s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Fatalf("missing JWT auth: %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"task_id":"kt-poll-1","task_status":"succeed","task_result":{"videos":[{"url":"https://cdn.example.com/out.mp4"}]}}}`))
	})

	res, err := adapter.PollTask(context.Background(), acc, &VideoTask{
		Model:          "kling-v1",
		TaskType:       VideoTaskTypeTextToVideo,
		Duration:       5,
		UpstreamTaskID: "kt-poll-1",
	})
	if err != nil {
		t.Fatalf("PollTask: %v", err)
	}
	if res.Status != VideoStatusSucceeded {
		t.Fatalf("status=%q want succeeded", res.Status)
	}
	if res.ResultURL != "https://cdn.example.com/out.mp4" {
		t.Fatalf("ResultURL=%q", res.ResultURL)
	}
}

func TestKlingRejectsHostileUpstreamTaskIDOnCreate(t *testing.T) {
	for _, hostile := range []string{"../evil", "a/b?x=1", "id with space", "a#frag", ""} {
		adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			body := `{"code":0,"data":{"task_id":` + jsonString(hostile) + `,"task_status":"submitted"}}`
			_, _ = w.Write([]byte(body))
		})
		res, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
			Model:    "kling-v1",
			TaskType: VideoTaskTypeTextToVideo,
			Prompt:   "x",
			Duration: 5,
		})
		if err == nil {
			t.Fatalf("hostile task_id %q: expected rejection, got %+v", hostile, res)
		}
		if res != nil && res.UpstreamTaskID != "" {
			t.Fatalf("hostile task_id %q must not be persisted, got %q", hostile, res.UpstreamTaskID)
		}
		if !strings.Contains(err.Error(), "KLING_INVALID_UPSTREAM_TASK_ID") &&
			!strings.Contains(err.Error(), "task_id") {
			t.Fatalf("hostile task_id %q: want invalid-id error, got %v", hostile, err)
		}
	}
}

func TestKlingPollRejectsHostileUpstreamTaskIDWithoutNetwork(t *testing.T) {
	for _, hostile := range []string{"../evil", "a/b?x=1", "ok/../x", "id?q=1"} {
		called := false
		var gotPath string
		adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
			called = true
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"x","task_status":"succeed","task_result":{"videos":[{"url":"https://cdn.example.com/out.mp4"}]}}}`))
		})
		_, err := adapter.PollTask(context.Background(), acc, &VideoTask{
			Model:          "kling-v1",
			TaskType:       VideoTaskTypeTextToVideo,
			Duration:       5,
			UpstreamTaskID: hostile,
		})
		if err == nil {
			t.Fatalf("hostile UpstreamTaskID %q: expected rejection", hostile)
		}
		if called {
			t.Fatalf("hostile UpstreamTaskID %q must not open network (path=%q)", hostile, gotPath)
		}
		if strings.Contains(gotPath, "..") || strings.Contains(gotPath, "?") {
			t.Fatalf("hostile path leaked to request: %q", gotPath)
		}
	}
}

func TestKlingCreateVideoExtendViaModelAlias(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-extend-1","task_status":"submitted"}}`))
	})

	// DB-safe task_type stays reference_to_video; model alias selects video-extend.
	res, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:           klingModelVideoExtend,
		TaskType:        VideoTaskTypeReferenceToVideo,
		Prompt:          "continue the clip",
		Duration:        5,
		UpstreamVideoID: "kling-vid-extend-src-1",
	})
	if err != nil {
		t.Fatalf("CreateTask extend: %v", err)
	}
	if gotPath != "/v1/videos/video-extend" {
		t.Fatalf("path=%q want /v1/videos/video-extend", gotPath)
	}
	if gotBody["video_id"] != "kling-vid-extend-src-1" {
		t.Fatalf("video_id=%v", gotBody["video_id"])
	}
	if _, hasURL := gotBody["video_url"]; hasURL {
		t.Fatalf("video_url must not be sent on extend, got %v", gotBody["video_url"])
	}
	if res.UpstreamTaskID != "kt-extend-1" {
		t.Fatalf("UpstreamTaskID=%q", res.UpstreamTaskID)
	}
}

func TestKlingCreateAvatarViaModelAlias(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-avatar-1","task_status":"submitted"}}`))
	})

	res, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    klingModelAvatar,
		TaskType: VideoTaskTypeImageToVideo,
		Prompt:   "speak",
		Duration: 5,
		Content: []VideoTaskContentItem{
			{Type: VideoContentTypeText, Text: "speak"},
			{Type: VideoContentTypeImageURL, Role: VideoContentRoleFirstFrame, URL: "https://cdn.example.com/face.png"},
			{Type: VideoContentTypeAudioURL, Role: VideoContentRoleReferenceAudio, URL: "https://cdn.example.com/voice.mp3"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTask avatar: %v", err)
	}
	if gotPath != "/v1/videos/avatar" {
		t.Fatalf("path=%q want /v1/videos/avatar", gotPath)
	}
	if gotBody["image"] != "https://cdn.example.com/face.png" {
		t.Fatalf("image=%v", gotBody["image"])
	}
	if gotBody["sound_file"] != "https://cdn.example.com/voice.mp3" {
		t.Fatalf("sound_file=%v", gotBody["sound_file"])
	}
	if _, hasAudioURL := gotBody["audio_url"]; hasAudioURL {
		t.Fatalf("audio_url must not be sent on avatar, got %v", gotBody["audio_url"])
	}
	if res.UpstreamTaskID != "kt-avatar-1" {
		t.Fatalf("UpstreamTaskID=%q", res.UpstreamTaskID)
	}
}

func TestKlingCreateAvatarViaPricingSourceHint(t *testing.T) {
	var gotPath string
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-avatar-2","task_status":"submitted"}}`))
	})

	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:             "kling-v1",
		TaskType:          VideoTaskTypeImageToVideo,
		Prompt:            "speak",
		Duration:          5,
		PricingSource:     "kling_mode:avatar",
		ReferenceImageURL: "https://cdn.example.com/face.png",
	})
	if err != nil {
		t.Fatalf("CreateTask avatar hint: %v", err)
	}
	if gotPath != "/v1/videos/avatar" {
		t.Fatalf("path=%q want /v1/videos/avatar", gotPath)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestKlingSmokeGateFailureDoesNotOpenNetwork(t *testing.T) {
	called := false
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"x","task_status":"submitted"}}`))
	})
	t.Setenv("SUB2API_VIDEO_REAL_SMOKE_ENABLED", "0")

	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v1",
		TaskType: VideoTaskTypeTextToVideo,
		Prompt:   "x",
		Duration: 5,
	})
	if err == nil {
		t.Fatal("expected smoke gate block")
	}
	if called {
		t.Fatal("gate failure must not open network")
	}
	if !strings.Contains(err.Error(), "SUB2API_VIDEO_REAL_SMOKE_ENABLED") &&
		!strings.Contains(err.Error(), "VIDEO_PROVIDER_DISABLED") {
		// ErrVideoProviderDisabled message may be generic; metadata carries reason.
		t.Logf("blocked with: %v", err)
	}
}

func TestKlingRejectsUnknownModel(t *testing.T) {
	called := false
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"x","task_status":"submitted"}}`))
	})

	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-unknown-99",
		TaskType: VideoTaskTypeTextToVideo,
		Prompt:   "x",
		Duration: 5,
	})
	if err == nil {
		t.Fatal("expected unknown model rejection")
	}
	if called {
		t.Fatal("unknown model must not open network")
	}
}

func TestKlingCreatePreArmRedactionSelfCheck(t *testing.T) {
	called := false
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"x","task_status":"submitted"}}`))
	})
	// Sub-floor secret that pattern redaction cannot reliably strip → pre-arm must abort.
	acc.PlainAccessKey = "shortak"
	acc.PlainSecretKey = "shortsk"

	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v1",
		TaskType: VideoTaskTypeTextToVideo,
		Prompt:   "x",
		Duration: 5,
	})
	if err == nil {
		t.Fatal("expected pre-arm redaction self-check failure")
	}
	if called {
		t.Fatal("pre-arm failure must not open network")
	}
	if !strings.Contains(err.Error(), "REDACTION_SELF_CHECK") && !strings.Contains(err.Error(), "redaction self-check") {
		t.Fatalf("want redaction self-check error, got %v", err)
	}
}

func TestKlingCreateAbortsOnUpstreamCredentialEcho(t *testing.T) {
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"echo ` + klingTestAccessKey + `","data":{"task_id":"kt-echo","task_status":"submitted"}}`))
	})

	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v1",
		TaskType: VideoTaskTypeTextToVideo,
		Prompt:   "x",
		Duration: 5,
	})
	if err == nil {
		t.Fatal("expected echo abort")
	}
	if strings.Contains(err.Error(), klingTestAccessKey) || strings.Contains(err.Error(), klingTestSecretKey) {
		t.Fatalf("error leaked credential: %v", err)
	}
	if !strings.Contains(err.Error(), "ECHOED_CREDENTIAL") && !strings.Contains(err.Error(), "echoed") {
		t.Fatalf("want echoed-credential error, got %v", err)
	}
}

func TestKlingBuildCreatePayloadPreviewIsReal(t *testing.T) {
	adapter := &klingVideoAdapter{}
	audio := false
	payload := adapter.BuildCreatePayload(&VideoProviderAccount{BaseURL: "https://api.klingai.com"}, &VideoTask{
		Model:          "kling-3.0",
		TaskType:       VideoTaskTypeTextToVideo,
		Prompt:         "preview",
		NegativePrompt: "bad",
		AspectRatio:    "9:16",
		Duration:       10,
		GenerateAudio:  &audio,
		Resolution:     "720p",
	})
	if payload["model_name"] != "kling-v2-6" {
		t.Fatalf("model_name=%v want mapped kling-v2-6", payload["model_name"])
	}
	if payload["duration"] != "10" {
		t.Fatalf("duration=%v want string 10", payload["duration"])
	}
	if payload["mode"] != "std" {
		t.Fatalf("mode=%v", payload["mode"])
	}
	if payload["sound"] != "off" {
		t.Fatalf("sound=%v", payload["sound"])
	}
	if payload["path"] != "/v1/videos/text2video" {
		t.Fatalf("path=%v", payload["path"])
	}
	if payload["source_docs"] == "https://app.klingai.com/cn/dev/document-api/apiReference/updateNotice" && payload["prompt"] == nil {
		t.Fatal("BuildCreatePayload must preview real request fields, not skeleton docs-only")
	}
	if _, ok := payload["prompt"]; !ok {
		t.Fatalf("missing prompt in preview: %+v", payload)
	}
}

func TestKlingCancelIsLocalOnly(t *testing.T) {
	adapter := &klingVideoAdapter{}
	res, err := adapter.CancelTask(context.Background(), &VideoProviderAccount{}, &VideoTask{UpstreamTaskID: "x"})
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	if res.Status != VideoStatusCancelled {
		t.Fatalf("status=%q", res.Status)
	}
	if mode, _ := res.Payload["mode"].(string); mode != "local_cancel_only" {
		t.Fatalf("cancel payload mode=%q want local_cancel_only", mode)
	}
}

func TestKlingNormalizeStatusMap(t *testing.T) {
	adapter := &klingVideoAdapter{}
	cases := map[string]string{
		"submitted":  VideoStatusSubmitted,
		"processing": VideoStatusRunning,
		"succeed":    VideoStatusSucceeded,
		"failed":     VideoStatusFailed,
	}
	for in, want := range cases {
		if got := adapter.NormalizeStatus(in); got != want {
			t.Fatalf("NormalizeStatus(%q)=%q want %q", in, got, want)
		}
	}
}

func TestKlingDurationOnlyFiveOrTenAccepted(t *testing.T) {
	for _, d := range []int{5, 10} {
		got, err := klingDurationString(d)
		if err != nil {
			t.Fatalf("duration %d: unexpected error %v", d, err)
		}
		if got != strconv.Itoa(d) {
			t.Fatalf("duration %d: got %q", d, got)
		}
	}
	for _, d := range []int{1, 3, 4, 7, 15, 0, -1, 6} {
		_, err := klingDurationString(d)
		if err == nil {
			t.Fatalf("duration %d: expected rejection", d)
		}
		if !strings.Contains(err.Error(), "5 or 10") {
			t.Fatalf("duration %d: want clear 5/10 message, got %v", d, err)
		}
	}

	// Payload build must reject invalid durations before any network call.
	called := false
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"x","task_status":"submitted"}}`))
	})
	acc.Metadata["production_authorized"] = true
	for _, d := range []int{3, 7, 15} {
		called = false
		_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
			Model:    "kling-v1",
			TaskType: VideoTaskTypeTextToVideo,
			Prompt:   "x",
			Duration: d,
		})
		if err == nil {
			t.Fatalf("CreateTask duration=%d: expected rejection", d)
		}
		if called {
			t.Fatalf("CreateTask duration=%d must not open network", d)
		}
	}

	// Duration 10 accepted on the wire when production-authorized.
	var gotBody map[string]any
	adapter, acc, _ = newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-10","task_status":"submitted"}}`))
	})
	acc.Metadata["production_authorized"] = true
	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v1",
		TaskType: VideoTaskTypeTextToVideo,
		Prompt:   "x",
		Duration: 10,
	})
	if err != nil {
		t.Fatalf("CreateTask duration=10: %v", err)
	}
	if gotBody["duration"] != "10" {
		t.Fatalf("wire duration=%v want \"10\"", gotBody["duration"])
	}
}

func TestKlingSmokeGateDurationFiveOnlyWithoutProduction(t *testing.T) {
	called := false
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"x","task_status":"submitted"}}`))
	})
	// Smoke auth only (no production_authorized): duration 10 must be blocked.
	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v1",
		TaskType: VideoTaskTypeTextToVideo,
		Prompt:   "x",
		Duration: 10,
	})
	if err == nil {
		t.Fatal("expected smoke gate to block duration 10 without production_authorized")
	}
	if called {
		t.Fatal("smoke gate must not open network for duration 10")
	}
	reasons := klingSmokeGateBlockedReasons(acc, &VideoTask{Model: "kling-v1", Duration: 10})
	joined := strings.Join(reasons, "; ")
	if !strings.Contains(joined, "single smoke duration must be 5") {
		t.Fatalf("expected smoke duration reason, got %q", joined)
	}

	// Duration 5 with smoke auth is allowed through the gate (network may proceed).
	called = false
	adapter, acc, _ = newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-smoke-5","task_status":"submitted"}}`))
	})
	res, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v1",
		TaskType: VideoTaskTypeTextToVideo,
		Prompt:   "x",
		Duration: 5,
	})
	if err != nil {
		t.Fatalf("smoke duration 5 should be allowed: %v", err)
	}
	if !called {
		t.Fatal("expected network call for smoke duration 5")
	}
	if res.UpstreamTaskID != "kt-smoke-5" {
		t.Fatalf("UpstreamTaskID=%q", res.UpstreamTaskID)
	}

	// Duration 3 is always rejected by the gate (and by payload build).
	reasons = klingSmokeGateBlockedReasons(acc, &VideoTask{Model: "kling-v1", Duration: 3})
	joined = strings.Join(reasons, "; ")
	if !strings.Contains(joined, "kling duration must be 5 or 10") {
		t.Fatalf("expected 5-or-10 gate reason for duration 3, got %q", joined)
	}
}

func TestKlingValidateVideoGenerationContractDuration(t *testing.T) {
	if err := validateVideoGenerationContract(VideoProviderKling, "kling-v1", 5, "720p"); err != nil {
		t.Fatalf("duration 5: %v", err)
	}
	if err := validateVideoGenerationContract(VideoProviderKling, "kling-v1", 10, "720p"); err != nil {
		t.Fatalf("duration 10: %v", err)
	}
	for _, d := range []int{1, 3, 7, 15, 60, -1} {
		if err := validateVideoGenerationContract(VideoProviderKling, "kling-v1", d, "720p"); err == nil {
			t.Fatalf("duration %d: expected contract rejection", d)
		}
	}
}

func TestKlingCreateWritesRedactedAuditLog(t *testing.T) {
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"token ` + klingTestSecretKey + `","data":{"task_id":"kt-audit","task_status":"submitted"}}`))
	})
	// Echo of SK must abort after audit; verify audit file has no secret.
	_, _ = adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    "kling-v1",
		TaskType: VideoTaskTypeTextToVideo,
		Prompt:   "x",
		Duration: 5,
	})
	raw, err := os.ReadFile(os.Getenv("SUB2API_VIDEO_REDACTED_EVENT_LOG"))
	if err != nil || len(raw) == 0 {
		t.Fatalf("expected audit log, err=%v len=%d", err, len(raw))
	}
	if strings.Contains(string(raw), klingTestAccessKey) || strings.Contains(string(raw), klingTestSecretKey) {
		t.Fatalf("audit log leaked credential: %s", raw)
	}
	if !strings.Contains(string(raw), `"provider":"kling"`) && !strings.Contains(string(raw), `"provider": "kling"`) {
		t.Fatalf("audit log should record kling provider: %s", raw)
	}
}

func TestKlingCreateVideoExtendRejectsHTTPURLWithoutID(t *testing.T) {
	called := false
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"x","task_status":"submitted"}}`))
	})
	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:             klingModelVideoExtend,
		TaskType:          VideoTaskTypeReferenceToVideo,
		Prompt:            "continue",
		Duration:          5,
		ReferenceVideoURL: "https://cdn.example.com/clip.mp4",
	})
	if err == nil {
		t.Fatal("expected KLING_MISSING_VIDEO_ID")
	}
	if !strings.Contains(err.Error(), "KLING_MISSING_VIDEO_ID") && !strings.Contains(err.Error(), "upstream_video_id") {
		t.Fatalf("want missing video id error, got %v", err)
	}
	if called {
		t.Fatal("must not open network without video_id")
	}
}

func TestKlingCreateAvatarAudioIDXorSoundFile(t *testing.T) {
	var gotBody map[string]any
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"task_id":"kt-avatar-id-1","task_status":"submitted"}}`))
	})
	_, err := adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    klingModelAvatar,
		TaskType: VideoTaskTypeImageToVideo,
		Prompt:   "speak",
		Duration: 5,
		AudioID:  "kling-audio-1",
		Content: []VideoTaskContentItem{
			{Type: VideoContentTypeText, Text: "speak"},
			{Type: VideoContentTypeImageURL, Role: VideoContentRoleFirstFrame, URL: "https://cdn.example.com/face.png"},
		},
	})
	if err != nil {
		t.Fatalf("CreateTask avatar audio_id: %v", err)
	}
	if gotBody["audio_id"] != "kling-audio-1" {
		t.Fatalf("audio_id=%v", gotBody["audio_id"])
	}
	if _, has := gotBody["sound_file"]; has {
		t.Fatalf("sound_file must be absent when audio_id set, got %v", gotBody["sound_file"])
	}

	_, err = adapter.CreateTask(context.Background(), acc, &VideoTask{
		Model:    klingModelAvatar,
		TaskType: VideoTaskTypeImageToVideo,
		Prompt:   "speak",
		Duration: 5,
		AudioID:  "kling-audio-1",
		Content: []VideoTaskContentItem{
			{Type: VideoContentTypeText, Text: "speak"},
			{Type: VideoContentTypeImageURL, Role: VideoContentRoleFirstFrame, URL: "https://cdn.example.com/face.png"},
			{Type: VideoContentTypeAudioURL, Role: VideoContentRoleReferenceAudio, URL: "https://cdn.example.com/voice.mp3"},
		},
	})
	if err == nil {
		t.Fatal("expected audio_id/sound_file conflict")
	}
	if !strings.Contains(err.Error(), "KLING_AVATAR_AUDIO_CONFLICT") && !strings.Contains(err.Error(), "audio_id") {
		t.Fatalf("want conflict error, got %v", err)
	}
}

func TestKlingPollPersistsUpstreamVideoID(t *testing.T) {
	adapter, acc, _ := newSmokeGatedKlingFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"task_id":"kt-poll-1","task_status":"succeed","task_result":{"videos":[{"id":"kling-vid-out-9","url":"https://cdn.example.com/out.mp4"}]}}}`))
	})
	res, err := adapter.PollTask(context.Background(), acc, &VideoTask{
		Model:          "kling-v1",
		TaskType:       VideoTaskTypeTextToVideo,
		Duration:       5,
		UpstreamTaskID: "kt-poll-1",
	})
	if err != nil {
		t.Fatalf("PollTask: %v", err)
	}
	if res.UpstreamVideoID != "kling-vid-out-9" {
		t.Fatalf("UpstreamVideoID=%q", res.UpstreamVideoID)
	}
	if res.ResultURL != "https://cdn.example.com/out.mp4" {
		t.Fatalf("ResultURL=%q", res.ResultURL)
	}
}

func TestKlingOmniDurationAllowsWiderRange(t *testing.T) {
	if err := validateVideoGenerationContract(VideoProviderKling, "kling-3.0-omni", 8, "720p"); err != nil {
		t.Fatalf("omni duration 8 should pass: %v", err)
	}
	if err := validateVideoGenerationContract(VideoProviderKling, "kling-v1", 8, "720p"); err == nil {
		t.Fatal("non-omni duration 8 must fail")
	}
	got, err := klingDurationStringForTask(&VideoTask{Model: "kling-3.0-omni", Duration: 8, TaskType: VideoTaskTypeReferenceToVideo})
	if err != nil {
		t.Fatalf("klingDurationStringForTask: %v", err)
	}
	if got != "8" {
		t.Fatalf("got %q want 8", got)
	}
}
