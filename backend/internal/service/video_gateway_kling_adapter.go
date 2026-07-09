package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	klingDefaultBaseURL = "https://api.klingai.com"

	klingPathText2Video       = "/v1/videos/text2video"
	klingPathImage2Video      = "/v1/videos/image2video"
	klingPathMultiImage2Video = "/v1/videos/multi-image2video"
	klingPathOmniVideo        = "/v1/videos/omni-video"
	klingPathVideoExtend      = "/v1/videos/video-extend"
	klingPathAvatar           = "/v1/videos/avatar"

	// Catalog / routing aliases that select extend/avatar endpoints while keeping
	// DB task_type in {text_to_video, image_to_video, reference_to_video}
	// (migration 136 CHECK). Detection is via model name or PricingSource /
	// ErrorMessage-free routing hints — see klingEndpointMode.
	klingModelVideoExtend = "kling-video-extend"
	klingModelAvatar      = "kling-avatar"
	klingModelLipSync     = "kling-lip-sync"
)

// klingSafeUpstreamTaskID matches Kling-realistic task ids we are willing to
// persist and interpolate into poll URL path segments. Rejects path traversal,
// query/fragment injection, and whitespace.
var klingSafeUpstreamTaskID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

type klingVideoAdapter struct{}

func (a *klingVideoAdapter) Provider() string { return VideoProviderKling }

