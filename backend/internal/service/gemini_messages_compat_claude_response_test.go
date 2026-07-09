package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertGeminiToClaudeMessage_ReturnsInlineDataImageAndSignatures(t *testing.T) {
	geminiResp := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{
							"text":             "here is an image",
							"thoughtSignature": "sig-text",
						},
						map[string]any{
							"thoughtSignature": "sig-image",
							"inlineData": map[string]any{
								"mimeType": "image/png",
								"data":     "iVBORw0KGgo=",
							},
						},
						map[string]any{
							"thoughtSignature": "sig-tool",
							"functionCall": map[string]any{
								"name": "lookup",
								"args": map[string]any{"q": "x"},
							},
						},
						map[string]any{
							"inline_data": map[string]any{
								"mime_type": "image/jpeg",
								"data":      "/9j/4AAQ=",
							},
						},
					},
				},
				"finishReason": "STOP",
			},
		},
	}

	msg, usage := convertGeminiToClaudeMessage(geminiResp, "claude-haiku-4-5-20251001", nil)
	require.NotNil(t, usage)

	content, ok := msg["content"].([]any)
	require.True(t, ok)
	require.Len(t, content, 4)

	textBlock, ok := content[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "text", textBlock["type"])
	require.Equal(t, "here is an image", textBlock["text"])
	require.Equal(t, "sig-text", textBlock["signature"])

	imageBlock, ok := content[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "image", imageBlock["type"])
	require.Equal(t, "sig-image", imageBlock["signature"])
	src, ok := imageBlock["source"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "base64", src["type"])
	require.Equal(t, "image/png", src["media_type"])
	require.Equal(t, "iVBORw0KGgo=", src["data"])

	toolBlock, ok := content[2].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "tool_use", toolBlock["type"])
	require.Equal(t, "lookup", toolBlock["name"])
	require.Equal(t, "sig-tool", toolBlock["signature"])

	imageSnake, ok := content[3].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "image", imageSnake["type"])
	_, hasSig := imageSnake["signature"]
	require.False(t, hasSig)
	src2, ok := imageSnake["source"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "image/jpeg", src2["media_type"])
	require.Equal(t, "/9j/4AAQ=", src2["data"])

	require.Equal(t, "tool_use", msg["stop_reason"])
}

func TestClaudeImageBlockFromGeminiPart_ExtractsImageAndThoughtSignature(t *testing.T) {
	block, ok := claudeImageBlockFromGeminiPart(map[string]any{
		"thought_signature": "stream-sig",
		"inlineData": map[string]any{
			"mimeType": "image/webp",
			"data":     "UklGRg==",
		},
	})
	require.True(t, ok)
	require.Equal(t, "image", block["type"])
	require.Equal(t, "stream-sig", block["signature"])
	src := block["source"].(map[string]any)
	require.Equal(t, "base64", src["type"])
	require.Equal(t, "image/webp", src["media_type"])
	require.Equal(t, "UklGRg==", src["data"])

	_, ok = claudeImageBlockFromGeminiPart(map[string]any{
		"inlineData": map[string]any{
			"mimeType": "application/pdf",
			"data":     "JVBERi0=",
		},
	})
	require.False(t, ok)
}

func TestConvertGeminiToClaudeMessage_ImageSignatureRoundTripToGemini(t *testing.T) {
	geminiResp := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{
							"thoughtSignature": "round-trip-sig",
							"inlineData": map[string]any{
								"mimeType": "image/png",
								"data":     "abc123",
							},
						},
					},
				},
			},
		},
	}

	claudeMsg, _ := convertGeminiToClaudeMessage(geminiResp, "claude-haiku-4-5-20251001", nil)
	content := claudeMsg["content"].([]any)
	require.Len(t, content, 1)

	claudeReq := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 16,
		"messages": []any{
			map[string]any{
				"role":    "assistant",
				"content": content,
			},
		},
	}
	body, err := json.Marshal(claudeReq)
	require.NoError(t, err)

	out, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(out, &parsed))
	contents := parsed["contents"].([]any)
	require.NotEmpty(t, contents)
	modelMsg := contents[0].(map[string]any)
	require.Equal(t, "model", modelMsg["role"])
	parts := modelMsg["parts"].([]any)
	require.Len(t, parts, 1)
	part := parts[0].(map[string]any)
	require.Equal(t, "round-trip-sig", part["thoughtSignature"])
	inline := part["inlineData"].(map[string]any)
	require.Equal(t, "image/png", inline["mimeType"])
	require.Equal(t, "abc123", inline["data"])
}
