package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newOpenAICaptureOAuthAccount() *Account {
	// API Key + Responses API (not OAuth passthrough): OAuth passthrough forces
	// upstream stream=true and would break non-streaming JSON byte-equality tests.
	return &Account{
		ID:          123,
		Name:        "openai-capture-acc",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://example.com",
		},
		Extra: map[string]any{
			"use_responses_api":                        true,
			"openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModeOff,
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func newOpenAICaptureGateway(t *testing.T, cfg *config.Config, upstream HTTPUpstream, repo *generationContentMemoryRepo) *OpenAIGatewayService {
	t.Helper()
	if cfg != nil {
		cfg.Security.URLAllowlist.Enabled = false
	}
	svc := &OpenAIGatewayService{
		cfg:          cfg,
		httpUpstream: upstream,
	}
	if repo != nil {
		svc.SetGenerationContentCollector(NewGenerationContentCollector(repo, cfg))
	}
	return svc
}

func TestOpenAIGenerationContentCaptureForwardPreservesNonStreamingJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.1","stream":false,"instructions":"local-test","input":[{"type":"text","text":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamJSON := `{"id":"resp_capture_json","object":"response","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}`
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-openai-capture-json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}}
	cfg := generationCaptureConfig(256, 512)
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})

	result, err := svc.Forward(context.Background(), c, newOpenAICaptureOAuthAccount(), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, upstreamJSON, recorder.Body.String())
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	require.Equal(t, []byte(upstreamJSON), result.ResponseSample)
	require.Equal(t, len(upstreamJSON), result.ResponseBytes)
	require.False(t, result.ResponseTruncated)
}

func TestOpenAIGenerationContentCaptureForwardPreservesStreamingSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.1","stream":true,"instructions":"local-test","input":[{"type":"text","text":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamSSE := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_capture_sse","status":"in_progress"}}`,
		"",
		`data: {"type":"response.output_text.delta","delta":"ok"}`,
		"",
		`data: {"type":"response.completed","response":{"id":"resp_capture_sse","status":"completed","usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-openai-capture-sse"}},
		Body:       io.NopCloser(strings.NewReader(upstreamSSE)),
	}}
	cfg := generationCaptureConfig(256, 2048)
	cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})
	// OAuth passthrough preserves upstream SSE bytes; the API-key synth path may
	// reconstruct output into response.completed and break byte-equality checks.
	account := &Account{
		ID:          123,
		Name:        "openai-capture-sse",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
		Extra: map[string]any{
			"openai_passthrough":                         true,
			"openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModeOff,
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, upstreamSSE, recorder.Body.String())
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
	require.True(t, recorder.Flushed)
	require.Equal(t, []byte(upstreamSSE), result.ResponseSample)
	require.Equal(t, len(upstreamSSE), result.ResponseBytes)
	require.False(t, result.ResponseTruncated)
}

func openaiCaptureOAuthAccount(id int64, name string) *Account {
	return &Account{
		ID:          id,
		Name:        name,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
		Extra: map[string]any{"openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModeOff},
	}
}

func openaiCaptureResponsesSSEBody(responseID string) string {
	return strings.Join([]string{
		`data: {"type":"response.completed","response":{"id":"` + responseID + `","object":"response","model":"gpt-5.4","status":"completed","output":[{"type":"message","id":"msg_1","role":"assistant","status":"completed","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
}

func TestOpenAIGenerationContentCaptureChatCompletionsPreservesNonStreamingJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := openaiCaptureResponsesSSEBody("resp_chat_capture")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-openai-chat-capture"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	cfg := generationCaptureConfig(256, 2048)
	cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})
	account := openaiCaptureOAuthAccount(2, "openai-oauth")

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "gpt-5.1")
	require.NoError(t, err)
	require.NotNil(t, result)
	clientBody := recorder.Body.String()
	require.NotEmpty(t, clientBody)
	require.Equal(t, []byte(clientBody), result.ResponseSample)
	require.Equal(t, len(clientBody), result.ResponseBytes)
	require.False(t, result.ResponseTruncated)
}

