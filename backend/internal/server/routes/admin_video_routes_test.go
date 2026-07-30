package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminVideoRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	h := &handler.Handlers{Admin: &handler.AdminHandlers{Video: &adminhandler.VideoHandler{}}}
	registerAdminVideoRoutes(v1.Group("/admin"), h)
	routes := router.Routes()
	want := map[string]bool{
		http.MethodGet + " /api/v1/admin/video/contract":                               false,
		http.MethodGet + " /api/v1/admin/video/providers":                              false,
		http.MethodPost + " /api/v1/admin/video/providers":                             false,
		http.MethodPut + " /api/v1/admin/video/providers/:id":                          false,
		http.MethodDelete + " /api/v1/admin/video/providers/:id":                       false,
		http.MethodPost + " /api/v1/admin/video/providers/:id/connectivity-check":      false,
		http.MethodPost + " /api/v1/admin/video/providers/:id/tiny-real-authorization": false,
		http.MethodGet + " /api/v1/admin/video/tasks":                                  false,
		http.MethodGet + " /api/v1/admin/video/tasks/:id":                              false,
		http.MethodGet + " /api/v1/admin/video/tasks/:id/local-asset":                  false,
		http.MethodPost + " /api/v1/admin/video/tasks/:id/asset-handoffs":              false,
		http.MethodGet + " /api/v1/admin/video/system-check":                           false,
	}
	for _, route := range routes {
		if _, ok := want[route.Method+" "+route.Path]; ok {
			want[route.Method+" "+route.Path] = true
		}
	}
	for route, found := range want {
		require.True(t, found, route)
	}
}

func TestAdminVideoLocalAssetRouteRejectsEmployeeBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserRole), c.GetHeader("X-Test-Role"))
		c.Next()
	}, middleware.AdminOnly())
	h := &handler.Handlers{Admin: &handler.AdminHandlers{Video: &adminhandler.VideoHandler{}}}
	registerAdminVideoRoutes(admin, h)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/video/tasks/501/local-asset", nil)
	req.Header.Set("X-Test-Role", service.RoleUser)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestPublicAssetHandoffConsumeRouteIsRegisteredWithoutTrustingForwardedIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	h := &handler.Handlers{Admin: &handler.AdminHandlers{Video: &adminhandler.VideoHandler{}}}
	RegisterAssetHandoffRoutes(v1, h)

	found := false
	for _, route := range router.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/public/asset-handoffs/consume" {
			found = true
		}
	}
	require.True(t, found)
}

func TestAdminAssetHandoffRouteRejectsEmployeeBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserRole), c.GetHeader("X-Test-Role"))
		c.Next()
	}, middleware.AdminOnly())
	h := &handler.Handlers{Admin: &handler.AdminHandlers{Video: &adminhandler.VideoHandler{}}}
	registerAdminVideoRoutes(admin, h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/video/tasks/501/asset-handoffs", nil)
	req.Header.Set("X-Test-Role", service.RoleUser)
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, req)
	require.Equal(t, http.StatusForbidden, responseRecorder.Code)
}

func TestAdminVideoContractRejectsEmployeeAndAllowsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUserRole), c.GetHeader("X-Test-Role"))
		c.Next()
	}, middleware.AdminOnly())
	h := &handler.Handlers{Admin: &handler.AdminHandlers{Video: &adminhandler.VideoHandler{}}}
	registerAdminVideoRoutes(admin, h)
	for _, test := range []struct {
		role   string
		status int
	}{{role: service.RoleUser, status: http.StatusForbidden}, {role: service.RoleAdmin, status: http.StatusOK}} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/video/contract", nil)
		req.Header.Set("X-Test-Role", test.role)
		responseRecorder := httptest.NewRecorder()
		router.ServeHTTP(responseRecorder, req)
		require.Equal(t, test.status, responseRecorder.Code)
	}
}
