package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *GatewayService) ForwardHCAtomSyncImage(ctx context.Context, c *gin.Context, account *Account, body []byte) (*ForwardResult, error) {
	started := time.Now()
	if s == nil || s.cfg == nil || !s.cfg.HCAtom.SyncImageEnabled {
		return nil, errors.New("HC-ATOM sync image dispatch is disabled")
	}
	var request struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		N      int    `json:"n"`
		Count  int    `json:"count"`
		Size   string `json:"size"`
	}
	if json.Unmarshal(body, &request) != nil {
		return nil, errors.New("HC-ATOM sync image request is invalid")
	}
	spec, ok := LookupHCAtomModel(HCAtomCapabilityImageSync, request.Model)
	if !ok {
		return nil, errors.New("HC-ATOM sync image model is not enabled")
	}
	normalizedBody, normalizedN, normalizedSize, err := normalizeHCAtomSyncImageBody(body, spec)
	if err != nil {
		return nil, err
	}
	body = normalizedBody
	request.N = normalizedN
	request.Size = normalizedSize
	cipher, err := NewHCAtomCredentialCipher(s.cfg.BatchImage.HCAtomEncryptionKey)
	if err != nil {
		return nil, ErrHCAtomCredentialKeyUnavailable
	}
	key, err := ResolveHCAtomAPIKey(account, cipher)
	if err != nil {
		return nil, err
	}
	response, err := s.hcAtomFixedClient().Do(ctx, HCAtomCapabilityImageSync, request.Model, key, body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		status := response.StatusCode
		data, _ := readHCAtomResponse(response, key)
		if s.shouldFailoverUpstreamError(status) {
			return nil, &UpstreamFailoverError{StatusCode: status, ResponseBody: data}
		}
		return nil, errors.New("HC-ATOM sync image upstream rejected the request")
	}
	requestID := response.Header.Get("x-request-id")
	data, err := readHCAtomResponse(response, key)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || len(envelope.Data) == 0 {
		return nil, errors.New("HC-ATOM sync image response is invalid")
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	imageCount := len(envelope.Data)
	if request.N > 0 && imageCount > request.N {
		imageCount = request.N
	}
	size := strings.TrimSpace(request.Size)
	return &ForwardResult{RequestID: requestID, Model: request.Model, UpstreamModel: spec.UpstreamModel, Duration: time.Since(started), ImageCount: imageCount, ImageSize: size, ImageOutputSize: size}, nil
}

func normalizeHCAtomSyncImageBody(body []byte, spec HCAtomModelSpec) ([]byte, int, string, error) {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return nil, 0, "", errors.New("HC-ATOM sync image request is invalid")
	}
	if strings.TrimSpace(stringValue(payload["prompt"])) == "" {
		return nil, 0, "", errors.New("HC-ATOM sync image prompt is required")
	}
	n := intValue(payload["n"])
	if n <= 0 {
		n = intValue(payload["count"])
	}
	if n <= 0 {
		n = 1
	}
	payload["n"] = n
	delete(payload, "count")
	delete(payload, "aspect")

	references, err := hcAtomSyncReferenceImages(payload)
	if err != nil {
		return nil, 0, "", err
	}
	delete(payload, "reference_images")
	if len(references) > 0 {
		if spec.PublicModel != HCAtomImageDoubaoSeedreamModel {
			return nil, 0, "", errors.New("HC-ATOM sync image model does not accept reference images")
		}
		if len(references) == 1 {
			payload["image"] = references[0]
		} else {
			payload["image"] = references
		}
	}

	quality := strings.ToUpper(strings.TrimSpace(stringValue(payload["quality"])))
	size := strings.TrimSpace(stringValue(payload["size"]))
	switch spec.PublicModel {
	case HCAtomImageSeedreamModel:
		switch quality {
		case "1K":
			size = "1024x1024"
		case "4K":
			size = "4096x4096"
		default:
			if size == "" || strings.Contains(size, ":") {
				size = "2048x2048"
			}
		}
	case HCAtomImageDoubaoSeedreamModel:
		if quality == "1K" || quality == "2K" || quality == "4K" {
			size = quality
		}
		if size == "" || strings.Contains(size, ":") {
			size = "1K"
		}
		payload["response_format"] = "url"
		payload["stream"] = false
	}
	delete(payload, "quality")
	payload["size"] = size
	payload["model"] = spec.UpstreamModel
	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, "", errors.New("HC-ATOM sync image request is invalid")
	}
	return normalized, n, size, nil
}

func hcAtomSyncReferenceImages(payload map[string]any) ([]string, error) {
	value, ok := payload["reference_images"]
	if !ok || value == nil {
		value = payload["image"]
	}
	if value == nil {
		return nil, nil
	}
	var raw []any
	switch current := value.(type) {
	case string:
		raw = []any{current}
	case []any:
		raw = current
	default:
		return nil, errors.New("HC-ATOM sync image reference images are invalid")
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		imageURL := strings.TrimSpace(stringValue(item))
		if !isHCAtomHTTPSReferenceURL(imageURL) {
			return nil, errors.New("HC-ATOM sync image reference images must be public HTTPS URLs")
		}
		out = append(out, imageURL)
	}
	return out, nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func intValue(value any) int {
	switch current := value.(type) {
	case float64:
		return int(current)
	case int:
		return current
	default:
		return 0
	}
}
