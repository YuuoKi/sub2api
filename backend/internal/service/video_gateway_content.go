package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	videoMediaMaxImageBytes = 30 * 1024 * 1024
	videoMediaMaxVideoBytes = 50 * 1024 * 1024
	videoMediaMaxAudioBytes = 15 * 1024 * 1024
	videoMediaHEADTimeout   = 5 * time.Second
)

// videoMediaHTTPDoer is the injectable HEAD client used by media physical checks.
// Tests replace videoMediaHTTPClient so unit suites never hit the network.
type videoMediaHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var videoMediaHTTPClient videoMediaHTTPDoer = &http.Client{Timeout: videoMediaHEADTimeout}

const (
	VideoContentTypeText     = "text"
	VideoContentTypeImageURL = "image_url"
	VideoContentTypeVideoURL = "video_url"
	VideoContentTypeAudioURL = "audio_url"

	VideoContentRoleFirstFrame     = "first_frame"
	VideoContentRoleLastFrame      = "last_frame"
	VideoContentRoleReferenceImage = "reference_image"
	VideoContentRoleReferenceVideo = "reference_video"
	VideoContentRoleReferenceAudio = "reference_audio"
)

func normalizeVideoTaskContent(p VideoTaskCreateParams) ([]VideoTaskContentItem, bool, error) {
	items := make([]VideoTaskContentItem, 0, len(p.Content)+3)
	prompt := strings.TrimSpace(p.Prompt)
	enforceModeMatch := len(p.Content) > 0

	hasText := false
	for _, raw := range p.Content {
		item, err := normalizeVideoContentItem(raw, p.TaskType)
		if err != nil {
			return nil, false, err
		}
		if item.Type == VideoContentTypeText {
			if hasText {
				return nil, false, badVideoContent("content may contain at most one text item")
			}
			hasText = true
			if strings.TrimSpace(item.Text) == "" {
				item.Text = prompt
			}
		}
		items = append(items, item)
	}

	if !hasText {
		items = append([]VideoTaskContentItem{{Type: VideoContentTypeText, Text: prompt}}, items...)
	}

	if legacyImageURL := strings.TrimSpace(p.ReferenceImageURL); legacyImageURL != "" {
		if isSafeDemoPlaceholderURL(legacyImageURL) && p.SafeDemoOnly {
			// Drama safe-demo tasks use internal asset placeholders, not provider-callable URLs.
		} else if isHTTPContentURL(legacyImageURL) {
			role := VideoContentRoleReferenceImage
			if strings.TrimSpace(p.TaskType) == VideoTaskTypeImageToVideo {
				role = VideoContentRoleFirstFrame
			}
			item, err := normalizeVideoContentItem(VideoTaskContentItem{Type: VideoContentTypeImageURL, Role: role, URL: legacyImageURL}, p.TaskType)
			if err != nil {
				return nil, false, err
			}
			items = append(items, item)
			enforceModeMatch = true
		} else {
			return nil, false, badVideoContent("reference_image_url must be an http(s) URL")
		}
	}
	if legacyVideoURL := strings.TrimSpace(p.ReferenceVideoURL); legacyVideoURL != "" {
		if isSafeDemoPlaceholderURL(legacyVideoURL) && p.SafeDemoOnly {
			// Drama safe-demo tasks use internal asset placeholders, not provider-callable URLs.
		} else if isHTTPContentURL(legacyVideoURL) {
			item, err := normalizeVideoContentItem(VideoTaskContentItem{Type: VideoContentTypeVideoURL, Role: VideoContentRoleReferenceVideo, URL: legacyVideoURL}, p.TaskType)
			if err != nil {
				return nil, false, err
			}
			items = append(items, item)
			enforceModeMatch = true
		} else {
			return nil, false, badVideoContent("reference_video_url must be an http(s) URL")
		}
	}

	if err := validateVideoContentContract(p.TaskType, items, enforceModeMatch); err != nil {
		return nil, false, err
	}

	hasVideoInput := false
	for _, item := range items {
		if item.Type == VideoContentTypeVideoURL {
			hasVideoInput = true
			break
		}
	}
	return items, hasVideoInput, nil
}

