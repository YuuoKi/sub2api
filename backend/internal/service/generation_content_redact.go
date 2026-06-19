package service

import (
	"regexp"

	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

// generationRedactionVersion 记录脱敏规则版本，写入 ai_generation_content.redaction_version。
// 任何脱敏规则变更都应 bump 此值，便于后续审计/回填判断某行经过了哪套规则。
const generationRedactionVersion = 1

var (
	// rePIIEmail 匹配常见 email。
	rePIIEmail = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	// rePIIPhone 保守匹配电话号码：以数字开头/结尾，中间为数字/分隔符（不含换行，避免跨行吞并）。
	// 过度脱敏（误伤长数字 ID）是可接受的取舍——采集用于分析而非保真。
	rePIIPhone = regexp.MustCompile(`\+?\d[\d().\-\t ]{7,}\d`)
)

// redactGenerationPII 在结构化/密钥脱敏之后，补一层自由文本 PII（email/电话）脱敏。
// 现有 logredact / content_moderation 脱敏均不覆盖自由文本 PII，这里补齐缺口。
func redactGenerationPII(s string) string {
	if s == "" {
		return s
	}
	s = rePIIEmail.ReplaceAllString(s, "[EMAIL]")
	s = rePIIPhone.ReplaceAllString(s, "[PHONE]")
	return s
}

// redactGenerationPrompt 对请求体（JSON）做入库前脱敏：
// 结构化敏感键(RedactJSON) → 自由文本 PII → 兜底密钥/Token 模式(content_moderation)。
func redactGenerationPrompt(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	out := logredact.RedactJSON(body)
	out = redactGenerationPII(out)
	out = redactContentModerationSecrets(out)
	return out
}

// redactGenerationResponse 对响应抽样（SSE/JSON 文本）做入库前脱敏：
// 轻量文本脱敏(RedactText) → 自由文本 PII → 兜底密钥/Token 模式。
func redactGenerationResponse(sample []byte) string {
	if len(sample) == 0 {
		return ""
	}
	out := logredact.RedactText(string(sample))
	out = redactGenerationPII(out)
	out = redactContentModerationSecrets(out)
	return out
}