func (a *klingVideoAdapter) CreateTask(ctx context.Context, account *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error) {
	if !account.APIKeyConfigured || !videoProviderHasPlainCredentials(account) {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"provider": "kling",
			"reason":   "api key is not configured; real call skipped",
		})
	}
	if reasons := klingSmokeGateBlockedReasons(account, task); len(reasons) > 0 {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"provider":           "kling",
			"reason":             strings.Join(reasons, "; "),
			"real_call_executed": "false",
		})
	}
	if err := klingPreArmRedactionSelfCheck(account); err != nil {
		return nil, err
	}

	modelName, err := mapKlingUpstreamModelName(task.Model)
	if err != nil {
		return nil, err
	}
	path, err := klingCreatePath(task, modelName)
	if err != nil {
		return nil, err
	}
	payload, err := buildKlingCreateRequestPayload(account, task, modelName, path)
	if err != nil {
		return nil, err
	}

	token, err := klingMintJWT(account.PlainAccessKey, account.PlainSecretKey)
	if err != nil {
		return nil, infraerrorsUnavailable("KLING_JWT_MINT_FAILED", "Kling JWT mint failed (credential values intentionally not included)")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("kling: marshal create payload: %w", err)
	}

	url := strings.TrimRight(klingBaseURL(account), "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("kling: build create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, infraerrorsUnavailable("KLING_CREATE_HTTP_ERROR", "Kling create task HTTP error: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("kling: read create response: %w", err)
	}

	if auditErr := appendRedactedVideoEventForAccount(account, "create", resp.StatusCode, string(respBody)); auditErr != nil {
		return nil, infraerrorsUnavailable("KLING_AUDIT_LOG_FAILED", "Kling create: redacted audit log unavailable: "+auditErr.Error())
	}
	if videoProviderUpstreamEchoedCredential(account, string(respBody)) {
		return nil, infraerrorsUnavailable("KLING_UPSTREAM_ECHOED_CREDENTIAL",
			"Kling create aborted: upstream response echoed the configured credential in a stored field; refusing to persist it (key value intentionally not included)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, infraerrorsUnavailable("KLING_CREATE_UPSTREAM_ERROR",
			fmt.Sprintf("Kling create task returned %d (%s): %s", resp.StatusCode, klingHTTPErrorType(resp.StatusCode, string(respBody)), truncate(klingRedactBody(account, string(respBody)), 500)))
	}

	parsed, err := parseKlingEnvelope(respBody)
	if err != nil {
		return nil, fmt.Errorf("kling: parse create response: %w", err)
	}
	if videoProviderUpstreamEchoedCredential(account, parsed.Data.TaskID) {
		return nil, infraerrorsUnavailable("KLING_UPSTREAM_ECHOED_CREDENTIAL",
			"Kling create aborted: upstream id field echoed the configured credential; refusing to persist it (key value intentionally not included)")
	}
	if parsed.Code != 0 {
		msg := firstNonEmptyVideo(parsed.Message, parsed.Data.TaskStatusMsg)
		return nil, infraerrorsUnavailable("KLING_CREATE_BUSINESS_ERROR", "Kling: "+klingRedactBody(account, msg))
	}
	taskID, err := sanitizeKlingUpstreamTaskID(parsed.Data.TaskID)
	if err != nil {
		return nil, err
	}

	status := a.NormalizeStatus(parsed.Data.TaskStatus)
	return &VideoAdapterResult{
		UpstreamTaskID: taskID,
		Status:         status,
		Payload: map[string]any{
			"provider":          "kling",
			"upstream_id":       taskID,
			"model":             modelName,
			"path":              path,
			"normalized_status": status,
			"qcanvas_status":    qcanvasVideoContractStatus(status, ""),
			"redacted_event":    true,
			"created_at":        time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

func (a *klingVideoAdapter) PollTask(ctx context.Context, account *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error) {
	if !account.APIKeyConfigured || !videoProviderHasPlainCredentials(account) {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"provider": "kling",
			"reason":   "api key is not configured; poll skipped",
		})
	}
	if reasons := klingSmokeGateBlockedReasons(account, task); len(reasons) > 0 {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"provider":           "kling",
			"reason":             strings.Join(reasons, "; "),
			"real_call_executed": "false",
		})
	}
	if err := klingPreArmRedactionSelfCheck(account); err != nil {
		return nil, err
	}
	taskID, err := sanitizeKlingUpstreamTaskID(task.UpstreamTaskID)
	if err != nil {
		return nil, err
	}

	modelName, err := mapKlingUpstreamModelName(task.Model)
	if err != nil {
		return nil, err
	}
	path, err := klingCreatePath(task, modelName)
	if err != nil {
		return nil, err
	}

	token, err := klingMintJWT(account.PlainAccessKey, account.PlainSecretKey)
	if err != nil {
		return nil, infraerrorsUnavailable("KLING_JWT_MINT_FAILED", "Kling JWT mint failed (credential values intentionally not included)")
	}

	// Re-validated charset above; PathEscape is defense-in-depth so a future
	// charset relaxation cannot reopen path injection.
	url := strings.TrimRight(klingBaseURL(account), "/") + path + "/" + url.PathEscape(taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("kling: build poll request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, infraerrorsUnavailable("KLING_POLL_HTTP_ERROR", "Kling poll task HTTP error: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("kling: read poll response: %w", err)
	}

	if auditErr := appendRedactedVideoEventForAccount(account, "poll", resp.StatusCode, string(respBody)); auditErr != nil {
		return nil, infraerrorsUnavailable("KLING_AUDIT_LOG_FAILED", "Kling poll: redacted audit log unavailable: "+auditErr.Error())
	}
	if videoProviderUpstreamEchoedCredential(account, string(respBody)) {
		return nil, infraerrorsUnavailable("KLING_UPSTREAM_ECHOED_CREDENTIAL",
			"Kling poll aborted: upstream response echoed the configured credential in a stored field; refusing to persist it (key value intentionally not included)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, infraerrorsUnavailable("KLING_POLL_UPSTREAM_ERROR",
			fmt.Sprintf("Kling poll task returned %d (%s): %s", resp.StatusCode, klingHTTPErrorType(resp.StatusCode, string(respBody)), truncate(klingRedactBody(account, string(respBody)), 500)))
	}

	parsed, err := parseKlingEnvelope(respBody)
	if err != nil {
		return nil, fmt.Errorf("kling: parse poll response: %w", err)
	}
	resultURL := ""
	if len(parsed.Data.TaskResult.Videos) > 0 {
		resultURL = strings.TrimSpace(parsed.Data.TaskResult.Videos[0].URL)
	}
	if videoProviderUpstreamEchoedCredential(account, parsed.Data.TaskID) ||
		videoProviderUpstreamEchoedCredential(account, resultURL) {
		return nil, infraerrorsUnavailable("KLING_UPSTREAM_ECHOED_CREDENTIAL",
			"Kling poll aborted: upstream id/result_url echoed the configured credential; refusing to persist it (key value intentionally not included)")
	}

	status := a.NormalizeStatus(parsed.Data.TaskStatus)
	polledID := taskID
	if strings.TrimSpace(parsed.Data.TaskID) != "" {
		if safe, idErr := sanitizeKlingUpstreamTaskID(parsed.Data.TaskID); idErr == nil {
			polledID = safe
		}
	}
	result := &VideoAdapterResult{
		UpstreamTaskID: polledID,
		Status:         status,
		Payload: map[string]any{
			"provider":          "kling",
			"normalized_status": status,
			"qcanvas_status":    qcanvasVideoContractStatus(status, parsed.Message),
			"redacted_event":    true,
			"path":              path,
			"polled_at":         time.Now().UTC().Format(time.RFC3339),
		},
	}
	if parsed.Code != 0 {
		result.Status = VideoStatusFailed
		result.ErrorMessage = klingRedactBody(account, firstNonEmptyVideo(parsed.Message, parsed.Data.TaskStatusMsg))
		return result, nil
	}
	if resultURL != "" {
		if err := validateExternalVideoURL(resultURL); err != nil {
			result.Status = VideoStatusFailed
			result.ErrorMessage = "upstream result_url failed validation: " + klingRedactBody(account, err.Error())
			result.Payload["result_url_rejected"] = true
			return result, nil
		}
		result.ResultURL = resultURL
	}
	if msg := strings.TrimSpace(parsed.Data.TaskStatusMsg); msg != "" && status == VideoStatusFailed {
		result.ErrorMessage = klingRedactBody(account, msg)
	}
	return result, nil
}