func normalizeVideoContentItem(raw VideoTaskContentItem, taskType string) (VideoTaskContentItem, error) {
	item := VideoTaskContentItem{
		Type:     strings.ToLower(strings.TrimSpace(raw.Type)),
		Role:     strings.ToLower(strings.TrimSpace(raw.Role)),
		URL:      strings.TrimSpace(raw.URL),
		Text:     strings.TrimSpace(raw.Text),
		VideoID:  strings.TrimSpace(raw.VideoID),
		AudioID:  strings.TrimSpace(raw.AudioID),
		Metadata: raw.Metadata,
	}
	if item.Type == "" {
		return item, badVideoContent("content item type is required")
	}

	switch item.Type {
	case VideoContentTypeText:
		item.Role = ""
		item.URL = ""
		item.VideoID = ""
		item.AudioID = ""
	case VideoContentTypeImageURL:
		if item.Role == "" {
			item.Role = VideoContentRoleReferenceImage
			if strings.TrimSpace(taskType) == VideoTaskTypeImageToVideo {
				item.Role = VideoContentRoleFirstFrame
			}
		}
		if item.Role != VideoContentRoleFirstFrame && item.Role != VideoContentRoleLastFrame && item.Role != VideoContentRoleReferenceImage {
			return item, badVideoContent("image_url role must be first_frame, last_frame, or reference_image")
		}
		if err := validateContentURL(item.URL, "image_url"); err != nil {
			return item, err
		}
	case VideoContentTypeVideoURL:
		if item.Role == "" {
			item.Role = VideoContentRoleReferenceVideo
		}
		if item.Role != VideoContentRoleReferenceVideo {
			return item, badVideoContent("video_url role must be reference_video")
		}
		// Allow video_id-only items (Kling extend) without an HTTP URL.
		if item.URL == "" && item.VideoID == "" {
			return item, badVideoContent("video_url url or video_id is required")
		}
		if item.URL != "" {
			if err := validateContentURL(item.URL, "video_url"); err != nil {
				return item, err
			}
		}
	case VideoContentTypeAudioURL:
		if item.Role == "" {
			item.Role = VideoContentRoleReferenceAudio
		}
		if item.Role != VideoContentRoleReferenceAudio {
			return item, badVideoContent("audio_url role must be reference_audio")
		}
		// Allow audio_id-only items (Kling avatar) without an HTTP URL.
		if item.URL == "" && item.AudioID == "" {
			return item, badVideoContent("audio_url url or audio_id is required")
		}
		if item.URL != "" {
			if err := validateContentURL(item.URL, "audio_url"); err != nil {
				return item, err
			}
		}
	default:
		return item, badVideoContent("content item type must be text, image_url, video_url, or audio_url")
	}
	return item, nil
}

func validateContentURL(rawURL, label string) error {
	if strings.TrimSpace(rawURL) == "" {
		return badVideoContent(fmt.Sprintf("%s url is required", label))
	}
	if err := validateExternalVideoURL(rawURL); err != nil {
		return infraerrors.BadRequest("VIDEO_UNSAFE_REFERENCE_URL", label+" failed SSRF/allowlist validation: "+err.Error())
	}
	// Physical size/MIME probe only after SSRF/allowlist validation, and only when an
	// allowlist is configured (production posture). Without an allowlist we refuse to
	// open outbound HEAD sockets from the gateway (fail-closed for probe amplification).
	if configuredMediaURLAllowlist() == "" {
		return nil
	}
	if err := probeVideoMediaConstraints(rawURL, label); err != nil {
		return err
	}
	return nil
}

func probeVideoMediaConstraints(rawURL, label string) error {
	maxBytes, wantKind := videoMediaLimitsForLabel(label)
	ctx, cancel := context.WithTimeout(context.Background(), videoMediaHEADTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return badVideoContent(fmt.Sprintf("%s media HEAD probe failed: %v", label, err))
	}
	resp, err := videoMediaHTTPClient.Do(req)
	if err != nil {
		return badVideoContent(fmt.Sprintf("%s media HEAD probe failed (fail-closed): %v", label, err))
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return badVideoContent(fmt.Sprintf("%s media HEAD probe returned HTTP %d (fail-closed)", label, resp.StatusCode))
	}

	if cl := strings.TrimSpace(resp.Header.Get("Content-Length")); cl != "" {
		n, err := strconv.ParseInt(cl, 10, 64)
		if err != nil || n < 0 {
			return badVideoContent(fmt.Sprintf("%s media Content-Length is invalid (fail-closed)", label))
		}
		if n > maxBytes {
			return badVideoContent(fmt.Sprintf("%s media exceeds %s size limit (%d bytes > %d)", label, wantKind, n, maxBytes))
		}
	}

	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		if err := validateVideoMediaContentType(ct, wantKind, label); err != nil {
			return err
		}
	}
	return nil
}

func videoMediaLimitsForLabel(label string) (maxBytes int64, kind string) {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "video_url", "video":
		return videoMediaMaxVideoBytes, "video"
	case "audio_url", "audio":
		return videoMediaMaxAudioBytes, "audio"
	default:
		return videoMediaMaxImageBytes, "image"
	}
}

