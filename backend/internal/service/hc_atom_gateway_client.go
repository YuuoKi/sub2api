package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const hcAtomGatewayMaxResponseBytes = 8 << 20

type hcAtomRoundTripClient interface {
	Do(*http.Request) (*http.Response, error)
}

// HCAtomFixedClient only accepts a catalog entry selected by capability and
// public model. Callers cannot provide a base URL or arbitrary path.
type HCAtomFixedClient struct {
	client hcAtomRoundTripClient
}

func NewHCAtomFixedClient(client *http.Client) *HCAtomFixedClient {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &HCAtomFixedClient{client: client}
}

func (c *HCAtomFixedClient) Do(ctx context.Context, capability, model, apiKey string, body []byte) (*http.Response, error) {
	spec, ok := LookupHCAtomModel(capability, model)
	if !ok {
		return nil, errors.New("HC-ATOM model is not enabled for this capability")
	}
	if !validHCAtomFixedRoute(spec) {
		return nil, errors.New("HC-ATOM model route is incomplete")
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, ErrHCAtomCredentialMissing
	}
	rewritten, err := replaceHCAtomRequestModel(body, spec.UpstreamModel)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, spec.Origin+spec.Path, bytes.NewReader(rewritten))
	if err != nil {
		return nil, errors.New("HC-ATOM request creation failed")
	}
	applyHCAtomAuthHeader(request.Header, spec.AuthScheme, apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return nil, errors.New("HC-ATOM transport failed")
	}
	if response == nil || response.Body == nil {
		return nil, errors.New("HC-ATOM returned an empty response")
	}
	return response, nil
}

func validHCAtomFixedRoute(spec HCAtomModelSpec) bool {
	origin, path := strings.TrimRight(strings.TrimSpace(spec.Origin), "/"), strings.TrimSpace(spec.Path)
	if origin != HCAtomChatOrigin && origin != HCAtomMediaOrigin {
		return false
	}
	return strings.HasPrefix(path, "/") && path != "/"
}

func applyHCAtomAuthHeader(header http.Header, scheme, apiKey string) {
	if strings.TrimSpace(scheme) == HCAtomAuthXAPIKey {
		header.Set("x-api-key", apiKey)
		return
	}
	header.Set("Authorization", "Bearer "+apiKey)
}

func replaceHCAtomRequestModel(body []byte, model string) ([]byte, error) {
	var payload map[string]any
	if len(body) == 0 || json.Unmarshal(body, &payload) != nil {
		return nil, errors.New("HC-ATOM request body is invalid")
	}
	payload["model"] = model
	return json.Marshal(payload)
}

func readHCAtomResponse(response *http.Response, apiKey string) ([]byte, error) {
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, hcAtomGatewayMaxResponseBytes+1))
	if err != nil {
		return nil, errors.New("HC-ATOM response read failed")
	}
	if len(data) > hcAtomGatewayMaxResponseBytes {
		return nil, errors.New("HC-ATOM response exceeds the size limit")
	}
	if apiKey != "" && strings.Contains(string(data), apiKey) {
		return nil, errors.New("HC-ATOM response echoed provider credential")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HC-ATOM request failed: http_%d", response.StatusCode)
	}
	return data, nil
}

func hcAtomUsageFromJSON(data []byte) ClaudeUsage {
	var envelope struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			InputTokens      int `json:"input_tokens"`
			OutputTokens     int `json:"output_tokens"`
		} `json:"usage"`
		Message struct {
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return ClaudeUsage{}
	}
	input, output := envelope.Usage.PromptTokens, envelope.Usage.CompletionTokens
	if input == 0 {
		input = envelope.Usage.InputTokens
	}
	if output == 0 {
		output = envelope.Usage.OutputTokens
	}
	if input == 0 {
		input = envelope.Message.Usage.InputTokens
	}
	if output == 0 {
		output = envelope.Message.Usage.OutputTokens
	}
	return ClaudeUsage{InputTokens: input, OutputTokens: output}
}

func hcAtomUsageFromSSE(data []byte) ClaudeUsage {
	var usage ClaudeUsage
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		current := hcAtomUsageFromJSON([]byte(payload))
		if current.InputTokens > usage.InputTokens {
			usage.InputTokens = current.InputTokens
		}
		if current.OutputTokens > usage.OutputTokens {
			usage.OutputTokens = current.OutputTokens
		}
	}
	return usage
}