// CancelTask is local-only: Kling has no reliable public cancel for these video
// paths in our integration surface, so we mark cancelled locally and never fake
// an upstream cancel call.
func (a *klingVideoAdapter) CancelTask(_ context.Context, _ *VideoProviderAccount, _ *VideoTask) (*VideoAdapterResult, error) {
	return &VideoAdapterResult{
		Status: VideoStatusCancelled,
		Payload: map[string]any{
			"provider": "kling",
			"mode":     "local_cancel_only",
			"note":     "no upstream cancel; local status only",
		},
	}, nil
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
	modelName, err := mapKlingUpstreamModelName(task.Model)
	if err != nil {
		return map[string]any{
			"base_url":            klingBaseURL(account),
			"model":               task.Model,
			"error":               err.Error(),
			"smoke_gate_required": true,
		}
	}
	path, pathErr := klingCreatePath(task, modelName)
	if pathErr != nil {
		path = ""
	}
	payload, buildErr := buildKlingCreateRequestPayload(account, task, modelName, path)
	if buildErr != nil {
		return map[string]any{
			"base_url":            klingBaseURL(account),
			"model_name":          modelName,
			"path":                path,
			"error":               buildErr.Error(),
			"smoke_gate_required": true,
		}
	}
	payload["base_url"] = klingBaseURL(account)
	payload["path"] = path
	payload["smoke_gate_required"] = true
	payload["redacted_event_log_required"] = true
	payload["asset_persistable"] = false
	payload["local_cancel_only"] = true
	return payload
}

func klingBaseURL(account *VideoProviderAccount) string {
	if account != nil && strings.TrimSpace(account.BaseURL) != "" {
		return strings.TrimSpace(account.BaseURL)
	}
	return klingDefaultBaseURL
}

