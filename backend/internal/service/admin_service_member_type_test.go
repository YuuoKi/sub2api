//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type memberTypeLookupCountingRepo struct {
	*userRepoStub
	getCalls int
}

func (r *memberTypeLookupCountingRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	r.getCalls++
	return r.userRepoStub.GetByID(ctx, id)
}

func TestAdminServiceCreateUserAppliesMemberType(t *testing.T) {
	for _, tt := range []struct {
		name       string
		notes      string
		memberType string
		wantNotes  string
	}{
		{name: "tool", notes: " [工具] [工具] storyboard runner ", memberType: UserMemberTypeTool, wantNotes: "[工具] storyboard runner"},
		{name: "human", notes: " [工具] designer ", memberType: UserMemberTypeHuman, wantNotes: "designer"},
		{name: "omitted defaults human", notes: "employee", wantNotes: "employee"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userRepoStub{nextID: 42}
			svc := &adminServiceImpl{userRepo: repo}

			user, err := svc.CreateUser(context.Background(), &CreateUserInput{
				Email:      "member@example.com",
				Notes:      tt.notes,
				MemberType: tt.memberType,
			})

			require.NoError(t, err)
			require.Equal(t, tt.wantNotes, user.Notes)
			require.Len(t, repo.created, 1)
			require.Equal(t, tt.wantNotes, repo.created[0].Notes)
		})
	}
}

func TestAdminServiceCreateUserRejectsInvalidMemberTypeBeforeRepository(t *testing.T) {
	repo := &userRepoStub{nextID: 42}
	svc := &adminServiceImpl{userRepo: repo}

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:      "member@example.com",
		Notes:      "runner",
		MemberType: "robot",
	})

	require.ErrorIs(t, err, ErrInvalidUserMemberType)
	require.Empty(t, repo.created)
}

func TestAdminServiceUpdateUserPreservesMemberTypeAndNotesBody(t *testing.T) {
	for _, tt := range []struct {
		name       string
		current    string
		notes      *string
		memberType *string
		wantNotes  string
	}{
		{name: "notes only retains tool", current: "[工具] old runner", notes: memberTypeStringPointer("new runner"), wantNotes: "[工具] new runner"},
		{name: "human only retains body", current: "[工具] old runner", memberType: memberTypeStringPointer(UserMemberTypeHuman), wantNotes: "old runner"},
		{name: "tool only retains body", current: "employee notes", memberType: memberTypeStringPointer(UserMemberTypeTool), wantNotes: "[工具] employee notes"},
		{name: "both deduplicate", current: "employee notes", notes: memberTypeStringPointer(" [工具] [工具] new runner "), memberType: memberTypeStringPointer(UserMemberTypeTool), wantNotes: "[工具] new runner"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userRepoStub{user: &User{ID: 42, Email: "member@example.com", Notes: tt.current, Role: RoleUser, Status: StatusActive}}
			svc := &adminServiceImpl{userRepo: repo}

			user, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Notes: tt.notes, MemberType: tt.memberType})

			require.NoError(t, err)
			require.Equal(t, tt.wantNotes, user.Notes)
			require.Len(t, repo.updated, 1)
			require.Equal(t, tt.wantNotes, repo.updated[0].Notes)
		})
	}
}

func TestAdminServiceUpdateUserRejectsInvalidMemberTypeBeforeRepository(t *testing.T) {
	for _, tt := range []struct {
		name string
		user *User
	}{
		{name: "existing user", user: &User{ID: 42, Email: "member@example.com", Notes: "employee", Role: RoleUser, Status: StatusActive}},
		{name: "missing user"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &memberTypeLookupCountingRepo{userRepoStub: &userRepoStub{user: tt.user}}
			svc := &adminServiceImpl{userRepo: repo}
			invalid := "robot"

			_, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{MemberType: &invalid})

			require.ErrorIs(t, err, ErrInvalidUserMemberType)
			require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
			require.Equal(t, "INVALID_MEMBER_TYPE", infraerrors.Reason(err))
			require.Zero(t, repo.getCalls, "invalid member_type must fail before any user repository lookup")
			require.Empty(t, repo.updated)
		})
	}
}

func memberTypeStringPointer(value string) *string { return &value }
