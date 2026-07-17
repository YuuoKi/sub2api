package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

// Shared generation-content capture helpers used by GatewayService (Anthropic)
// and OpenAIGatewayService. Behavior is parameterized by cfg/collector; callers
// keep their own beginResponseCapture policies (OpenAI sink-key reuse vs
// Anthropic per-call install).

func generationContentCaptureEnabled(cfg *config.Config) bool {
	return cfg != nil && cfg.Gateway.ContentCapture.Enabled
}

func generationResponseCaptureMaxBytes(cfg *config.Config) int {
	if cfg != nil {
		return boundedGenerationBytes(cfg.Gateway.ContentCapture.ResponseMaxBytes, defaultGenerationResponseMaxBytes, maxGenerationResponseMaxBytes)
	}
	return defaultGenerationResponseMaxBytes
}

func generationPromptCaptureMaxBytes(cfg *config.Config) int {
	if cfg != nil {
		return boundedGenerationBytes(cfg.Gateway.ContentCapture.PromptMaxBytes, defaultGenerationPromptMaxBytes, maxGenerationPromptMaxBytes)
	}
	return defaultGenerationPromptMaxBytes
}

func installResponseCapture(c *gin.Context, maxBytes int) (*cappedSink, func()) {
	sink := newCappedSink(maxBytes)
	original := c.Writer
	c.Writer = &capturingResponseWriter{ResponseWriter: original, sink: sink}
	return sink, func() { c.Writer = original }
}

func snapshotGenerationPrompt(cfg *config.Config, body []byte) GenerationPromptSnapshot {
	if !generationContentCaptureEnabled(cfg) {
		return GenerationPromptSnapshot{}
	}
	limit := generationPromptCaptureMaxBytes(cfg)
	bounded := truncateValidUTF8(body, limit)
	return GenerationPromptSnapshot{
		Body:          append([]byte(nil), bounded...),
		OriginalBytes: len(body),
		Truncated:     len(bounded) < len(body),
	}
}

func collectGenerationContent(ctx context.Context, cfg *config.Config, collector *GenerationContentCollector, args GenerationContentCaptureArgs) {
	if collector == nil || !generationContentCaptureEnabled(cfg) {
		return
	}
	collector.Collect(ctx, args)
}

func fillCappedResponseSample(sample *[]byte, responseBytes *int, truncated *bool, sink *cappedSink) {
	if sample == nil || responseBytes == nil || truncated == nil || sink == nil {
		return
	}
	*sample = append([]byte(nil), sink.Bytes()...)
	*truncated = sink.Truncated()
	*responseBytes = sink.Total()
}
