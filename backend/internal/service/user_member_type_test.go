package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserMemberTypeFromNotes(t *testing.T) {
	for _, tt := range []struct {
		name  string
		notes string
		want  string
	}{
		{name: "empty is human", notes: "", want: UserMemberTypeHuman},
		{name: "plain notes are human", notes: "designer", want: UserMemberTypeHuman},
		{name: "tool prefix", notes: "[工具] storyboard bot", want: UserMemberTypeTool},
		{name: "leading whitespace", notes: "  [工具] automation", want: UserMemberTypeTool},
		{name: "duplicate prefix", notes: "[工具] [工具] runner", want: UserMemberTypeTool},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, UserMemberTypeFromNotes(tt.notes))
		})
	}
}

func TestUserMemberNotesBodyHidesAndDeduplicatesStoragePrefix(t *testing.T) {
	require.Equal(t, "storyboard bot", UserMemberNotesBody("  [工具] [工具] storyboard bot  "))
	require.Equal(t, "designer", UserMemberNotesBody(" designer "))
}

func TestApplyUserMemberTypeToNotesPreservesBodyAndDeduplicatesPrefix(t *testing.T) {
	toolNotes, err := ApplyUserMemberTypeToNotes(" [工具] [工具] storyboard bot ", UserMemberTypeTool)
	require.NoError(t, err)
	require.Equal(t, "[工具] storyboard bot", toolNotes)

	humanNotes, err := ApplyUserMemberTypeToNotes(" [工具] storyboard bot ", UserMemberTypeHuman)
	require.NoError(t, err)
	require.Equal(t, "storyboard bot", humanNotes)

	_, err = ApplyUserMemberTypeToNotes("notes", "robot")
	require.ErrorIs(t, err, ErrInvalidUserMemberType)
}

func TestMergeUserMemberNotesPreservesUnchangedSide(t *testing.T) {
	newBody := "renamed runner"
	merged, err := MergeUserMemberNotes("[工具] old runner", &newBody, nil)
	require.NoError(t, err)
	require.Equal(t, "[工具] renamed runner", merged, "notes-only update must retain tool classification")

	human := UserMemberTypeHuman
	merged, err = MergeUserMemberNotes("[工具] old runner", nil, &human)
	require.NoError(t, err)
	require.Equal(t, "old runner", merged, "member-type-only update must retain notes body")

	tool := UserMemberTypeTool
	duplicateBody := " [工具] [工具] new runner "
	merged, err = MergeUserMemberNotes("human notes", &duplicateBody, &tool)
	require.NoError(t, err)
	require.Equal(t, "[工具] new runner", merged)

	empty := ""
	_, err = MergeUserMemberNotes("notes", nil, &empty)
	require.ErrorIs(t, err, ErrInvalidUserMemberType, "an explicitly supplied empty member_type is invalid")
}
