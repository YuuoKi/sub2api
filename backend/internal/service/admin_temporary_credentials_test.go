//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestAdminServiceCreateUserIssuesOneTimeTemporaryCredential(t *testing.T) {
	repo := &userRepoStub{nextID: 71}
	svc := &adminServiceImpl{userRepo: repo}

	result, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:       "temporary@example.com",
		Concurrency: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.InitialCredential)
	require.NotEmpty(t, result.InitialCredential.TemporaryPassword)
	require.WithinDuration(t, time.Now().UTC().Add(24*time.Hour), result.InitialCredential.ExpiresAt, 5*time.Second)
	require.True(t, result.MustChangePassword)
	require.NotNil(t, result.TemporaryPasswordExpiresAt)
	require.True(t, result.CheckPassword(result.InitialCredential.TemporaryPassword))
	require.NotContains(t, result.InitialCredential.TemporaryPassword, result.PasswordHash)
}

func TestGenerateTemporaryPasswordUsesUniqueCryptographicValues(t *testing.T) {
	seen := make(map[string]struct{}, 64)
	for range 64 {
		password, err := generateTemporaryPassword()
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(password), 32)
		_, duplicate := seen[password]
		require.False(t, duplicate)
		seen[password] = struct{}{}
	}
}

func TestAdminServiceResetUserPasswordReplacesCredentialAndRevokesOldPassword(t *testing.T) {
	user := &User{
		ID:          72,
		Email:       "reset@example.com",
		Role:        RoleUser,
		Status:      StatusActive,
		Concurrency: 1,
	}
	require.NoError(t, user.SetPassword("old-password"))
	repo := &userRepoStub{user: user}
	svc := &adminServiceImpl{userRepo: repo}

	credential, err := svc.ResetUserPassword(context.Background(), user.ID)

	require.NoError(t, err)
	require.NotEmpty(t, credential.TemporaryPassword)
	require.False(t, user.CheckPassword("old-password"))
	require.True(t, user.CheckPassword(credential.TemporaryPassword))
	require.True(t, user.MustChangePassword)
	require.NotNil(t, user.TemporaryPasswordExpiresAt)
}

func TestAuthServiceRejectsExpiredTemporaryPassword(t *testing.T) {
	expiredAt := time.Now().UTC().Add(-time.Minute)
	user := &User{
		ID:                         73,
		Email:                      "expired@example.com",
		Role:                       RoleUser,
		Status:                     StatusActive,
		MustChangePassword:         true,
		TemporaryPasswordExpiresAt: &expiredAt,
	}
	require.NoError(t, user.SetPassword("expired-temporary-password"))
	repo := &userRepoStub{user: user, usersByEmail: map[string]*User{user.Email: user}}
	cfg := &config.Config{}
	cfg.JWT.Secret = "temporary-credential-test-secret"
	cfg.JWT.ExpireHour = 1
	svc := NewAuthService(nil, repo, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)

	_, _, err := svc.Login(context.Background(), user.Email, "expired-temporary-password")

	require.ErrorIs(t, err, ErrTemporaryPasswordExpired)
}

func TestUserServiceChangePasswordClearsTemporaryCredentialRequirement(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	user := &User{
		ID:                         74,
		Email:                      "change@example.com",
		Role:                       RoleUser,
		Status:                     StatusActive,
		MustChangePassword:         true,
		TemporaryPasswordExpiresAt: &expiresAt,
	}
	require.NoError(t, user.SetPassword("temporary-password"))
	repo := &userRepoStub{user: user}
	svc := NewUserService(repo, nil, nil, nil)

	err := svc.ChangePassword(context.Background(), user.ID, ChangePasswordRequest{
		CurrentPassword: "temporary-password",
		NewPassword:     "permanent-password",
	})

	require.NoError(t, err)
	require.False(t, user.MustChangePassword)
	require.Nil(t, user.TemporaryPasswordExpiresAt)
	require.True(t, user.CheckPassword("permanent-password"))
}
