package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
)

func (s *GatewayService) hcAtomFixedClient() *HCAtomFixedClient {
	client := s.hcAtomClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &HCAtomFixedClient{client: client}
}

func (s *GatewayService) resolveHCAtomAccountKey(account *Account) (string, error) {
	if s == nil || s.cfg == nil || !s.cfg.HCAtom.LLMEnabled {
		return "", errors.New("HC-ATOM LLM dispatch is disabled")
	}
	cipher, err := NewHCAtomCredentialCipher(s.cfg.BatchImage.HCAtomEncryptionKey)
	if err != nil {
		return "", ErrHCAtomCredentialKeyUnavailable
	}
	return ResolveHCAtomAPIKey(account, cipher)
}

func (s *GatewayService) forwardHCAtomMessages(ctx context.Context, c *gin.Context, account *Account, body []byte, model string, stream bool) (*ForwardResult, error) {
	return s.forwardHCAtomPassthrough(ctx, c, account, body, model, stream, HCAtomCapabilityMessages)
}

func (s *GatewayService) forwardHCAtomPassthrough(ctx context.Context, c *gin.Context, account *Account, body []byte, model string, stream bool, capability string) (*ForwardResult, error) {
	started := time.Now()
	if _, ok := LookupHCAtomModel(capability, model); !ok {
		return nil, errors.New("HC-ATOM model is not enabled for this endpoint")
	}
	key, err := s.resolveHCAtomAccountKey(account)
	if err != nil {
		return nil, err
	}
	response, err := s.hcAtomFixedClient().Do(ctx, capability, model, key, body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		status := response.StatusCode
		data, readErr := readHCAtomResponse(response, key)
		if readErr != nil {
			data = nil
		}
		if s.shouldFailoverUpstreamError(status) {
			return nil, &UpstreamFailoverError{StatusCode: status, ResponseBody: data}
		}
		return nil, fmt.Errorf("HC-ATOM request failed: http_%d", status)
	}
	contentType := response.Header.Get("Content-Type")
	requestID := response.Header.Get("x-request-id")
	data, err := readHCAtomResponse(response, key)
	if err != nil {
		return nil, err
	}
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") || stream {
		contentType = "text/event-stream"
	}
	if contentType == "" {
		contentType = "application/json; charset=utf-8"
	}
	c.Data(http.StatusOK, contentType, data)
	usage := hcAtomUsageFromJSON(data)
	if strings.Contains(contentType, "text/event-stream") {
		usage = hcAtomUsageFromSSE(data)
	}
	spec, _ := LookupHCAtomModel(capability, model)
	return &ForwardResult{RequestID: requestID, Usage: usage, Model: model, UpstreamModel: spec.UpstreamModel, Stream: stream, Duration: time.Since(started)}, nil
}