func mapKlingUpstreamModelName(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "kling-v1":
		return "kling-v1", nil
	case "kling-2.6-pro", "kling-v2-6":
		return "kling-v2-6", nil
	case "kling-3.0":
		return "kling-v2-6", nil
	case "kling-3.0-omni":
		return "kling-v3-omni", nil
	case "kling-o1":
		return "kling-video-o1", nil
	// Routing aliases (not sent as upstream model_name for extend/avatar —
	// those endpoints use a base generation model; we map to kling-v1).
	case klingModelVideoExtend, klingModelAvatar, klingModelLipSync, "kling-lipsync":
		return "kling-v1", nil
	case "":
		return "", infraerrors.BadRequest("KLING_MODEL_REQUIRED", "kling model is required")
	default:
		return "", infraerrors.BadRequest("KLING_MODEL_NOT_ALLOWED", "kling model is not in the fail-closed allowlist: "+strings.TrimSpace(raw))
	}
}

func klingUsesOmniEndpoint(modelName string) bool {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "kling-v3-omni", "kling-video-o1":
		return true
	default:
		return false
	}
}

// klingEndpointMode selects the official path without expanding DB task_type
// beyond migration 136's CHECK. Prefer (in order):
//  1. catalog model aliases: kling-video-extend / kling-avatar / kling-lip-sync
//  2. PricingSource hint: "kling_mode:video_extend" | "kling_mode:avatar" | "kling_mode:lip_sync"
//  3. standard task_type → text2video / image2video / multi-image2video / omni
func klingEndpointMode(task *VideoTask) string {
	if task == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(task.Model)) {
	case klingModelVideoExtend:
		return "video_extend"
	case klingModelAvatar, klingModelLipSync, "kling-lipsync":
		return "avatar"
	}
	hint := strings.ToLower(strings.TrimSpace(task.PricingSource))
	switch {
	case hint == "kling_mode:video_extend" || hint == "video_extend":
		return "video_extend"
	case hint == "kling_mode:avatar" || hint == "kling_mode:lip_sync" || hint == "avatar" || hint == "lip_sync" || hint == "lip-sync":
		return "avatar"
	}
	return ""
}

func klingCreatePath(task *VideoTask, modelName string) (string, error) {
	if task == nil {
		return "", infraerrors.BadRequest("KLING_TASK_REQUIRED", "kling task is required")
	}
	switch klingEndpointMode(task) {
	case "video_extend":
		return klingPathVideoExtend, nil
	case "avatar":
		return klingPathAvatar, nil
	}

	taskType := strings.ToLower(strings.TrimSpace(task.TaskType))
	switch taskType {
	case VideoTaskTypeTextToVideo, "":
		if klingUsesOmniEndpoint(modelName) {
			return klingPathOmniVideo, nil
		}
		return klingPathText2Video, nil
	case VideoTaskTypeImageToVideo:
		if klingUsesOmniEndpoint(modelName) {
			return klingPathOmniVideo, nil
		}
		return klingPathImage2Video, nil
	case VideoTaskTypeReferenceToVideo:
		if klingUsesOmniEndpoint(modelName) {
			return klingPathOmniVideo, nil
		}
		return klingPathMultiImage2Video, nil
	default:
		// Do not accept video_extend/avatar as task_type — DB CHECK forbids storing them.
		return "", infraerrors.BadRequest("KLING_UNSUPPORTED_TASK_TYPE",
			"unsupported kling task_type: "+taskType+"; use model kling-video-extend/kling-avatar or PricingSource kling_mode:* (DB task_type must stay text/image/reference)")
	}
}

// sanitizeKlingUpstreamTaskID rejects empty/hostile task ids before they are
// persisted or interpolated into poll URLs.
func sanitizeKlingUpstreamTaskID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", infraerrors.BadRequest("KLING_INVALID_UPSTREAM_TASK_ID", "kling upstream task_id is empty")
	}
	if !klingSafeUpstreamTaskID.MatchString(id) {
		return "", infraerrors.BadRequest("KLING_INVALID_UPSTREAM_TASK_ID",
			"kling upstream task_id failed charset validation (allowed: A-Za-z0-9_-, max 128)")
	}
	return id, nil
}

