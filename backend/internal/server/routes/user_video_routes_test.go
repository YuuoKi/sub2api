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

func TestUserVideoLocalAssetRouteIsRegisteredBehindJWTMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	jwtCalled := false
	jwt := middleware.JWTAuthMiddleware(func(c *gin.Context) {
		jwtCalled = true
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	h := &handler.Handlers{VideoGateway: &handler.VideoGatewayHandler{}}
	RegisterUserRoutes(router.Group("/api/v1"), h, jwt, nil)
	found := false
	for _, route := range router.Routes() {
		if route.Method == http.MethodGet && route.Path == "/api/v1/video/tasks/:id/local-asset" {
			found = true
		}
	}
	require.True(t, found)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/video/tasks/42/local-asset", nil))
	require.True(t, jwtCalled)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}
