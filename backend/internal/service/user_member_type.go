package service

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	UserMemberTypeHuman = "human"
	UserMemberTypeTool  = "tool"

	toolMemberNotesPrefix = "[工具]"
)

var ErrInvalidUserMemberType = infraerrors.BadRequest("INVALID_MEMBER_TYPE", "member_type must be human or tool")

// UserMemberTypeFromNotes derives the lightweight member classification without
// adding a schema column. P0-3 stores tool ownership in users.notes by contract.
func UserMemberTypeFromNotes(notes string) string {
	if strings.HasPrefix(strings.TrimSpace(notes), toolMemberNotesPrefix) {
		return UserMemberTypeTool
	}
	return UserMemberTypeHuman
}

func ApplyUserMemberTypeToNotes(notes string, memberType string) (string, error) {
	switch normalizeUserMemberType(memberType) {
	case "":
		return notes, nil
	case UserMemberTypeHuman:
		return stripToolMemberNotesPrefix(notes), nil
	case UserMemberTypeTool:
		return prefixToolMemberNotes(notes), nil
	default:
		return "", ErrInvalidUserMemberType
	}
}

func normalizeUserMemberType(memberType string) string {
	return strings.ToLower(strings.TrimSpace(memberType))
}

func prefixToolMemberNotes(notes string) string {
	body := stripToolMemberNotesPrefix(notes)
	if body == "" {
		return toolMemberNotesPrefix
	}
	return toolMemberNotesPrefix + " " + body
}

func stripToolMemberNotesPrefix(notes string) string {
	trimmed := strings.TrimSpace(notes)
	if !strings.HasPrefix(trimmed, toolMemberNotesPrefix) {
		return trimmed
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, toolMemberNotesPrefix))
}
