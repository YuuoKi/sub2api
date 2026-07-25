package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newBudgetAPIKeyRoutesRouter(auth middleware.AdminAuthMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{
		Dashboard: adminhandler.NewDashboardHandler(nil, nil, nil),
		APIKey:    adminhandler.NewAdminAPIKeyHandler(nil, nil),
	}}
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, auth, nil)
	return router
}

func TestBudgetAndAPIKeyAdminRoutesRegisteredExactly(t *testing.T) {
	router := newBudgetAPIKeyRoutesRouter(middleware.AdminAuthMiddleware(func(c *gin.Context) { c.Next() }))
	want := map[string]bool{
		http.MethodPut + " /api/v1/admin/dashboard/monthly-budget": false,
		http.MethodPost + " /api/v1/admin/users/:id/api-keys":      false,
		http.MethodGet + " /api/v1/admin/api-keys/:id/reveal":      false,
		http.MethodPut + " /api/v1/admin/api-keys/:id":             false,
		http.MethodDelete + " /api/v1/admin/api-keys/:id":          false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, registered := range want {
		require.True(t, registered, route)
	}
}

func TestBudgetAndAPIKeyAdminRoutesRequireAdminAuth(t *testing.T) {
	router := newBudgetAPIKeyRoutesRouter(middleware.AdminAuthMiddleware(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	}))
	tests := []struct{ method, path string }{
		{http.MethodPut, "/api/v1/admin/dashboard/monthly-budget"},
		{http.MethodPost, "/api/v1/admin/users/42/api-keys"},
		{http.MethodGet, "/api/v1/admin/api-keys/10/reveal"},
		{http.MethodPut, "/api/v1/admin/api-keys/10"},
		{http.MethodDelete, "/api/v1/admin/api-keys/10"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, nil)
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusUnauthorized, recorder.Code, "%s %s", test.method, test.path)
	}
}