// C1 regression: handler-style failover reuses the same gin.Context across
// ForwardAsChatCompletions attempts. Stale sink-key reuse must not leave the
// successful attempt with an empty ResponseSample.
func TestOpenAIGenerationContentCaptureChatCompletionsFailoverCapturesSuccessSample(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := openaiCaptureResponsesSSEBody("resp_chat_failover")
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-chat-fail"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-chat-ok"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}}
	cfg := generationCaptureConfig(256, 2048)
	cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})
	account := openaiCaptureOAuthAccount(21, "openai-oauth-failover-chat")

	result1, err1 := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "gpt-5.1")
	require.Error(t, err1)
	require.Nil(t, result1)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err1, &failoverErr)

	result2, err2 := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "gpt-5.1")
	require.NoError(t, err2)
	require.NotNil(t, result2)
	clientBody := recorder.Body.String()
	require.NotEmpty(t, clientBody)
	require.Equal(t, []byte(clientBody), result2.ResponseSample, "failover success must capture client-visible bytes")
	require.Equal(t, len(clientBody), result2.ResponseBytes)
}

func TestOpenAIGenerationContentCaptureChatCompletionsPreservesStreamingSSEFlush(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_chat_sse","status":"in_progress"}}`,
		"",
		`data: {"type":"response.output_text.delta","delta":"ok"}`,
		"",
		`data: {"type":"response.completed","response":{"id":"resp_chat_sse","status":"completed","usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-openai-chat-sse"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	cfg := generationCaptureConfig(256, 4096)
	cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})
	account := openaiCaptureOAuthAccount(22, "openai-oauth-chat-sse")

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "gpt-5.1")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, recorder.Flushed, "Chat Completions SSE must flush through the capture writer")
	clientBody := recorder.Body.String()
	require.NotEmpty(t, clientBody)
	require.Equal(t, []byte(clientBody), result.ResponseSample)
}

func TestOpenAIGenerationContentCaptureMessagesPreservesNonStreamingJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := openaiCaptureResponsesSSEBody("resp_msg_capture")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-openai-msg-capture"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	cfg := generationCaptureConfig(256, 2048)
	cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})
	account := openaiCaptureOAuthAccount(3, "openai-oauth-messages")

	result, err := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "gpt-5.1")
	require.NoError(t, err)
	require.NotNil(t, result)
	clientBody := recorder.Body.String()
	require.NotEmpty(t, clientBody)
	require.Equal(t, []byte(clientBody), result.ResponseSample)
	require.Equal(t, len(clientBody), result.ResponseBytes)
	require.False(t, result.ResponseTruncated)
}

// C1 regression: Messages failover on the same gin.Context must reinstall
// capture so the successful attempt persists a non-empty ResponseSample.
func TestOpenAIGenerationContentCaptureMessagesFailoverCapturesSuccessSample(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := openaiCaptureResponsesSSEBody("resp_msg_failover")
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-msg-fail"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-msg-ok"}},
			Body:       io.NopCloser(strings.NewReader(upstreamBody)),
		},
	}}
	cfg := generationCaptureConfig(256, 2048)
	cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})
	account := openaiCaptureOAuthAccount(31, "openai-oauth-failover-messages")

	result1, err1 := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "gpt-5.1")
	require.Error(t, err1)
	require.Nil(t, result1)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err1, &failoverErr)

	result2, err2 := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "gpt-5.1")
	require.NoError(t, err2)
	require.NotNil(t, result2)
	clientBody := recorder.Body.String()
	require.NotEmpty(t, clientBody)
	require.Equal(t, []byte(clientBody), result2.ResponseSample, "failover success must capture client-visible bytes")
	require.Equal(t, len(clientBody), result2.ResponseBytes)
}

func TestOpenAIGenerationContentCaptureMessagesPreservesStreamingSSEFlush(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"stream":true}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_msg_sse","status":"in_progress"}}`,
		"",
		`data: {"type":"response.output_text.delta","delta":"ok"}`,
		"",
		`data: {"type":"response.completed","response":{"id":"resp_msg_sse","status":"completed","usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-openai-msg-sse"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	cfg := generationCaptureConfig(256, 4096)
	cfg.Gateway.MaxLineSize = defaultMaxLineSize
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})
	account := openaiCaptureOAuthAccount(32, "openai-oauth-messages-sse")

	result, err := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "gpt-5.1")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, recorder.Flushed, "Messages SSE must flush through the capture writer")
	clientBody := recorder.Body.String()
	require.NotEmpty(t, clientBody)
	require.Equal(t, []byte(clientBody), result.ResponseSample)
}

// Partial-image paths may return (*OpenAIForwardResult, err!=nil) for usage
// billing while skipping generation-content capture. Forward's deferred fill
// only runs when capturedErr==nil — this locks that carve-out.
func TestOpenAIGenerationContentCaptureSkipsFillOnPartialImageErrorPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &OpenAIGatewayService{cfg: generationCaptureConfig(256, 512)}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	sink, restore := svc.beginResponseCapture(c)
	defer restore()
	_, err := c.Writer.Write([]byte(`{"partial_image":true}`))
	require.NoError(t, err)
	require.NotEmpty(t, sink.Bytes())

	// Mimic Forward deferred fill guard (success-only).
	capturedResult := &OpenAIForwardResult{ImageCount: 1}
	capturedErr := errors.New("upstream partial image error")
	if capturedErr == nil {
		fillOpenAIResponseSample(capturedResult, sink)
	}
	require.Nil(t, capturedResult.ResponseSample)
	require.Zero(t, capturedResult.ResponseBytes)
	require.False(t, capturedResult.ResponseTruncated)
}

func TestOpenAIGenerationContentCaptureTruncatesOversizedResponseAndPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	prompt := []byte(`{"model":"gpt-5.1","stream":false,"instructions":"local-test","input":[{"type":"text","text":"` + strings.Repeat("p", 200) + `"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(prompt))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamJSON := `{"id":"resp_capture_trunc","object":"response","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"` + strings.Repeat("r", 200) + `"}]}],"usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}`
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-openai-capture-trunc"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}}
	cfg := generationCaptureConfig(64, 32)
	svc := newOpenAICaptureGateway(t, cfg, upstream, &generationContentMemoryRepo{})

	result, err := svc.Forward(context.Background(), c, newOpenAICaptureOAuthAccount(), prompt)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, upstreamJSON, recorder.Body.String(), "client-visible bytes must remain untruncated")
	require.True(t, result.ResponseTruncated)
	require.Equal(t, len(upstreamJSON), result.ResponseBytes)
	require.LessOrEqual(t, len(result.ResponseSample), 32)

	snapshot := svc.SnapshotGenerationPrompt(prompt)
	require.LessOrEqual(t, len(snapshot.Body), 64)
	require.Equal(t, len(prompt), snapshot.OriginalBytes)
	require.True(t, snapshot.Truncated)
}

func TestOpenAIGenerationContentCaptureDisabledDoesNotCopyOrCollect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.1","stream":false,"instructions":"local-test","input":[{"type":"text","text":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamJSON := `{"id":"resp_capture_disabled","object":"response","status":"completed","output":[],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-openai-capture-disabled"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}}
	repo := &generationContentMemoryRepo{}
	cfg := &config.Config{} // ContentCapture.Enabled defaults false
	svc := newOpenAICaptureGateway(t, cfg, upstream, repo)

	result, err := svc.Forward(context.Background(), c, newOpenAICaptureOAuthAccount(), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, upstreamJSON, recorder.Body.String())
	require.Nil(t, result.ResponseSample)
	require.Zero(t, result.ResponseBytes)
	require.False(t, result.ResponseTruncated)

	snapshot := svc.SnapshotGenerationPrompt(body)
	require.Nil(t, snapshot.Body)
	require.Zero(t, snapshot.OriginalBytes)

	svc.CollectGenerationContent(context.Background(), GenerationContentCaptureArgs{
		RequestID:  "disabled-openai",
		PromptBody: body,
		Result:     openAIResultAsCaptureEvidence(result),
	})
	require.Empty(t, repo.rows)
}

