package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	SeedanceModel               = "doubao-seedance-2-0-260128"
	SeedanceBaseURL             = "https://ark.cn-beijing.volces.com/api/v3"
	HCAtomSeedanceV3Provider    = "hc_atom_seedance_v3"
	HCAtomSeedanceV3BaseURL     = "https://api-aigc.fzyinghe.com"
	HCAtomSeedanceV3Host        = "api-aigc.fzyinghe.com"
	HCAtomSeedanceV3Path        = "/v3/video/tasks"
	HCAtomSeedanceV3PublicModel = "doubao-seedance-2.0"
	HCAtomSeedanceV3Model       = "doubao-seedance-2-0-260128"
)

var (
	ErrVideoRealDispatchDenied   = errors.New("real video dispatch denied: single_smoke_authorized is required")
	ErrVideoRealDispatchConsumed = errors.New("real video dispatch denied: single_smoke_authorized was already consumed")
)

type VideoProviderTransportError struct{ err error }

func (e *VideoProviderTransportError) Error() string { return e.err.Error() }
func (e *VideoProviderTransportError) Unwrap() error { return e.err }

type SingleSmokeAuthorization struct {
	mu                sync.Mutex
	allowed, consumed bool
}

func NewSingleSmokeAuthorization(allowed bool) *SingleSmokeAuthorization {
	return &SingleSmokeAuthorization{allowed: allowed}
}

func (a *SingleSmokeAuthorization) Consume() error {
	if a == nil || !a.allowed {
		return ErrVideoRealDispatchDenied
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.consumed {
		return ErrVideoRealDispatchConsumed
	}
	a.consumed = true
	return nil
}

func (a *SingleSmokeAuthorization) Allowed() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.allowed && !a.consumed
}

type VideoCreateRequest struct {
	Model             string             `json:"model,omitempty"`
	Prompt            string             `json:"prompt"`
	Content           []VideoContentItem `json:"content,omitempty"`
	Ratio             string             `json:"ratio,omitempty"`
	GenerateAudio     bool               `json:"generate_audio,omitempty"`
	Duration          int                `json:"duration"`
	Resolution        string             `json:"resolution"`
	ReferenceImageURL string             `json:"reference_image_url,omitempty"`
	ReturnLastFrame   bool               `json:"return_last_frame"`
	Watermark         bool               `json:"watermark,omitempty"`
}

type VideoContentURL struct {
	URL string `json:"url"`
}
type VideoContentItem struct {
	Type     string           `json:"type"`
	Text     string           `json:"text,omitempty"`
	ImageURL *VideoContentURL `json:"image_url,omitempty"`
	VideoURL *VideoContentURL `json:"video_url,omitempty"`
	AudioURL *VideoContentURL `json:"audio_url,omitempty"`
}

// VideoProviderClient keeps dispatch protocol-neutral while retaining the
// official Ark adapter as a compatible implementation.
type VideoProviderClient interface {
	Create(context.Context, VideoCreateRequest) (*VideoProviderTask, error)
	Poll(context.Context, string) (*VideoProviderTask, error)
	Cancel(context.Context, string) (*VideoProviderTask, error)
}

func ValidateTinyRealContract(r VideoCreateRequest) error {
	if strings.TrimSpace(r.Prompt) == "" {
		return errors.New("video prompt is required")
	}
	if r.Duration != 4 {
		return errors.New("tiny_real duration must be 4 seconds")
	}
	if r.Resolution != "720p" {
		return errors.New("tiny_real resolution must be 720p")
	}
	if strings.TrimSpace(r.ReferenceImageURL) != "" {
		return errors.New("tiny_real does not accept reference images")
	}
	if len(r.Content) != 0 || strings.TrimSpace(r.Ratio) != "" || r.GenerateAudio || r.Watermark {
		return errors.New("tiny_real accepts prompt-only requests")
	}
	return nil
}

func (a *SeedanceAdapter) Cancel(context.Context, string) (*VideoProviderTask, error) {
	return nil, errors.New("seedance upstream cancel is unsupported")
}

type HCAtomV3Adapter struct {
	client          *http.Client
	baseURL, apiKey string
}

func NewHCAtomV3Adapter(client *http.Client, baseURL, apiKey string) *HCAtomV3Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HCAtomV3Adapter{client: client, baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), apiKey: apiKey}
}

func validateHCAtomV3BaseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.ToLower(u.Hostname()) != HCAtomSeedanceV3Host || strings.TrimRight(u.Path, "/") != "" {
		return nil, errors.New("hc atom endpoint is not allowlisted")
	}
	return u, nil
}

