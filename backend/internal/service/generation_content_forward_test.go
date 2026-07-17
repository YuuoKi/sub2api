package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGenerationContentCaptureForwardPreservesNonStreamingJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	body := []byte(`{"model":"claude-3-5-sonnet-latest","messages":[{"role":"user","content":"hello"}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-3-5-sonnet-latest"}
	upstreamJSON := `{"id":"msg_capture_json","type":"message","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":2,"output_tokens":1}}`
	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-capture-json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}}
	cfg := generationCaptureConfig(256, 512)
	svc := &GatewayService{cfg: cfg, responseHeaderFilter: compileResponseHeaderFilter(cfg), httpUpstream: upstream, rateLimitService: &RateLimitService{}, deferredService: &DeferredService{}}

	result, err := svc.Forward(context.Background(), c, newAnthropicAPIKeyAccountForTest(), parsed)
	require.NoError(t, err)
	require.Equal(t, upstreamJSON, recorder.Body.String())
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	require.Equal(t, []byte(upstreamJSON), result.ResponseSample)
	require.Equal(t, len(upstreamJSON), result.ResponseBytes)
	require.False(t, result.ResponseTruncated)
}

func TestGenerationContentCaptureForwardPreservesStreamingSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	body := []byte(`{"model":"claude-3-5-sonnet-latest","stream":true,"messages":[{"role":"user","content":"hello"}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-3-5-sonnet-latest", Stream: true}
	upstreamSSE := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":2}}}`,
		"",
		`data: {"type":"message_delta","usage":{"output_tokens":1}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-capture-sse"}},
		Body:       io.NopCloser(strings.NewReader(upstreamSSE)),
	}}
	cfg := generationCaptureConfig(256, 2048)
	cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc := &GatewayService{cfg: cfg, responseHeaderFilter: compileResponseHeaderFilter(cfg), httpUpstream: upstream, rateLimitService: &RateLimitService{}, deferredService: &DeferredService{}}

	result, err := svc.Forward(context.Background(), c, newAnthropicAPIKeyAccountForTest(), parsed)
	require.NoError(t, err)
	require.Equal(t, upstreamSSE, recorder.Body.String())
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
	require.True(t, recorder.Flushed)
	require.Equal(t, []byte(upstreamSSE), result.ResponseSample)
	require.Equal(t, len(upstreamSSE), result.ResponseBytes)
	require.False(t, result.ResponseTruncated)
}
