//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestJWTAuthTemporaryCredentialAllowsOnlyIdentityPasswordAndLogoutPaths(t *testing.T) {
	user := &service.User{
		ID:                 81,
		Email:              "forced@example.com",
		Role:               service.RoleUser,
		Status:             service.StatusActive,
		Concurrency:        1,
		MustChangePassword: true,
	}
	router, authSvc := newJWTTestEnv(map[int64]*service.User{user.ID: user})
	router.GET("/api/v1/auth/me", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.PUT("/api/v1/user/password", func(c *gin.Context) { c.Status(http.StatusOK) })
	token, err := authSvc.GenerateToken(user)
	require.NoError(t, err)

	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/protected", http.StatusForbidden},
		{http.MethodGet, "/api/v1/auth/me", http.StatusOK},
		{http.MethodPut, "/api/v1/user/password", http.StatusOK},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)
		require.Equal(t, tc.want, w.Code, tc.method+" "+tc.path)
	}
}