func normalizeHCAtomV3Content(input VideoCreateRequest) ([]VideoContentItem, error) {
	content := input.Content
	if len(content) == 0 && strings.TrimSpace(input.Prompt) != "" {
		content = []VideoContentItem{{Type: "text", Text: strings.TrimSpace(input.Prompt)}}
	}
	if len(content) == 0 {
		return nil, errors.New("hc atom content is required")
	}
	for _, item := range content {
		switch item.Type {
		case "text":
			if strings.TrimSpace(item.Text) == "" || item.ImageURL != nil || item.VideoURL != nil || item.AudioURL != nil {
				return nil, errors.New("hc atom text content is invalid")
			}
		case "image_url", "video_url", "audio_url":
			var media *VideoContentURL
			if item.Type == "image_url" {
				media = item.ImageURL
			}
			if item.Type == "video_url" {
				media = item.VideoURL
			}
			if item.Type == "audio_url" {
				media = item.AudioURL
			}
			if media == nil || strings.TrimSpace(item.Text) != "" || (item.Type != "image_url" && item.ImageURL != nil) || (item.Type != "video_url" && item.VideoURL != nil) || (item.Type != "audio_url" && item.AudioURL != nil) || validateHCAtomMediaURL(media.URL) != nil {
				return nil, errors.New("hc atom media content is invalid")
			}
		default:
			return nil, errors.New("hc atom content type is unsupported")
		}
	}
	return content, nil
}

func ValidateHCAtomV3Request(input VideoCreateRequest) error {
	_, err := normalizeHCAtomV3Content(input)
	return err
}

func validateHCAtomMediaURL(raw string) error {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "data:") {
		return errors.New("data URLs are forbidden")
	}
	if strings.HasPrefix(raw, "asset://") {
		identifier := strings.TrimPrefix(raw, "asset://")
		if strings.HasPrefix(identifier, "asset-") && !strings.ContainsAny(identifier, "?#/ ") {
			return nil
		}
		return errors.New("asset URL is invalid")
	}
	return validatePublicHTTPSAssetURL(raw)
}

func (a *HCAtomV3Adapter) taskURL() (string, error) {
	if _, err := validateHCAtomV3BaseURL(a.baseURL); err != nil {
		return "", err
	}
	return HCAtomSeedanceV3BaseURL + HCAtomSeedanceV3Path, nil
}

func (a *HCAtomV3Adapter) Create(ctx context.Context, input VideoCreateRequest) (*VideoProviderTask, error) {
	endpoint, err := a.taskURL()
	if err != nil {
		return nil, err
	}
	content, err := normalizeHCAtomV3Content(input)
	if err != nil {
		return nil, err
	}
	payload := struct {
		Model           string             `json:"model"`
		Content         []VideoContentItem `json:"content"`
		Ratio           string             `json:"ratio,omitempty"`
		GenerateAudio   bool               `json:"generate_audio"`
		ReturnLastFrame bool               `json:"return_last_frame"`
		Watermark       bool               `json:"watermark"`
		Duration        int                `json:"duration,omitempty"`
		Resolution      string             `json:"resolution,omitempty"`
	}{HCAtomSeedanceV3Model, content, input.Ratio, input.GenerateAudio, input.ReturnLastFrame, input.Watermark, input.Duration, input.Resolution}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return a.doTask(ctx, http.MethodPost, endpoint, body)
}

