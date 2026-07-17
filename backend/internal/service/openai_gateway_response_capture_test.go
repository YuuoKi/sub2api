package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIBeginResponseCaptureDisabledDoesNotReplaceWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	original := c.Writer

	sink, restore := svc.beginResponseCapture(c)
	defer restore()
	if sink != nil || c.Writer != original {
		t.Fatal("disabled OpenAI capture must not allocate or replace writer")
	}
}

func TestOpenAICompactKeepaliveWithCapturePreservesAdjustedWrittenSizeAndWriterChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, rec := newCompactBridgeTestContext(t, true)
	// Response cap must leave room for ~200ms of 10ms keepalive comments plus payload.
	svc := &OpenAIGatewayService{cfg: generationCaptureConfig(128, 4096)}

	// Required Phase 2 order: real ← capture ← keepalive (install capture BEFORE keepalive).
	sink, restore := svc.beginResponseCapture(c)
	defer restore()
	require.NotNil(t, sink)

	captureWriter, ok := c.Writer.(*capturingResponseWriter)
	require.True(t, ok, "capture must wrap the real ResponseWriter before keepalive starts")
	realWriter := captureWriter.ResponseWriter

	stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
	defer stop()

	keepaliveWriter, ok := c.Writer.(*openAICompactKeepaliveWriter)
	require.True(t, ok, "keepalive must wrap capture so chain is real←capture←keepalive")
	innerCapture, ok := keepaliveWriter.ResponseWriter.(*capturingResponseWriter)
	require.True(t, ok, "keepalive inner writer must be the capture tee")
	require.Equal(t, realWriter, innerCapture.ResponseWriter)

	before := OpenAICompactKeepaliveAdjustedWrittenSize(c)
	waitForKeepaliveBeats()
	require.Equal(t, before, OpenAICompactKeepaliveAdjustedWrittenSize(c), "keepalive comment bytes must still be deducted with capture installed")
	require.Contains(t, rec.Body.String(), ": keepalive\n\n")
	require.Contains(t, string(sink.Bytes()), ": keepalive\n\n", "heartbeats must hit the capture sink")

	_, err := c.Writer.Write([]byte("real-bytes"))
	require.NoError(t, err)
	require.Equal(t, len("real-bytes"), OpenAICompactKeepaliveAdjustedWrittenSize(c))
	require.Contains(t, string(sink.Bytes()), "real-bytes")
}

func TestOpenAIFillResponseSampleOnForwardResult(t *testing.T) {
	sink := newCappedSink(8)
	_, _ = sink.Write([]byte(strings.Repeat("B", 20)))
	result := &OpenAIForwardResult{}
	fillOpenAIResponseSample(result, sink)
	require.Equal(t, []byte(strings.Repeat("B", 8)), result.ResponseSample)
	require.Equal(t, 20, result.ResponseBytes)
	require.True(t, result.ResponseTruncated)
}

// C1: Chat/Messages failover restores the writer but must also clear the sink
// key so the next attempt re-wraps. Stale-key reuse returns a sink with a noop
// restore and no writer wrap → success bytes bypass capture.
func TestOpenAIBeginResponseCaptureRestoreClearsSinkKeyForFailoverReinstall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &OpenAIGatewayService{cfg: generationCaptureConfig(128, 256)}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	sink1, restore1 := svc.beginResponseCapture(c)
	require.NotNil(t, sink1)
	restore1()

	sink2, restore2 := svc.beginResponseCapture(c)
	defer restore2()
	require.NotNil(t, sink2)
	require.NotSame(t, sink1, sink2, "failover reinstall must allocate a fresh sink after restore")

	_, err := c.Writer.Write([]byte("success-bytes"))
	require.NoError(t, err)
	require.Equal(t, []byte("success-bytes"), sink2.Bytes(), "second-attempt client bytes must hit the reinstalled sink")
}

// Responses compact path: handler installs capture before keepalive; Forward
// must reuse that sink without re-wrapping inside the keepalive chain.
func TestOpenAIBeginResponseCaptureReusesHandlerInstalledSinkWithoutRewrap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := newCompactBridgeTestContext(t, true)
	svc := &OpenAIGatewayService{cfg: generationCaptureConfig(128, 4096)}

	handlerSink, restoreHandler := svc.beginResponseCapture(c)
	defer restoreHandler()
	require.NotNil(t, handlerSink)

	stop := StartOpenAICompactSSEKeepalive(c, keepaliveTestInterval)
	defer stop()

	forwardSink, restoreForward := svc.beginResponseCapture(c)
	require.Same(t, handlerSink, forwardSink)
	restoreForward() // noop; must not clear handler-owned key

	forwardSink2, restoreForward2 := svc.beginResponseCapture(c)
	require.Same(t, handlerSink, forwardSink2)
	restoreForward2()

	_, err := c.Writer.Write([]byte("compact-ok"))
	require.NoError(t, err)
	require.Contains(t, string(handlerSink.Bytes()), "compact-ok")
}
