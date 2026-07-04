//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupAllowsModelScope(t *testing.T) {
	group := &Group{
		Platform:             PlatformAntigravity,
		SupportedModelScopes: []string{GroupModelScopeClaude},
	}
	require.True(t, GroupAllowsModelScope(group, "claude-sonnet-4", false))
	require.False(t, GroupAllowsModelScope(group, "gemini-2.5-pro", false))
	require.False(t, GroupAllowsModelScope(group, "gemini-2.5-flash-image", true))
}

func TestGroupAllowsModelScopeEmptyMeansUnrestricted(t *testing.T) {
	group := &Group{
		Platform:             PlatformAntigravity,
		SupportedModelScopes: nil,
	}
	require.True(t, GroupAllowsModelScope(group, "gemini-2.5-pro", false))
}
