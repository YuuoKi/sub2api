package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type realtimeDashboardRepoStub struct {
	service.UsageLogRepository
	window *usagestats.DashboardStats
	health *usagestats.DashboardStats
}

func (r *realtimeDashboardRepoStub) GetDashboardStats(context.Context) (*usagestats.DashboardStats, error) {
	return r.health, nil
}

func (r *realtimeDashboardRepoStub) GetDashboardStatsWithRange(context.Context, time.Time, time.Time) (*usagestats.DashboardStats, error) {
	return r.window, nil
}

type realtimeActiveCounterStub struct {
	n int64
}

func (s *realtimeActiveCounterStub) CountActiveRequests(context.Context) (int64, error) {
	return s.n, nil
}

func TestDashboardHandlerGetRealtimeMetricsUsesService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &realtimeDashboardRepoStub{
		window: &usagestats.DashboardStats{TotalRequests: 9, AverageDurationMs: 150},
		health: &usagestats.DashboardStats{TotalAccounts: 4, ErrorAccounts: 1},
	}
	dashboardSvc := service.NewDashboardService(repo, nil, nil, nil)
	dashboardSvc.SetActiveRequestCounter(&realtimeActiveCounterStub{n: 5})
	h := NewDashboardHandler(dashboardSvc, nil, nil)

	router := gin.New()
	router.GET("/admin/dashboard/realtime", h.GetRealtimeMetrics)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/realtime", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.EqualValues(t, 5, body.Data["active_requests"])
	require.EqualValues(t, 9, body.Data["requests_per_minute"])
	require.EqualValues(t, 150, body.Data["average_response_time"])
	require.EqualValues(t, 25, body.Data["error_rate"])
}