func buildKlingCreateRequestPayload(account *VideoProviderAccount, task *VideoTask, modelName, path string) (map[string]any, error) {
	if task == nil {
		return nil, infraerrors.BadRequest("KLING_TASK_REQUIRED", "kling task is required")
	}
	payload := map[string]any{
		"model_name": modelName,
		"prompt":     task.Prompt,
	}
	if strings.TrimSpace(task.NegativePrompt) != "" {
		payload["negative_prompt"] = task.NegativePrompt
	}
	durationStr, err := klingDurationString(task.Duration)
	if err != nil {
		return nil, err
	}
	payload["duration"] = durationStr
	if ratio := strings.TrimSpace(task.AspectRatio); ratio != "" {
		payload["aspect_ratio"] = ratio
	}
	if mode := klingModeFromTask(task); mode != "" {
		payload["mode"] = mode
	}
	if sound := klingSoundFromTask(task); sound != "" {
		payload["sound"] = sound
	}

	switch path {
	case klingPathImage2Video:
		imageURL, tailURL, err := klingResolveImagePair(account, task)
		if err != nil {
			return nil, err
		}
		if imageURL == "" {
			return nil, infraerrors.BadRequest("KLING_MISSING_IMAGE", "image_to_video requires first_frame image or reference_image_url")
		}
		payload["image"] = imageURL
		if tailURL != "" {
			payload["image_tail"] = tailURL
		}
	case klingPathMultiImage2Video:
		images, err := klingCollectReferenceImages(account, task)
		if err != nil {
			return nil, err
		}
		if len(images) == 0 {
			return nil, infraerrors.BadRequest("KLING_MISSING_IMAGE", "reference_to_video requires reference images")
		}
		list := make([]map[string]any, 0, len(images))
		for _, u := range images {
			list = append(list, map[string]any{"image": u})
		}
		payload["image_list"] = list
	case klingPathOmniVideo:
		images, err := klingCollectReferenceImages(account, task)
		if err != nil {
			return nil, err
		}
		if len(images) > 0 {
			list := make([]map[string]any, 0, len(images))
			for _, u := range images {
				list = append(list, map[string]any{"image_url": u})
			}
			payload["image_list"] = list
		}
		if task.ReferenceVideoURL != "" {
			if err := validateExternalVideoURL(task.ReferenceVideoURL); err != nil {
				return nil, infraerrors.BadRequest("VIDEO_UNSAFE_REFERENCE_URL",
					"reference_video_url failed SSRF/allowlist validation: "+klingRedactBody(account, err.Error()))
			}
			payload["video_list"] = []map[string]any{{"video_url": task.ReferenceVideoURL}}
		}
	case klingPathVideoExtend:
		videoURL := strings.TrimSpace(task.ReferenceVideoURL)
		if videoURL == "" {
			for _, item := range task.Content {
				if item.Type == VideoContentTypeVideoURL && strings.TrimSpace(item.URL) != "" {
					videoURL = strings.TrimSpace(item.URL)
					break
				}
			}
		}
		if videoURL == "" {
			return nil, infraerrors.BadRequest("KLING_MISSING_VIDEO", "video_extend requires reference_video_url")
		}
		if err := validateExternalVideoURL(videoURL); err != nil {
			return nil, infraerrors.BadRequest("VIDEO_UNSAFE_REFERENCE_URL",
				"reference_video_url failed SSRF/allowlist validation: "+klingRedactBody(account, err.Error()))
		}
		payload["video_url"] = videoURL
	case klingPathAvatar:
		imageURL, _, err := klingResolveImagePair(account, task)
		if err != nil {
			return nil, err
		}
		if imageURL != "" {
			payload["image"] = imageURL
		}
		audioURL := ""
		for _, item := range task.Content {
			if item.Type == VideoContentTypeAudioURL && strings.TrimSpace(item.URL) != "" {
				audioURL = strings.TrimSpace(item.URL)
				break
			}
		}
		if audioURL != "" {
			if err := validateExternalVideoURL(audioURL); err != nil {
				return nil, infraerrors.BadRequest("VIDEO_UNSAFE_REFERENCE_URL",
					"audio_url failed SSRF/allowlist validation: "+klingRedactBody(account, err.Error()))
			}
			payload["audio_url"] = audioURL
		}
	}
	return payload, nil
}

