package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserFromServiceAdmin_MapsActivityTimestamps(t *testing.T) {
	t.Parallel()

	lastLoginAt := time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC)
	lastActiveAt := lastLoginAt.Add(15 * time.Minute)
	lastUsedAt := lastLoginAt.Add(45 * time.Minute)

	out := UserFromServiceAdmin(&service.User{
		ID:           42,
		Email:        "admin@example.com",
		Username:     "admin",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		LastActiveAt: &lastActiveAt,
		LastUsedAt:   &lastUsedAt,
	})

	require.NotNil(t, out)
	require.NotNil(t, out.LastActiveAt)
	require.NotNil(t, out.LastUsedAt)
	require.WithinDuration(t, lastActiveAt, *out.LastActiveAt, time.Second)
	require.WithinDuration(t, lastUsedAt, *out.LastUsedAt, time.Second)
}

func TestUserFromServiceAdminSeparatesMemberTypeFromNotesStorage(t *testing.T) {
	tool := UserFromServiceAdmin(&service.User{Notes: "  [工具] [工具] storyboard runner  "})
	require.NotNil(t, tool)
	require.Equal(t, service.UserMemberTypeTool, tool.MemberType)
	require.Equal(t, "storyboard runner", tool.Notes)
	encoded, err := json.Marshal(tool)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "[工具]", "DTO must not leak the notes storage prefix")
	require.Contains(t, string(encoded), `"member_type":"tool"`)

	human := UserFromServiceAdmin(&service.User{Notes: " designer "})
	require.NotNil(t, human)
	require.Equal(t, service.UserMemberTypeHuman, human.MemberType)
	require.Equal(t, "designer", human.Notes)
}
