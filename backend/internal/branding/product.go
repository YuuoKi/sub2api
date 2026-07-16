package branding

import "strings"

const (
	DefaultProductName         = "无界 · 企业 AI 管理中台"
	UpstreamDefaultProductName = "Sub2API"
)

// ResolveProductName keeps explicit administrator branding while replacing
// an empty or untouched upstream name with the Wujie product identity.
func ResolveProductName(siteName string) string {
	normalized := strings.TrimSpace(siteName)
	if normalized == "" || normalized == UpstreamDefaultProductName {
		return DefaultProductName
	}
	return normalized
}
