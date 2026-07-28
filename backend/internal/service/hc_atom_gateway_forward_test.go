package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newHCAtomForwardTestService(t *testing.T, transport hcAtomGatewayRoundTripFunc) (*GatewayService, *Account) {
	t.Helper()
	keyHex := strings.Repeat("a", 64)
	cipher, err := NewHCAtomCredentialCipher(keyHex)
	require.NoError(t, err)
	encrypted, err := cipher.Encrypt("provider-secret")
	require.NoError(t, err)
	return &GatewayService{
		cfg:          &config.Config{BatchImage: config.BatchImageConfig{HCAtomEncryptionKey: keyHex}, HCAtom: config.HCAtomConfig{LLMEnabled: true, SyncImageEnabled: true}},
		hcAtomClient: &http.Client{Transport: transport},
	}, &Account{ID: 9, Platform: PlatformHCAtom, Type: AccountTypeAPIKey, Credentials: map[string]any{HCAtomAPIKeyCiphertextField: encrypted}}
}

func newHCAtomGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	return context, recorder
}

func TestHCAtomSelectedAccountForwardingUsesFixedLLMEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		capability string
		model      string
		endpoint   string
		response   string
	}{
		{"chat", HCAtomCapabilityChat, "gpt-5.6-sol", "https://ai-aigc.fzyinghe.com/v1/chat/completions", `{"id":"chat-1","choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`},
		{"messages", HCAtomCapabilityMessages, "claude-opus-4-6", "https://ai-aigc.fzyinghe.com/v1/messages", `{"id":"msg-1","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":2,"output_tokens":1}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, account := newHCAtomForwardTestService(t, hcAtomGatewayRoundTripFunc(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, test.endpoint, request.URL.String())
				require.Equal(t, "Bearer provider-secret", request.Header.Get("Authorization"))
				headers := make(http.Header)
				headers.Set("Content-Type", "application/json")
				headers.Set("x-request-id", "req-1")
				return &http.Response{StatusCode: http.StatusOK, Header: headers, Body: io.NopCloser(strings.NewReader(test.response))}, nil
			}))
			context, recorder := newHCAtomGinContext()
			result, err := service.forwardHCAtomPassthrough(context.Request.Context(), context, account, []byte(`{"model":"`+test.model+`","messages":[{"role":"user","content":"hi"}]}`), test.model, false, test.capability)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, "req-1", result.RequestID)
			require.Equal(t, ClaudeUsage{InputTokens: 2, OutputTokens: 1}, result.Usage)
		})
	}
}

func TestHCAtomResponsesCompatibilityForChatAndClaude(t *testing.T) {
	tests := []struct {
		model    string
		endpoint string
		response string
	}{
		{"gpt-5.6-sol", "https://ai-aigc.fzyinghe.com/v1/chat/completions", `{"id":"chat-1","object":"chat.completion","model":"gpt-5.6-sol","choices":[{"index":0,"message":{"role":"assistant","content":"chat ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`},
		{"claude-opus-4-6", "https://ai-aigc.fzyinghe.com/v1/messages", `{"id":"msg-1","type":"message","role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"claude ok"}],"stop_reason":"end_turn","usage":{"input_tokens":4,"output_tokens":2}}`},
	}
	for _, test := range tests {
		t.Run(test.model, func(t *testing.T) {
			service, account := newHCAtomForwardTestService(t, hcAtomGatewayRoundTripFunc(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, test.endpoint, request.URL.String())
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(test.response))}, nil
			}))
			context, recorder := newHCAtomGinContext()
			result, err := service.forwardHCAtomResponses(context.Request.Context(), context, account, []byte(`{"model":"`+test.model+`","input":"hello"}`))
			require.NoError(t, err)
			require.False(t, result.Stream)
			require.Contains(t, recorder.Body.String(), `"object":"response"`)
			require.Contains(t, recorder.Body.String(), `"output_text"`)
		})
	}
}

func TestHCAtomResponsesChatSSECompatibility(t *testing.T) {
	sse := "data: {\"id\":\"chat-stream\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt-5.6-sol\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chat-stream\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt-5.6-sol\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":1}}\n\n" +
		"data: [DONE]\n\n"
	service, account := newHCAtomForwardTestService(t, hcAtomGatewayRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(sse))}, nil
	}))
	context, recorder := newHCAtomGinContext()
	result, err := service.forwardHCAtomResponses(context.Request.Context(), context, account, []byte(`{"model":"gpt-5.6-sol","input":"hello","stream":true}`))
	require.NoError(t, err)
	require.True(t, result.Stream)
	require.Contains(t, recorder.Body.String(), "response.output_text.delta")
	require.Contains(t, recorder.Body.String(), "response.completed")
}

func TestHCAtomSyncImageUsesSelectedAccountAndFixedEndpoint(t *testing.T) {
	service, account := newHCAtomForwardTestService(t, hcAtomGatewayRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "https://ai-aigc.fzyinghe.com/v1/images/generations", request.URL.String())
		require.Equal(t, "Bearer provider-secret", request.Header.Get("Authorization"))
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"created":1,"data":[{"url":"https://assets.example/result.png"}]}`))}, nil
	}))
	context, recorder := newHCAtomGinContext()
	result, err := service.ForwardHCAtomSyncImage(context.Request.Context(), context, account, []byte(`{"model":"seedream-5.0","prompt":"red fox","n":1,"size":"1024x1024"}`))
	require.NoError(t, err)
	require.Equal(t, 1, result.ImageCount)
	require.Contains(t, recorder.Body.String(), "result.png")
}

func TestHCAtomForwardingFailsClosedWhenFeatureDisabled(t *testing.T) {
	service, account := newHCAtomForwardTestService(t, hcAtomGatewayRoundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("transport must not be called")
		return nil, nil
	}))
	service.cfg.HCAtom.LLMEnabled = false
	context, _ := newHCAtomGinContext()
	_, err := service.forwardHCAtomPassthrough(context.Request.Context(), context, account, []byte(`{"model":"gpt-5.6-sol"}`), "gpt-5.6-sol", false, HCAtomCapabilityChat)
	require.ErrorContains(t, err, "disabled")
}

var _ = context.Background