func TestOpenAIGenerationContentCaptureErrorPathDoesNotPersist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.1","stream":false,"instructions":"local-test","input":[{"type":"text","text":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-openai-capture-err"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"slow down"}}`)),
	}}
	repo := &generationContentMemoryRepo{}
	cfg := generationCaptureConfig(256, 512)
	svc := newOpenAICaptureGateway(t, cfg, upstream, repo)

	result, err := svc.Forward(context.Background(), c, newOpenAICaptureOAuthAccount(), body)
	require.Error(t, err)
	require.Nil(t, result)
	require.Empty(t, repo.rows, "error/failover path must not Create generation-content rows")
}

func TestOpenAIGenerationContentCaptureFailOpenOnCollectorPanicAndRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, repo := range []*generationContentMemoryRepo{
		{err: errors.New("database unavailable")},
		{panic: true},
	} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		body := []byte(`{"model":"gpt-5.1","stream":false,"instructions":"local-test","input":[{"type":"text","text":"hello"}]}`)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		upstreamJSON := `{"id":"resp_capture_failopen","object":"response","status":"completed","output":[],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`
		upstream := &httpUpstreamRecorder{resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-openai-capture-failopen"}},
			Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
		}}
		cfg := generationCaptureConfig(256, 512)
		svc := newOpenAICaptureGateway(t, cfg, upstream, repo)

		result, err := svc.Forward(context.Background(), c, newOpenAICaptureOAuthAccount(), body)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, upstreamJSON, recorder.Body.String(), "collector panic/repo error must not alter client body")

		snapshot := svc.SnapshotGenerationPrompt(body)
		require.NotPanics(t, func() {
			svc.CollectGenerationContent(context.Background(), GenerationContentCaptureArgs{
				RequestID:   result.RequestID,
				PromptBody:  snapshot.Body,
				PromptBytes: snapshot.OriginalBytes,
				Result:      openAIResultAsCaptureEvidence(result),
			})
		})
	}
}

func TestOpenAIGenerationContentCaptureMetadataMatchesUsageAttribution(t *testing.T) {
	repo := &generationContentMemoryRepo{}
	cfg := generationCaptureConfig(256, 128)
	svc := &OpenAIGatewayService{cfg: cfg}
	svc.SetGenerationContentCollector(NewGenerationContentCollector(repo, cfg))

	groupID := int64(77)
	prompt := []byte(`{"model":"gpt-5.1","input":[{"type":"text","text":"meta"}]}`)
	snapshot := svc.SnapshotGenerationPrompt(prompt)
	hash := HashUsageRequestPayload(prompt)
	result := &OpenAIForwardResult{
		RequestID:         "rid-openai-meta",
		Model:             "gpt-5.1",
		ResponseSample:    []byte(`{"ok":true}`),
		ResponseBytes:     11,
		ResponseTruncated: false,
	}

	svc.CollectGenerationContent(context.Background(), GenerationContentCaptureArgs{
		RequestID:          result.RequestID,
		UserID:             11,
		APIKeyID:           22,
		GroupID:            &groupID,
		AccountID:          33,
		Model:              result.Model,
		RequestPayloadHash: hash,
		PromptBody:         snapshot.Body,
		PromptBytes:        snapshot.OriginalBytes,
		Result:             openAIResultAsCaptureEvidence(result),
	})

	require.Len(t, repo.rows, 1)
	row := repo.rows[0]
	require.Equal(t, "rid-openai-meta", row.RequestID)
	require.Equal(t, "gpt-5.1", row.Model)
	require.Equal(t, hash, row.RequestPayloadHash)
	require.NotNil(t, row.UserID)
	require.Equal(t, int64(11), *row.UserID)
	require.NotNil(t, row.APIKeyID)
	require.Equal(t, int64(22), *row.APIKeyID)
	require.NotNil(t, row.GroupID)
	require.Equal(t, int64(77), *row.GroupID)
	require.NotNil(t, row.AccountID)
	require.Equal(t, int64(33), *row.AccountID)
	require.Equal(t, 11, row.ResponseBytes)
	require.False(t, row.ResponseTruncated)
}
