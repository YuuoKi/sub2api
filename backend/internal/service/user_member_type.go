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

func UserMemberTypeFromNotes(notes string) string {
	if strings.HasPrefix(strings.TrimSpace(notes), toolMemberNotesPrefix) {
		return UserMemberTypeTool
	}
	return UserMemberTypeHuman
}

func UserMemberNotesBody(notes string) string {
	body := strings.TrimSpace(notes)
	for strings.HasPrefix(body, toolMemberNotesPrefix) {
		body = strings.TrimSpace(strings.TrimPrefix(body, toolMemberNotesPrefix))
	}
	return body
}

func ApplyUserMemberTypeToNotes(notes, memberType string) (string, error) {
	normalized := normalizeUserMemberType(memberType)
	if normalized == "" {
		return notes, nil
	}
	normalized, err := normalizeAndValidateUserMemberType(normalized)
	if err != nil {
		return "", err
	}
	body := UserMemberNotesBody(notes)
	switch normalized {
	case UserMemberTypeHuman:
		return body, nil
	case UserMemberTypeTool:
		if body == "" {
			return toolMemberNotesPrefix, nil
		}
		return toolMemberNotesPrefix + " " + body, nil
	}
	return "", ErrInvalidUserMemberType
}

func MergeUserMemberNotes(currentNotes string, notes, memberType *string) (string, error) {
	targetType := UserMemberTypeFromNotes(currentNotes)
	if memberType != nil {
		var err error
		targetType, err = normalizeAndValidateUserMemberType(*memberType)
		if err != nil {
			return "", err
		}
	}
	bodySource := currentNotes
	if notes != nil {
		bodySource = *notes
	}
	return ApplyUserMemberTypeToNotes(bodySource, targetType)
}

func normalizeUserMemberType(memberType string) string {
	return strings.ToLower(strings.TrimSpace(memberType))
}

func normalizeAndValidateUserMemberType(memberType string) (string, error) {
	normalized := normalizeUserMemberType(memberType)
	if normalized != UserMemberTypeHuman && normalized != UserMemberTypeTool {
		return "", ErrInvalidUserMemberType
	}
	return normalized, nil
}
