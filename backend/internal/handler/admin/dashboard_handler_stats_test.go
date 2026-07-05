package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dashboardStatsRepoStub struct {
	service.UsageLogRepository
	stats *usagestats.DashboardStats
}

func (r *dashboardStatsRepoStub) GetDashboardStats(context.Context) (*usagestats.DashboardStats, error) {
	return r.stats, nil
}

type dashboardSettingRepoStub struct {
	value string
}

func (r *dashboardSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *dashboardSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return r.value, nil
}

func (r *dashboardSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (r *dashboardSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (r *dashboardSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (r *dashboardSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (r *dashboardSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestDashboardHandlerGetStatsIncludesUSDCNYRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dashboardSvc := service.NewDashboardService(&dashboardStatsRepoStub{stats: &usagestats.DashboardStats{
		TotalUsers:   3,
		CostCurrency: "USD",
	}}, nil, nil, nil)
	settingSvc := service.NewSettingService(&dashboardSettingRepoStub{value: "7.35"}, nil)
	handler := NewDashboardHandler(dashboardSvc, nil, settingSvc)
	router := gin.New()
	router.GET("/admin/dashboard/stats", handler.GetStats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Code int `json:"code"`
		Data struct {
			USDCNYRate float64 `json:"usd_cny_rate"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 7.35, body.Data.USDCNYRate)
}