func validateVideoMediaContentType(contentType, kind, label string) error {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if mediaType == "" || mediaType == "application/octet-stream" {
		return nil
	}
	switch kind {
	case "image":
		if strings.HasPrefix(mediaType, "image/") {
			return nil
		}
	case "video":
		if strings.HasPrefix(mediaType, "video/") {
			return nil
		}
	case "audio":
		if strings.HasPrefix(mediaType, "audio/") {
			return nil
		}
	}
	return badVideoContent(fmt.Sprintf("%s media Content-Type %q is not a valid %s type (fail-closed)", label, mediaType, kind))
}

func isHTTPContentURL(rawURL string) bool {
	value := strings.ToLower(strings.TrimSpace(rawURL))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func isSafeDemoPlaceholderURL(rawURL string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(rawURL)), "safe-demo://")
}

func validateVideoContentContract(taskType string, items []VideoTaskContentItem, enforceModeMatch bool) error {
	var (
		textCount           int
		firstFrameCount     int
		lastFrameCount      int
		referenceImageCount int
		videoCount          int
		audioCount          int
	)

	for _, item := range items {
		switch item.Type {
		case VideoContentTypeText:
			textCount++
		case VideoContentTypeImageURL:
			switch item.Role {
			case VideoContentRoleFirstFrame:
				firstFrameCount++
			case VideoContentRoleLastFrame:
				lastFrameCount++
			case VideoContentRoleReferenceImage:
				referenceImageCount++
			}
		case VideoContentTypeVideoURL:
			videoCount++
		case VideoContentTypeAudioURL:
			audioCount++
		}
	}

	if textCount > 1 {
		return badVideoContent("content may contain at most one text item")
	}
	if firstFrameCount > 1 || lastFrameCount > 1 || firstFrameCount+lastFrameCount > 2 {
		return badVideoContent("first/last frame image mode allows at most one first_frame and one last_frame")
	}
	if lastFrameCount > 0 && firstFrameCount == 0 {
		return badVideoContent("last_frame requires first_frame")
	}
	if referenceImageCount > 9 {
		return badVideoContent("reference_image content is limited to 9 images")
	}
	if videoCount > 3 {
		return badVideoContent("reference_video content is limited to 3 videos")
	}
	if audioCount > 3 {
		return badVideoContent("reference_audio content is limited to 3 audio clips")
	}

	frameMode := firstFrameCount+lastFrameCount > 0
	referenceMode := referenceImageCount+videoCount+audioCount > 0
	if frameMode && referenceMode {
		return badVideoContent("image modes are mutually exclusive: first/last frame cannot mix with reference media")
	}
	if audioCount > 0 && referenceImageCount+videoCount == 0 {
		return badVideoContent("audio_url content cannot be used without at least one image_url or video_url reference")
	}

	derived := VideoTaskTypeTextToVideo
	if frameMode {
		derived = VideoTaskTypeImageToVideo
	} else if referenceMode {
		derived = VideoTaskTypeReferenceToVideo
	}
	if enforceModeMatch && strings.TrimSpace(taskType) != derived {
		return badVideoContent(fmt.Sprintf("task_type %q does not match content mode %q", strings.TrimSpace(taskType), derived))
	}
	return nil
}

func validateVideoGenerationContract(provider, model string, duration int, resolution string) error {
	normalizedResolution := strings.ToLower(strings.TrimSpace(resolution))
	switch normalizedResolution {
	case "", "480p", "720p", "1080p":
	default:
		return badVideoContent("resolution must be one of 480p, 720p, or 1080p")
	}
	if strings.TrimSpace(provider) == VideoProviderSeedance {
		if strings.Contains(strings.ToLower(model), "fast") && normalizedResolution == "1080p" {
			return badVideoContent("Seedance 2.0 fast does not support 1080p resolution")
		}
		if duration != -1 && (duration < 4 || duration > 15) {
			return badVideoContent("Seedance duration must be -1 or between 4 and 15 seconds")
		}
		return nil
	}
	if strings.TrimSpace(provider) == VideoProviderKling {
		modelLower := strings.ToLower(strings.TrimSpace(model))
		omni := strings.Contains(modelLower, "omni") || strings.Contains(modelLower, "kling-video-o1") || modelLower == "kling-o1"
		if omni {
			// Official omni duration is wider than classic t2v/i2v (5|10).
			// Documented allow-range used here: 3–15 seconds inclusive.
			if duration < 3 || duration > 15 {
				return badVideoContent("kling omni duration must be between 3 and 15 seconds")
			}
			return nil
		}
		if duration != 5 && duration != 10 {
			return badVideoContent("kling duration must be 5 or 10 seconds")
		}
		return nil
	}
	if duration == -1 {
		return nil
	}
	if duration < 1 || duration > 60 {
		return badVideoContent("duration must be -1 or between 1 and 60 seconds")
	}
	return nil
}

func badVideoContent(message string) error {
	return infraerrors.BadRequest("VIDEO_INVALID_CONTENT", message)
}