func klingModeFromTask(task *VideoTask) string {
	if task == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(task.Resolution)) {
	case "1080p", "pro":
		return "pro"
	case "720p", "std", "480p":
		return "std"
	}
	// Catalog ids that imply pro quality.
	switch strings.ToLower(strings.TrimSpace(task.Model)) {
	case "kling-2.6-pro":
		return "pro"
	}
	return "std"
}

func klingSoundFromTask(task *VideoTask) string {
	if task == nil || task.GenerateAudio == nil {
		return ""
	}
	if *task.GenerateAudio {
		return "on"
	}
	return "off"
}

func klingResolveImagePair(account *VideoProviderAccount, task *VideoTask) (imageURL, tailURL string, err error) {
	for _, item := range task.Content {
		if item.Type != VideoContentTypeImageURL {
			continue
		}
		u := strings.TrimSpace(item.URL)
		if u == "" {
			continue
		}
		if err := validateExternalVideoURL(u); err != nil {
			return "", "", infraerrors.BadRequest("VIDEO_UNSAFE_REFERENCE_URL",
				"content.image_url failed SSRF/allowlist validation: "+klingRedactBody(account, err.Error()))
		}
		switch item.Role {
		case VideoContentRoleFirstFrame:
			if imageURL == "" {
				imageURL = u
			}
		case VideoContentRoleLastFrame:
			if tailURL == "" {
				tailURL = u
			}
		case VideoContentRoleReferenceImage:
			if imageURL == "" {
				imageURL = u
			}
		default:
			if imageURL == "" {
				imageURL = u
			}
		}
	}
	if imageURL == "" && strings.TrimSpace(task.ReferenceImageURL) != "" {
		u := strings.TrimSpace(task.ReferenceImageURL)
		if err := validateExternalVideoURL(u); err != nil {
			return "", "", infraerrors.BadRequest("VIDEO_UNSAFE_REFERENCE_URL",
				"reference_image_url failed SSRF/allowlist validation: "+klingRedactBody(account, err.Error()))
		}
		imageURL = u
	}
	return imageURL, tailURL, nil
}

func klingCollectReferenceImages(account *VideoProviderAccount, task *VideoTask) ([]string, error) {
	out := make([]string, 0, 4)
	seen := map[string]struct{}{}
	add := func(u string) error {
		u = strings.TrimSpace(u)
		if u == "" {
			return nil
		}
		if err := validateExternalVideoURL(u); err != nil {
			return infraerrors.BadRequest("VIDEO_UNSAFE_REFERENCE_URL",
				"reference image failed SSRF/allowlist validation: "+klingRedactBody(account, err.Error()))
		}
		if _, ok := seen[u]; ok {
			return nil
		}
		seen[u] = struct{}{}
		out = append(out, u)
		return nil
	}
	for _, item := range task.Content {
		if item.Type != VideoContentTypeImageURL {
			continue
		}
		if err := add(item.URL); err != nil {
			return nil, err
		}
	}
	if len(out) == 0 {
		if err := add(task.ReferenceImageURL); err != nil {
			return nil, err
		}
	}
	return out, nil
}

type klingEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID        string `json:"task_id"`
		TaskStatus    string `json:"task_status"`
		TaskStatusMsg string `json:"task_status_msg"`
		TaskResult    struct {
			Videos []struct {
				URL string `json:"url"`
			} `json:"videos"`
		} `json:"task_result"`
	} `json:"data"`
}

