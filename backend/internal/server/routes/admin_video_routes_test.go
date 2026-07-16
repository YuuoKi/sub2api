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
		http.MethodPost + " /api/v1/admin/video/providers/:id/tiny-real-authorization": false,
		http.MethodGet + " /api/v1/admin/video/tasks":                                  false,
		http.MethodGet + " /api/v1/admin/video/tasks/:id":                              false,
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
