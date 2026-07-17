package service

import (
	"context"

	"github.com/gin-gonic/gin"
)

// openAIResponseCaptureSinkKey stores a pre-installed capture sink so Forward
// can reuse the tee when the Responses handler installs capture before compact
// keepalive (required chain: real←capture←keepalive).
const openAIResponseCaptureSinkKey = "openai_response_capture_sink"

// BeginResponseCapture installs the response tee when content capture is enabled.
// Call this before StartOpenAICompactSSEKeepalive so heartbeats and response
// bytes share one chain (real←capture←keepalive). Disabled config is a no-op.
func (s *OpenAIGatewayService) BeginResponseCapture(c *gin.Context) func() {
	_, restore := s.beginResponseCapture(c)
	return restore
}

func (s *OpenAIGatewayService) beginResponseCapture(c *gin.Context) (*cappedSink, func()) {
	if c == nil || !s.contentCaptureEnabled() {
		return nil, func() {}
	}
	if existing, ok := c.Get(openAIResponseCaptureSinkKey); ok {
		if sink, ok := existing.(*cappedSink); ok && sink != nil {
			// Already installed (Responses compact path). Caller must not restore
			// the writer or clear the key — the handler-owned restore owns both.
			return sink, func() {}
		}
	}
	sink, unwrap := installResponseCapture(c, s.responseCaptureMaxBytes())
	c.Set(openAIResponseCaptureSinkKey, sink)
	// Restore must clear the sink key. Chat/Messages failover reuses the same
	// gin.Context across attempts; leaving the key set after unwrap causes the
	// next beginResponseCapture to return a stale sink with a noop restore and
	// no writer wrap, so success bytes bypass capture (empty ResponseSample).
	return sink, func() {
		unwrap()
		c.Set(openAIResponseCaptureSinkKey, nil)
	}
}

func (s *OpenAIGatewayService) responseCaptureMaxBytes() int {
	if s != nil {
		return generationResponseCaptureMaxBytes(s.cfg)
	}
	return defaultGenerationResponseMaxBytes
}

func (s *OpenAIGatewayService) SetGenerationContentCollector(collector *GenerationContentCollector) {
	if s != nil {
		s.generationCollector = collector
	}
}

func (s *OpenAIGatewayService) contentCaptureEnabled() bool {
	return s != nil && generationContentCaptureEnabled(s.cfg)
}

func (s *OpenAIGatewayService) SnapshotGenerationPrompt(body []byte) GenerationPromptSnapshot {
	if s == nil {
		return GenerationPromptSnapshot{}
	}
	return snapshotGenerationPrompt(s.cfg, body)
}

func (s *OpenAIGatewayService) CollectGenerationContent(ctx context.Context, args GenerationContentCaptureArgs) {
	if s == nil {
		return
	}
	collectGenerationContent(ctx, s.cfg, s.generationCollector, args)
}

func fillOpenAIResponseSample(result *OpenAIForwardResult, sink *cappedSink) {
	if result == nil {
		return
	}
	fillCappedResponseSample(&result.ResponseSample, &result.ResponseBytes, &result.ResponseTruncated, sink)
}

// openAIResultAsCaptureEvidence adapts OpenAI forward evidence for the shared
// GenerationContentCollector (reuses ForwardResult sample fields only).
func openAIResultAsCaptureEvidence(result *OpenAIForwardResult) *ForwardResult {
	if result == nil {
		return nil
	}
	return &ForwardResult{
		ResponseSample:    result.ResponseSample,
		ResponseBytes:     result.ResponseBytes,
		ResponseTruncated: result.ResponseTruncated,
	}
}

// OpenAIResultAsCaptureEvidence is the handler-facing adapter for CollectGenerationContent.
func OpenAIResultAsCaptureEvidence(result *OpenAIForwardResult) *ForwardResult {
	return openAIResultAsCaptureEvidence(result)
}