func (a *HCAtomV3Adapter) Poll(ctx context.Context, id string) (*VideoProviderTask, error) {
	endpoint, err := a.taskURL()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("hc atom upstream task id is required")
	}
	return a.doTask(ctx, http.MethodGet, endpoint+"/"+url.PathEscape(id), nil)
}
func (a *HCAtomV3Adapter) Cancel(ctx context.Context, id string) (*VideoProviderTask, error) {
	endpoint, err := a.taskURL()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("hc atom upstream task id is required")
	}
	return a.doTask(ctx, http.MethodDelete, endpoint+"/"+url.PathEscape(id), nil)
}
func (a *HCAtomV3Adapter) doTask(ctx context.Context, method, endpoint string, body []byte) (*VideoProviderTask, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, &VideoProviderTransportError{fmt.Errorf("hc atom transport failed: %s", RedactVideoSecrets(err.Error(), a.apiKey))}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errors.New("hc atom response read failed")
	}
	if a.apiKey != "" && strings.Contains(string(data), a.apiKey) {
		return nil, errors.New("hc atom response echoed provider credential")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hc atom request failed: http_%d", resp.StatusCode)
	}
	var parsed struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Content struct {
			VideoURL     string `json:"video_url"`
			LastFrameURL string `json:"last_frame_url"`
		} `json:"content"`
		Usage struct {
			CompletionTokens *int64 `json:"completion_tokens"`
			TotalTokens      *int64 `json:"total_tokens"`
		} `json:"usage"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.New("hc atom response is invalid")
	}
	if strings.TrimSpace(parsed.ID) == "" {
		return nil, errors.New("hc atom response missing task id")
	}
	status, err := normalizeSeedanceStatus(parsed.Status)
	if err != nil {
		return nil, errors.New("hc atom returned unknown task status")
	}
	if parsed.Content.VideoURL != "" && validatePublicHTTPSAssetURL(parsed.Content.VideoURL) != nil {
		return nil, errors.New("hc atom returned unsafe result URL")
	}
	if parsed.Content.LastFrameURL != "" && validatePublicHTTPSAssetURL(parsed.Content.LastFrameURL) != nil {
		return nil, errors.New("hc atom returned unsafe last-frame URL")
	}
	usage := parsed.Usage.CompletionTokens
	if usage == nil || *usage <= 0 {
		usage = parsed.Usage.TotalTokens
	}
	if usage != nil && *usage <= 0 {
		usage = nil
	}
	return &VideoProviderTask{UpstreamTaskID: parsed.ID, Status: status, ResultURL: parsed.Content.VideoURL, LastFrameURL: parsed.Content.LastFrameURL, CompletionTokens: usage, ErrorCode: sanitizeProviderCode(parsed.Error.Code), ErrorMessage: sanitizeProviderMessage(parsed.Error.Message)}, nil
}

type VideoProviderTask struct {
	UpstreamTaskID          string  `json:"id"`
	Status                  string  `json:"status"`
	ResultURL               string  `json:"result_url"`
	LastFrameURL            string  `json:"last_frame_url"`
	CompletionTokens        *int64  `json:"completion_tokens"`
	ErrorCode               string  `json:"error_code"`
	ErrorMessage            string  `json:"error_message"`
	UpstreamModel           *string `json:"-"`
	UpstreamDurationSeconds *int    `json:"-"`
	UpstreamResolution      *string `json:"-"`
}

type SeedanceAdapter struct {
	client          *http.Client
	baseURL, apiKey string
}

func NewSeedanceAdapter(client *http.Client, baseURL, apiKey string) *SeedanceAdapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &SeedanceAdapter{client: client, baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), apiKey: apiKey}
}

func validateArkBaseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("seedance endpoint must be HTTPS")
	}
	host := strings.ToLower(u.Hostname())
	if host != "ark.cn-beijing.volces.com" {
		return nil, errors.New("seedance endpoint host is not allowlisted")
	}
	return u, nil
}

func arkTaskCollectionURL(u *url.URL) string {
	base := strings.TrimRight(u.String(), "/")
	if strings.HasSuffix(u.Path, "/api/v3") {
		return base + "/contents/generations/tasks"
	}
	return base + "/api/v3/contents/generations/tasks"
}

func (a *SeedanceAdapter) Create(ctx context.Context, input VideoCreateRequest) (*VideoProviderTask, error) {
	if err := ValidateTinyRealContract(input); err != nil {
		return nil, err
	}
	u, err := validateArkBaseURL(a.baseURL)
	if err != nil {
		return nil, err
	}
	type contentItem struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	payload := struct {
		Model           string        `json:"model"`
		Content         []contentItem `json:"content"`
		Duration        int           `json:"duration"`
		Resolution      string        `json:"resolution"`
		ReturnLastFrame bool          `json:"return_last_frame"`
	}{SeedanceModel, []contentItem{{Type: "text", Text: input.Prompt}}, input.Duration, input.Resolution, input.ReturnLastFrame}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, arkTaskCollectionURL(u), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("seedance create transport failed: %s", RedactVideoSecrets(err.Error(), a.apiKey))
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return nil, errors.New("seedance create response read failed")
	}
	if a.apiKey != "" && strings.Contains(string(data), a.apiKey) {
		return nil, errors.New("seedance response echoed provider credential")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("seedance create failed: http_%d", resp.StatusCode)
	}
	var task VideoProviderTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, errors.New("seedance create response is invalid")
	}
	if task.UpstreamTaskID == "" {
		return nil, errors.New("seedance create response missing task id")
	}
	return &task, nil
}

func (a *SeedanceAdapter) Poll(ctx context.Context, upstreamTaskID string) (*VideoProviderTask, error) {
	u, err := validateArkBaseURL(a.baseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(upstreamTaskID) == "" {
		return nil, errors.New("seedance upstream task id is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, arkTaskCollectionURL(u)+"/"+url.PathEscape(upstreamTaskID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("seedance poll transport failed: %s", RedactVideoSecrets(err.Error(), a.apiKey))
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errors.New("seedance poll response read failed")
	}
	if a.apiKey != "" && strings.Contains(string(data), a.apiKey) {
		return nil, errors.New("seedance response echoed provider credential")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("seedance poll failed: http_%d", resp.StatusCode)
	}
	var parsed struct {
		ID           string          `json:"id"`
		Status       string          `json:"status"`
		Model        string          `json:"model"`
		Duration     json.RawMessage `json:"duration"`
		Resolution   string          `json:"resolution"`
		LastFrameURL string          `json:"last_frame_url"`
		Content      struct {
			VideoURL     string `json:"video_url"`
			LastFrameURL string `json:"last_frame_url"`
		} `json:"content"`
		Usage struct {
			CompletionTokens *int64 `json:"completion_tokens"`
			TotalTokens      *int64 `json:"total_tokens"`
		} `json:"usage"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.New("seedance poll response is invalid")
	}
	upstreamDuration, err := parseOptionalProviderDuration(parsed.Duration)
	if err != nil {
		return nil, errors.New("seedance poll response has invalid duration")
	}
	status, err := normalizeSeedanceStatus(parsed.Status)
	if err != nil {
		return nil, err
	}
	lastFrame := parsed.LastFrameURL
	if lastFrame == "" {
		lastFrame = parsed.Content.LastFrameURL
	}
	if parsed.Content.VideoURL != "" {
		if err := validatePublicHTTPSAssetURL(parsed.Content.VideoURL); err != nil {
			return nil, errors.New("seedance returned unsafe result URL")
		}
	}
	if lastFrame != "" {
		if err := validatePublicHTTPSAssetURL(lastFrame); err != nil {
			return nil, errors.New("seedance returned unsafe last-frame URL")
		}
	}
	return &VideoProviderTask{UpstreamTaskID: parsed.ID, Status: status, ResultURL: parsed.Content.VideoURL, LastFrameURL: lastFrame,
		CompletionTokens: parsed.Usage.CompletionTokens, ErrorCode: sanitizeProviderCode(parsed.Error.Code), ErrorMessage: sanitizeProviderMessage(parsed.Error.Message),
		UpstreamModel: optionalTrimmedString(parsed.Model), UpstreamDurationSeconds: upstreamDuration,
		UpstreamResolution: optionalTrimmedString(parsed.Resolution)}, nil
}

