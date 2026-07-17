package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserSimulationVideoCreateRouteRegisteredBehindJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	jwtCalled := false
	jwt := middleware.JWTAuthMiddleware(func(c *gin.Context) {
		jwtCalled = true
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	h := &handler.Handlers{VideoGateway: &handler.VideoGatewayHandler{}}
	RegisterUserRoutes(router.Group("/api/v1"), h, jwt, nil)

	wantPaths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/user/video/simulation/contract"},
		{http.MethodPost, "/api/v1/user/video/simulation/tasks"},
		{http.MethodGet, "/api/v1/user/video/simulation/tasks"},
		{http.MethodGet, "/api/v1/user/video/simulation/tasks/:id"},
		{http.MethodPost, "/api/v1/user/video/simulation/tasks/:id/cancel"},
		{http.MethodGet, "/api/v1/user/video/simulation/tasks/:id/result"},
	}
	found := map[string]bool{}
	for _, route := range router.Routes() {
		found[route.Method+" "+route.Path] = true
	}
	for _, want := range wantPaths {
		require.True(t, found[want.method+" "+want.path], "missing route %s %s", want.method, want.path)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/user/video/simulation/tasks", nil))
	require.True(t, jwtCalled)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}
