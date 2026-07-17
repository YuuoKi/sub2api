package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimulationSVGEscapesPromptAndRejectsScriptInjection(t *testing.T) {
	task := &VideoTask{
		ID:       42,
		Provider: VideoProviderMock,
		Model:    VideoModelMockVideoV1,
		Status:   VideoStatusSucceeded,
		Prompt:   `<script>alert("xss")</script>&"'`,
	}
	body := RenderSimulationSVG(task)
	svg := string(body)
	require.Contains(t, svg, "模拟视频结果")
	require.Contains(t, svg, "42")
	require.NotContains(t, svg, `<script>alert("xss")</script>`)
	require.Contains(t, svg, "&lt;script&gt;")
	require.Contains(t, svg, "&amp;")
	require.True(t, strings.HasPrefix(strings.TrimSpace(svg), "<svg") || strings.Contains(svg, "<svg"))
	require.NotContains(t, strings.ToLower(svg), "<script")
}

func TestSimulationResultContractIsLabeledImageNotFakeVideo(t *testing.T) {
	task := &VideoTask{ID: 7, Prompt: "poster", Status: VideoStatusSucceeded, Provider: VideoProviderMock}
	result := BuildSimulationResult(task)
	require.Equal(t, "image", result.MediaKind)
	require.Equal(t, "image/svg+xml", result.ContentType)
	require.Equal(t, "模拟视频结果", result.Label)
	require.Contains(t, result.Filename, "7")
	require.NotContains(t, result.Filename, `\`)
	require.NotContains(t, string(result.Body), "assets/video/")
}
