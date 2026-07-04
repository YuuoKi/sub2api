package service

import "strings"

const (
	GroupModelScopeClaude      = "claude"
	GroupModelScopeGeminiText  = "gemini_text"
	GroupModelScopeGeminiImage = "gemini_image"
)

const groupModelScopePermissionMessage = "Requested model is not enabled for this group"

// GroupModelScopePermissionMessage returns the stable end-user error text for scope violations.
func GroupModelScopePermissionMessage() string {
	return groupModelScopePermissionMessage
}

// ResolveGroupModelScope maps a requested model to an antigravity scope bucket.
func ResolveGroupModelScope(model string, imageIntent bool) string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if imageIntent || modelMatchesGeminiImageScope(normalized) {
		return GroupModelScopeGeminiImage
	}
	if strings.Contains(normalized, "claude") {
		return GroupModelScopeClaude
	}
	if strings.Contains(normalized, "gemini") {
		return GroupModelScopeGeminiText
	}
	return ""
}

func modelMatchesGeminiImageScope(model string) bool {
	return strings.Contains(model, "gemini") && strings.Contains(model, "image")
}

// GroupAllowsModelScope enforces SupportedModelScopes for antigravity groups.
func GroupAllowsModelScope(group *Group, model string, imageIntent bool) bool {
	if group == nil || group.Platform != PlatformAntigravity || len(group.SupportedModelScopes) == 0 {
		return true
	}
	scope := ResolveGroupModelScope(model, imageIntent)
	if scope == "" {
		return true
	}
	for _, allowed := range group.SupportedModelScopes {
		if strings.TrimSpace(allowed) == scope {
			return true
		}
	}
	return false
}
