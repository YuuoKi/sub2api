package service

import (
	"encoding/xml"
	"fmt"
	"strings"
	"unicode/utf8"
)

const simulationResultLabel = "模拟视频结果"

// RenderSimulationSVG builds a deterministic labeled SVG poster for a mock task.
func RenderSimulationSVG(task *VideoTask) []byte {
	id := int64(0)
	prompt := ""
	if task != nil {
		id = task.ID
		prompt = task.Prompt
	}
	excerpt := truncateRunes(prompt, 80)
	escapedPrompt := xmlEscape(excerpt)
	escapedLabel := xmlEscape(simulationResultLabel)
	svg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="720" viewBox="0 0 1280 720">
  <rect width="1280" height="720" fill="#1a1a1a"/>
  <text x="64" y="120" fill="#f5f5f5" font-size="48" font-family="sans-serif">%s</text>
  <text x="64" y="200" fill="#cccccc" font-size="28" font-family="sans-serif">task_id=%d</text>
  <text x="64" y="280" fill="#aaaaaa" font-size="24" font-family="sans-serif">%s</text>
</svg>
`, escapedLabel, id, escapedPrompt)
	return []byte(svg)
}

// BuildSimulationResult returns the authenticated preview/download contract.
func BuildSimulationResult(task *VideoTask) *VideoSimulationResult {
	id := int64(0)
	if task != nil {
		id = task.ID
	}
	body := RenderSimulationSVG(task)
	return &VideoSimulationResult{
		TaskID:      id,
		MediaKind:   "image",
		ContentType: "image/svg+xml",
		Filename:    fmt.Sprintf("simulation-task-%d.svg", id),
		Label:       simulationResultLabel,
		Body:        body,
	}
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
