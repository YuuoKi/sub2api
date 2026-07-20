//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
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

func TestLANAdminCreateUserAllowsOnlyToolServiceIdentitiesOrAdministrators(t *testing.T) {
	newService := func(repo *userRepoStub) *adminServiceImpl {
		settings := NewSettingService(nil, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})
		return &adminServiceImpl{userRepo: repo, settingService: settings}
	}

	for _, tt := range []struct {
		name       string
		input      CreateUserInput
		wantErr    bool
		wantReason string
		wantCred   bool
	}{
		{
			name:       "omitted member type defaults to forbidden human user",
			input:      CreateUserInput{Email: "employee@example.com"},
			wantErr:    true,
			wantReason: "LAN_ADMIN_HUMAN_USER_DISABLED",
		},
		{
			name:       "explicit human user is forbidden",
			input:      CreateUserInput{Email: "employee@example.com", MemberType: UserMemberTypeHuman, Role: RoleUser},
			wantErr:    true,
			wantReason: "LAN_ADMIN_HUMAN_USER_DISABLED",
		},
		{
			name:     "tool service identity has no interactive temporary credential",
			input:    CreateUserInput{Email: "qcanvas@wujie.local", MemberType: UserMemberTypeTool, Role: RoleUser},
			wantCred: false,
		},
		{
			name:     "independent administrator keeps bootstrap credential path",
			input:    CreateUserInput{Email: "executive@example.com", MemberType: UserMemberTypeHuman, Role: RoleAdmin},
			wantCred: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userRepoStub{nextID: 42}
			input := tt.input
			zeroBalance := 0.0
			input.Balance = &zeroBalance
			user, err := newService(repo).CreateUser(context.Background(), &input)

			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, http.StatusForbidden, infraerrors.Code(err))
				require.Equal(t, tt.wantReason, infraerrors.Reason(err))
				require.Empty(t, repo.created)
				return
			}

			require.NoError(t, err)
			require.Len(t, repo.created, 1)
			if tt.wantCred {
				require.NotNil(t, user.InitialCredential)
				require.True(t, user.MustChangePassword)
				require.NotNil(t, user.TemporaryPasswordExpiresAt)
			} else {
				require.Nil(t, user.InitialCredential)
				require.False(t, user.MustChangePassword)
				require.Nil(t, user.TemporaryPasswordExpiresAt)
			}
		})
	}
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

func TestLANAdminUpdateUserPreservesServiceIdentityAndAdministratorBoundary(t *testing.T) {
	newService := func(user *User) (*adminServiceImpl, *userRepoStub) {
		repo := &userRepoStub{user: user}
		settings := NewSettingService(nil, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})
		return &adminServiceImpl{userRepo: repo, settingService: settings}, repo
	}

	for _, tt := range []struct {
		name       string
		user       *User
		input      UpdateUserInput
		wantErr    bool
		wantReason string
	}{
		{
			name:       "tool service identity cannot become a human user",
			user:       &User{ID: 42, Email: "qcanvas@wujie.local", Notes: "[工具] QCanvas", Role: RoleUser, Status: StatusActive},
			input:      UpdateUserInput{MemberType: memberTypeStringPointer(UserMemberTypeHuman)},
			wantErr:    true,
			wantReason: "LAN_ADMIN_HUMAN_USER_DISABLED",
		},
		{
			name:       "human administrator cannot be demoted into a human user",
			user:       &User{ID: 42, Email: "executive@example.com", Notes: "executive", Role: RoleAdmin, Status: StatusActive},
			input:      UpdateUserInput{Role: RoleUser},
			wantErr:    true,
			wantReason: "LAN_ADMIN_HUMAN_USER_DISABLED",
		},
		{
			name:       "tool service identity cannot become an administrator without becoming human",
			user:       &User{ID: 42, Email: "qcanvas@wujie.local", Notes: "[工具] QCanvas", Role: RoleUser, Status: StatusActive},
			input:      UpdateUserInput{Role: RoleAdmin},
			wantErr:    true,
			wantReason: "LAN_ADMIN_ADMIN_MUST_BE_HUMAN",
		},
		{
			name:       "tool service identity cannot be converted into a human administrator",
			user:       &User{ID: 42, Email: "qcanvas@wujie.local", Notes: "[工具] QCanvas", Role: RoleUser, Status: StatusActive},
			input:      UpdateUserInput{Role: RoleAdmin, MemberType: memberTypeStringPointer(UserMemberTypeHuman)},
			wantErr:    true,
			wantReason: "LAN_ADMIN_IDENTITY_KIND_IMMUTABLE",
		},
		{
			name:       "human administrator cannot be converted into a tool service identity",
			user:       &User{ID: 42, Email: "executive@example.com", Notes: "executive", Role: RoleAdmin, Status: StatusActive},
			input:      UpdateUserInput{Role: RoleUser, MemberType: memberTypeStringPointer(UserMemberTypeTool)},
			wantErr:    true,
			wantReason: "LAN_ADMIN_IDENTITY_KIND_IMMUTABLE",
		},
		{
			name:  "tool service identity can update ordinary metadata",
			user:  &User{ID: 42, Email: "qcanvas@wujie.local", Notes: "[工具] QCanvas", Role: RoleUser, Status: StatusActive},
			input: UpdateUserInput{Notes: memberTypeStringPointer("QCanvas production")},
		},
		{
			name:  "human administrator can update ordinary metadata",
			user:  &User{ID: 42, Email: "executive@example.com", Notes: "executive", Role: RoleAdmin, Status: StatusActive},
			input: UpdateUserInput{Notes: memberTypeStringPointer("executive account")},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newService(tt.user)
			_, err := svc.UpdateUser(context.Background(), tt.user.ID, &tt.input)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, http.StatusForbidden, infraerrors.Code(err))
				require.Equal(t, tt.wantReason, infraerrors.Reason(err))
				require.Empty(t, repo.updated)
				return
			}
			require.NoError(t, err)
			require.Len(t, repo.updated, 1)
		})
	}
}

func TestLANAdminUserRPMLimitAlwaysRemainsUnlimited(t *testing.T) {
	settings := NewSettingService(nil, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})
	repo := &userRepoStub{nextID: 42}
	svc := &adminServiceImpl{userRepo: repo, settingService: settings}
	zeroBalance := 0.0

	created, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:      "qcanvas@wujie.local",
		MemberType: UserMemberTypeTool,
		Role:       RoleUser,
		Balance:    &zeroBalance,
		RPMLimit:   60,
	})
	require.NoError(t, err)
	require.Zero(t, created.RPMLimit)

	repo.user = &User{ID: 42, Email: "qcanvas@wujie.local", Notes: "[工具] QCanvas", Role: RoleUser, Status: StatusActive, RPMLimit: 10}
	newLimit := 120
	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{RPMLimit: &newLimit})
	require.NoError(t, err)
	require.Zero(t, updated.RPMLimit)
	require.Zero(t, repo.updated[0].RPMLimit)
}

func memberTypeStringPointer(value string) *string { return &value }
