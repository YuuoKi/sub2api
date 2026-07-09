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

type dashboardStatsRepoStub struct {
	service.UsageLogRepository
	stats   *usagestats.DashboardStats
	ranking *usagestats.UserSpendingRankingResponse
}

func (r *dashboardStatsRepoStub) GetDashboardStats(context.Context) (*usagestats.DashboardStats, error) {
	return r.stats, nil
}

func (r *dashboardStatsRepoStub) GetUserSpendingRanking(context.Context, time.Time, time.Time, int) (*usagestats.UserSpendingRankingResponse, error) {
	if r.ranking != nil {
		return r.ranking, nil
	}
	return &usagestats.UserSpendingRankingResponse{}, nil
}

type dashboardSettingRepoStub struct {
	value  string // legacy: usd_cny_rate only (other packages' tests)
	values map[string]string
}

func (r *dashboardSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (r *dashboardSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if r.values != nil {
		if v, ok := r.values[key]; ok {
			return v, nil
		}
	}
	if key == service.SettingKeyUSDCNYRate && r.value != "" {
		return r.value, nil
	}
	return "", nil
}

func (r *dashboardSettingRepoStub) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
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
	settingSvc := service.NewSettingService(&dashboardSettingRepoStub{values: map[string]string{
		service.SettingKeyUSDCNYRate: "7.35",
	}}, nil)
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

func TestDashboardHandlerGetStatsIncludesMonthlyBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dashboardSvc := service.NewDashboardService(&dashboardStatsRepoStub{
		stats: &usagestats.DashboardStats{TotalUsers: 1},
		ranking: &usagestats.UserSpendingRankingResponse{
			TotalCombinedActualCost: 100, // USD
		},
	}, nil, nil, nil)
	settingSvc := service.NewSettingService(&dashboardSettingRepoStub{values: map[string]string{
		service.SettingKeyUSDCNYRate:              "7.2",
		service.SettingKeyCompanyMonthlyBudgetCNY: "1000",
	}}, nil)
	handler := NewDashboardHandler(dashboardSvc, nil, settingSvc)
	router := gin.New()
	router.GET("/admin/dashboard/stats", handler.GetStats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data struct {
			MonthlyBudgetCNY           float64 `json:"monthly_budget_cny"`
			MonthlySpendCNY            float64 `json:"monthly_spend_cny"`
			MonthlyBudgetUsagePercent  float64 `json:"monthly_budget_usage_percent"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 1000.0, body.Data.MonthlyBudgetCNY)
	require.InDelta(t, 720.0, body.Data.MonthlySpendCNY, 0.01)
	require.InDelta(t, 72.0, body.Data.MonthlyBudgetUsagePercent, 0.01)
}

