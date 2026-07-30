package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/buildinfo"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterSystemRoutesKeepsVersionOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	h := &handler.Handlers{
		Admin: &handler.AdminHandlers{
			System: adminhandler.NewSystemHandler(buildinfo.New("0.1.151", "abc", "2026-07-25", "source"), nil, nil),
		},
	}
	registerSystemRoutes(admin, h)

	mounted := map[string]bool{}
	for _, route := range router.Routes() {
		mounted[route.Method+" "+route.Path] = true
	}

	require.True(t, mounted[http.MethodGet+" /api/v1/admin/system/version"])
	require.False(t, mounted[http.MethodGet+" /api/v1/admin/system/check-updates"])
	require.False(t, mounted[http.MethodGet+" /api/v1/admin/system/rollback-versions"])
	require.False(t, mounted[http.MethodPost+" /api/v1/admin/system/update"])
	require.False(t, mounted[http.MethodPost+" /api/v1/admin/system/rollback"])
}

func TestAdminSystemVersionReturnsBuildIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	info := buildinfo.New("0.1.151", "deadbeefcafebabe", "2026-07-25T01:02:03Z", "source")
	h := &handler.Handlers{
		Admin: &handler.AdminHandlers{
			System: adminhandler.NewSystemHandler(info, nil, nil),
		},
	}
	registerSystemRoutes(admin, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/system/version", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "广州内部版 2026.07.25-r151")
	require.Contains(t, body, "deadbeefcafebabe")
	require.Contains(t, body, "2026-07-25T01:02:03Z")
	require.NotContains(t, body, "vdeadbeefcafebabe")
}