func (s *GatewayService) forwardHCAtomResponses(ctx context.Context, c *gin.Context, account *Account, body []byte) (*ForwardResult, error) {
	started := time.Now()
	var request apicompat.ResponsesRequest
	if err := json.Unmarshal(body, &request); err != nil || strings.TrimSpace(request.Model) == "" {
		return nil, errors.New("parse HC-ATOM responses request")
	}
	key, err := s.resolveHCAtomAccountKey(account)
	if err != nil {
		return nil, err
	}
	var capability string
	var upstreamBody []byte
	if _, ok := LookupHCAtomModel(HCAtomCapabilityChat, request.Model); ok {
		capability = HCAtomCapabilityChat
		chatRequest, convertErr := apicompat.ResponsesToChatCompletionsRequest(&request)
		if convertErr != nil {
			return nil, fmt.Errorf("convert Responses request to chat completions: %w", convertErr)
		}
		upstreamBody, err = json.Marshal(chatRequest)
	} else if _, ok := LookupHCAtomModel(HCAtomCapabilityMessages, request.Model); ok {
		capability = HCAtomCapabilityMessages
		anthropicRequest, convertErr := apicompat.ResponsesToAnthropicRequest(&request)
		if convertErr != nil {
			return nil, fmt.Errorf("convert Responses request to messages: %w", convertErr)
		}
		upstreamBody, err = json.Marshal(anthropicRequest)
	} else {
		return nil, errors.New("HC-ATOM model is not enabled for Responses")
	}
	if err != nil {
		return nil, errors.New("marshal HC-ATOM compatibility request")
	}
	response, err := s.hcAtomFixedClient().Do(ctx, capability, request.Model, key, upstreamBody)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		status := response.StatusCode
		data, _ := readHCAtomResponse(response, key)
		if s.shouldFailoverUpstreamError(status) {
			return nil, &UpstreamFailoverError{StatusCode: status, ResponseBody: data}
		}
		return nil, fmt.Errorf("HC-ATOM request failed: http_%d", status)
	}
	requestID := response.Header.Get("x-request-id")
	data, err := readHCAtomResponse(response, key)
	if err != nil {
		return nil, err
	}
	var output []byte
	if request.Stream {
		if capability == HCAtomCapabilityChat {
			output, err = hcAtomChatSSEToResponses(data, request.Model)
		} else {
			output, err = hcAtomMessagesSSEToResponses(data, request.Model)
		}
		if err == nil {
			c.Data(http.StatusOK, "text/event-stream", output)
		}
	} else {
		if capability == HCAtomCapabilityChat {
			var chatResponse apicompat.ChatCompletionsResponse
			if err = json.Unmarshal(data, &chatResponse); err == nil {
				output, err = json.Marshal(apicompat.ChatCompletionsResponseToResponses(&chatResponse, request.Model, apicompat.CustomToolNames(request.Tools), apicompat.HasToolSearchTool(request.Tools), apicompat.NamespaceToolNames(request.Tools)))
			}
		} else {
			var messagesResponse apicompat.AnthropicResponse
			if err = json.Unmarshal(data, &messagesResponse); err == nil {
				converted := apicompat.AnthropicToResponsesResponse(&messagesResponse)
				converted.Model = request.Model
				output, err = json.Marshal(converted)
			}
		}
		if err == nil {
			c.Data(http.StatusOK, "application/json; charset=utf-8", output)
		}
	}
	if err != nil {
		return nil, errors.New("HC-ATOM compatibility response is invalid")
	}
	usage := hcAtomUsageFromJSON(data)
	if request.Stream {
		usage = hcAtomUsageFromSSE(data)
	}
	spec, _ := LookupHCAtomModel(capability, request.Model)
	return &ForwardResult{RequestID: requestID, Usage: usage, Model: request.Model, UpstreamModel: spec.UpstreamModel, Stream: request.Stream, Duration: time.Since(started)}, nil
}

func hcAtomChatSSEToResponses(data []byte, model string) ([]byte, error) {
	state := apicompat.NewChatCompletionsToResponsesStreamState(model)
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	writeEvents := func(events []apicompat.ResponsesStreamEvent) error {
		for _, event := range events {
			line, err := apicompat.ResponsesEventToSSE(event)
			if err != nil {
				return err
			}
			out.WriteString(line)
		}
		return nil
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var chunk apicompat.ChatCompletionsChunk
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			return nil, errors.New("invalid HC-ATOM chat stream")
		}
		if err := writeEvents(apicompat.ChatCompletionsChunkToResponsesEvents(&chunk, state)); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := writeEvents(apicompat.FinalizeChatCompletionsResponsesStream(state)); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func hcAtomMessagesSSEToResponses(data []byte, model string) ([]byte, error) {
	state := apicompat.NewAnthropicEventToResponsesState()
	state.Model = model
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	writeEvents := func(events []apicompat.ResponsesStreamEvent) error {
		for _, event := range events {
			line, err := apicompat.ResponsesEventToSSE(event)
			if err != nil {
				return err
			}
			out.WriteString(line)
		}
		return nil
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event apicompat.AnthropicStreamEvent
		if payload == "" || json.Unmarshal([]byte(payload), &event) != nil {
			continue
		}
		if err := writeEvents(apicompat.AnthropicEventToResponsesEvents(&event, state)); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := writeEvents(apicompat.FinalizeAnthropicResponsesStream(state)); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