func optionalTrimmedString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseOptionalProviderDuration(raw json.RawMessage) (*int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		var text string
		if stringErr := json.Unmarshal(raw, &text); stringErr != nil {
			return nil, err
		}
		parsed, parseErr := strconv.Atoi(strings.TrimSpace(text))
		if parseErr != nil {
			return nil, parseErr
		}
		value = parsed
	}
	if value <= 0 {
		return nil, errors.New("duration must be positive")
	}
	return &value, nil
}

func validatePublicHTTPSAssetURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" {
		return errors.New("asset URL must be HTTPS")
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return errors.New("asset URL host is private")
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast()) {
		return errors.New("asset URL host is private")
	}
	return nil
}

func sanitizeProviderCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) > 64 {
		value = value[:64]
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return "provider_error"
		}
	}
	if value == "" {
		return "provider_error"
	}
	return value
}

func sanitizeProviderMessage(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "upstream provider reported an error"
}

func normalizeSeedanceStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "queued", "pending", "created", "submitted", "in_queue":
		return VideoStatusSubmitted, nil
	case "running", "processing":
		return VideoStatusRunning, nil
	case "succeeded", "success", "completed":
		return VideoStatusSucceeded, nil
	case "failed", "error", "timeout", "expired":
		return VideoStatusFailed, nil
	case "cancelled", "canceled":
		return VideoStatusCancelled, nil
	default:
		return "", errors.New("seedance returned unknown task status")
	}
}

func RedactVideoSecrets(value string, secrets ...string) string {
	out := value
	for _, secret := range secrets {
		if secret != "" {
			out = strings.ReplaceAll(out, secret, "[REDACTED]")
		}
	}
	return out
}