func parseKlingEnvelope(raw []byte) (*klingEnvelope, error) {
	var parsed klingEnvelope
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func klingRedactBody(account *VideoProviderAccount, s string) string {
	return redactVideoUpstreamSecretsForAccount(account, s)
}

func klingPreArmRedactionSelfCheck(account *VideoProviderAccount) error {
	if account == nil {
		return nil
	}
	keys := make([]string, 0, 3)
	for _, raw := range []string{account.PlainAccessKey, account.PlainSecretKey} {
		if k := strings.TrimSpace(raw); k != "" {
			keys = append(keys, k)
		}
	}
	if token, err := klingMintJWT(account.PlainAccessKey, account.PlainSecretKey); err == nil && token != "" {
		keys = append(keys, token)
	}
	for _, key := range keys {
		for _, probe := range []string{
			key,
			"Bearer " + key,
			`{"message":"` + key + `"}`,
			`{"error":{"message":"rejected token ` + key + `"}}`,
		} {
			if strings.Contains(klingRedactBody(account, probe), key) {
				return infraerrorsUnavailable("KLING_REDACTION_SELF_CHECK_FAILED",
					"pre-arm redaction self-check failed: the configured key is not stripped in every upstream echo shape; "+
						"aborting before any billed call (key value intentionally not included)")
			}
		}
	}
	return nil
}

func klingSmokeGateBlockedReasons(account *VideoProviderAccount, task *VideoTask) []string {
	reasons := []string{}
	productionAuthorized := klingProductionAuthorized(account)
	if strings.TrimSpace(os.Getenv("SUB2API_VIDEO_REAL_SMOKE_ENABLED")) != "1" {
		reasons = append(reasons, "SUB2API_VIDEO_REAL_SMOKE_ENABLED is not 1")
	}
	if !klingRealCallAuthorized(account) {
		reasons = append(reasons, "provider metadata does not record single smoke authorization or production authorization")
	}
	if strings.TrimSpace(os.Getenv("SUB2API_VIDEO_REDACTED_EVENT_LOG")) == "" {
		reasons = append(reasons, "redacted event log path is missing")
	}
	if configuredMediaURLAllowlist() == "" {
		reasons = append(reasons, mediaURLAllowlistMissingReason())
	}
	if task == nil || strings.TrimSpace(task.Model) == "" {
		reasons = append(reasons, "kling model is not explicit")
	} else if _, err := mapKlingUpstreamModelName(task.Model); err != nil {
		reasons = append(reasons, "kling model is not in the fail-closed allowlist")
	}
	duration := 0
	if task != nil {
		duration = task.Duration
	}
	if duration != 5 && duration != 10 {
		reasons = append(reasons, "kling duration must be 5 or 10 seconds")
	} else if !productionAuthorized && duration != 5 {
		reasons = append(reasons, "single smoke duration must be 5 seconds")
	}
	return reasons
}

// klingDurationString maps task.Duration to the official Kling string enum.
// Only "5" and "10" are accepted; all other values fail closed at payload build.
func klingDurationString(duration int) (string, error) {
	switch duration {
	case 5, 10:
		return strconv.Itoa(duration), nil
	default:
		return "", infraerrors.BadRequest("KLING_INVALID_DURATION", "kling duration must be 5 or 10 seconds")
	}
}

func klingRealCallAuthorized(account *VideoProviderAccount) bool {
	if account == nil {
		return false
	}
	return metadataBool(account.Metadata, "single_smoke_authorized") ||
		metadataBool(account.Metadata, "real_smoke_authorized") ||
		klingProductionAuthorized(account)
}

func klingProductionAuthorized(account *VideoProviderAccount) bool {
	if account == nil {
		return false
	}
	return metadataBool(account.Metadata, "production_authorized")
}

func klingHTTPErrorType(statusCode int, body string) string {
	return seedanceHTTPErrorType(statusCode, body)
}
