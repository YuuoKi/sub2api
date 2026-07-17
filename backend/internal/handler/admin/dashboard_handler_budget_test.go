package admin

import (
	"bytes"
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

type budgetDashboardRepoStub struct {
	service.UsageLogRepository
	stats   *usagestats.DashboardStats
	ranking *usagestats.UserSpendingRankingResponse
}

func (r *budgetDashboardRepoStub) GetDashboardStats(context.Context) (*usagestats.DashboardStats, error) {
	return r.stats, nil
}
func (r *budgetDashboardRepoStub) GetUserSpendingRanking(context.Context, time.Time, time.Time, int) (*usagestats.UserSpendingRankingResponse, error) {
	return r.ranking, nil
}

type budgetDashboardSettingRepoStub struct{ values map[string]string }

func (*budgetDashboardSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}
func (r *budgetDashboardSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}
func (r *budgetDashboardSettingRepoStub) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}
func (*budgetDashboardSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}
func (*budgetDashboardSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}
func (*budgetDashboardSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}
func (r *budgetDashboardSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func newBudgetDashboardRouter(repo *budgetDashboardRepoStub, settingRepo *budgetDashboardSettingRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	dashboardSvc := service.NewDashboardService(repo, nil, nil, nil)
	settingSvc := service.NewSettingService(settingRepo, nil)
	h := NewDashboardHandler(dashboardSvc, nil, settingSvc)
	router := gin.New()
	router.GET("/admin/dashboard/stats", h.GetStats)
	router.GET("/admin/dashboard/snapshot-v2", h.GetSnapshotV2)
	router.PUT("/admin/dashboard/monthly-budget", h.UpdateMonthlyBudget)
	return router
}

func TestDashboardBudgetPayloadAndUpdate(t *testing.T) {
	repo := &budgetDashboardRepoStub{
		stats:   &usagestats.DashboardStats{TotalUsers: 1},
		ranking: &usagestats.UserSpendingRankingResponse{TotalActualCost: 100},
	}
	settings := &budgetDashboardSettingRepoStub{values: map[string]string{
		service.SettingKeyUSDCNYRate:              "7.2",
		service.SettingKeyCompanyMonthlyBudgetCNY: "1000",
	}}
	router := newBudgetDashboardRouter(repo, settings)

	for _, path := range []string{"/admin/dashboard/stats"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, rec.Code, path)
		require.Contains(t, rec.Body.String(), `"monthly_budget_cny":1000`, path)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/dashboard/monthly-budget", bytes.NewBufferString(`{"monthly_budget_cny":2500.5}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "2500.5", settings.values[service.SettingKeyCompanyMonthlyBudgetCNY])
	require.Contains(t, rec.Body.String(), `"monthly_budget_cny":2500.5`)
	require.Contains(t, rec.Body.String(), `"monthly_spend_cny":720`)
}

func TestDashboardBudgetUpdateRejectsInvalidJSONAndValues(t *testing.T) {
	repo := &budgetDashboardRepoStub{
		stats:   &usagestats.DashboardStats{},
		ranking: &usagestats.UserSpendingRankingResponse{},
	}
	tests := []string{
		`{}`,
		`{"monthly_budget_cny":-1}`,
		`{"monthly_budget_cny":1e309}`,
		`{"monthly_budget_cny":10} {}`,
		`{"monthly_budget_cny":`,
	}
	for _, body := range tests {
		settings := &budgetDashboardSettingRepoStub{values: map[string]string{service.SettingKeyCompanyMonthlyBudgetCNY: "1000"}}
		router := newBudgetDashboardRouter(repo, settings)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/admin/dashboard/monthly-budget", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, body)
		require.Equal(t, "1000", settings.values[service.SettingKeyCompanyMonthlyBudgetCNY], body)
	}
}

func TestDashboardBudgetUpdateZeroClears(t *testing.T) {
	repo := &budgetDashboardRepoStub{stats: &usagestats.DashboardStats{}, ranking: &usagestats.UserSpendingRankingResponse{}}
	settings := &budgetDashboardSettingRepoStub{values: map[string]string{service.SettingKeyCompanyMonthlyBudgetCNY: "1000"}}
	router := newBudgetDashboardRouter(repo, settings)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/dashboard/monthly-budget", bytes.NewBufferString(`{"monthly_budget_cny":0}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	_, exists := settings.values[service.SettingKeyCompanyMonthlyBudgetCNY]
	require.False(t, exists)

	var responseBody map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responseBody))
}
