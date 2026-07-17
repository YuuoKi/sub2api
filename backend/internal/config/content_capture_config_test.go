package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentCaptureDefaultsDisabledWithSafeBounds(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.Gateway.ContentCapture.Enabled)
	require.Equal(t, 256*1024, cfg.Gateway.ContentCapture.PromptMaxBytes)
	require.Equal(t, 64*1024, cfg.Gateway.ContentCapture.ResponseMaxBytes)
}

func TestContentCaptureLoadsTypedEnvironmentValues(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GATEWAY_CONTENT_CAPTURE_ENABLED", "true")
	t.Setenv("GATEWAY_CONTENT_CAPTURE_PROMPT_MAX_BYTES", "2048")
	t.Setenv("GATEWAY_CONTENT_CAPTURE_RESPONSE_MAX_BYTES", "1024")
	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.Gateway.ContentCapture.Enabled)
	require.Equal(t, 2048, cfg.Gateway.ContentCapture.PromptMaxBytes)
	require.Equal(t, 1024, cfg.Gateway.ContentCapture.ResponseMaxBytes)
}
