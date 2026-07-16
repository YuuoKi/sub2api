//go:build unit

package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAuthServiceRefreshTokenPairPreservesTemporaryCredentialRestriction(t *testing.T) {
	ctx := context.Background()
	user := &service.User{
		ID:                 82,
		Email:              "temporary-refresh@example.com",
		Role:               service.RoleUser,
		Status:             service.StatusActive,
		TokenVersion:       3,
		MustChangePassword: true,
	}
	userRepo := newEmailBindUserRepoStub(user)
	refreshTokenCache := newEmailBindRefreshTokenCacheStub()
	cfg := &config.Config{JWT: config.JWTConfig{
		Secret:                   "temporary-refresh-test-secret",
		AccessTokenExpireMinutes: 60,
		RefreshTokenExpireDays:   7,
	}}
	authService := service.NewAuthService(
		nil,
		userRepo,
		nil,
		refreshTokenCache,
		cfg,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	initialPair, err := authService.GenerateTokenPair(ctx, user, "")
	require.NoError(t, err)

	refreshedPair, err := authService.RefreshTokenPair(ctx, initialPair.RefreshToken)
	require.NoError(t, err)
	require.True(t, refreshedPair.MustChangePassword)
	require.NotEqual(t, initialPair.RefreshToken, refreshedPair.RefreshToken)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.HandlerFunc(servermiddleware.NewJWTAuthMiddleware(
		authService,
		service.NewUserService(userRepo, nil, nil, nil),
	)))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/auth/me", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.PUT("/api/v1/user/password", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/api/v1/auth/revoke-all-sessions", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{method: http.MethodGet, path: "/protected", want: http.StatusForbidden},
		{method: http.MethodGet, path: "/api/v1/auth/me", want: http.StatusOK},
		{method: http.MethodPut, path: "/api/v1/user/password", want: http.StatusOK},
		{method: http.MethodPost, path: "/api/v1/auth/revoke-all-sessions", want: http.StatusForbidden},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, tc.path, nil)
		request.Header.Set("Authorization", "Bearer "+refreshedPair.AccessToken)
		router.ServeHTTP(recorder, request)
		require.Equal(t, tc.want, recorder.Code, tc.method+" "+tc.path)
	}

	_, err = authService.RefreshTokenPair(ctx, initialPair.RefreshToken)
	require.ErrorIs(t, err, service.ErrRefreshTokenInvalid, "refresh token rotation must keep the old token unusable")
}
