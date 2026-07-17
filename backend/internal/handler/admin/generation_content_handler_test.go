package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type generationContentHandlerRepo struct {
	stats         *service.GenerationContentStats
	recent        []service.GenerationContentSample
	report        *service.GenerationContentWeeklyReport
	adoptionInput service.GenerationContentAdoptionInput
}

func (r *generationContentHandlerRepo) Create(context.Context, *service.GenerationContent) error {
	return nil
}
func (r *generationContentHandlerRepo) GetCaptureStats(context.Context) (*service.GenerationContentStats, error) {
	return r.stats, nil
}
func (r *generationContentHandlerRepo) GetRecent(context.Context, int) ([]service.GenerationContentSample, error) {
	return r.recent, nil
}
func (r *generationContentHandlerRepo) UpdateTaskAdoption(_ context.Context, input service.GenerationContentAdoptionInput) (*service.GenerationContentAdoption, error) {
	r.adoptionInput = input
	return &service.GenerationContentAdoption{TaskID: input.TaskID, AdoptionStatus: input.AdoptionStatus, QualityScore: input.QualityScore, Notes: input.Notes, Saved: true}, nil
}
func (r *generationContentHandlerRepo) GetWeeklyReport(context.Context, time.Time, time.Time) (*service.GenerationContentWeeklyReport, error) {
	return r.report, nil
}

type generationRateSettingRepo struct{ value string }

func (r *generationRateSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (r *generationRateSettingRepo) GetValue(context.Context, string) (string, error) {
	return r.value, nil
}
func (r *generationRateSettingRepo) Set(context.Context, string, string) error { return nil }
func (r *generationRateSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (r *generationRateSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *generationRateSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return nil, nil
}
func (r *generationRateSettingRepo) Delete(context.Context, string) error { return nil }

func newGenerationContentHandlerRouter(repo *generationContentHandlerRepo, cfg *config.Config, settings *service.SettingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewGenerationContentHandler(repo, cfg, settings)
	router.GET("/stats", h.GetStats)
	router.GET("/samples", h.GetSamples)
	router.GET("/weekly", h.GetWeeklyReport)
	router.POST("/adoption/:task_id", h.UpdateAdoption)
	return router
}

func TestGenerationContentHandlerWeeklyPayloadUsesSettingRate(t *testing.T) {
	repo := &generationContentHandlerRepo{report: &service.GenerationContentWeeklyReport{
		PeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		Entries: 4, VideoTasks: 1, TotalCostEstimate: 1.25, AdoptedCount: 1, AdoptionRate: 0.25,
	}}
	settings := service.NewSettingService(&generationRateSettingRepo{value: "7.41"}, nil)
	router := newGenerationContentHandlerRouter(repo, &config.Config{}, settings)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/weekly", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"total_cost_estimate":1.25`)
	require.Contains(t, recorder.Body.String(), `"usd_cny_rate":7.41`)
	require.Contains(t, recorder.Body.String(), "Weekly Production Ledger")
}

func TestGenerationContentHandlerSamplesExposePricingSource(t *testing.T) {
	repo := &generationContentHandlerRepo{recent: []service.GenerationContentSample{{
		Model: "claude", CreatedAt: time.Now().UTC(), CostEstimate: 0.5, Currency: "USD", PricingSource: "usage_logs.actual_cost",
	}}}
	router := newGenerationContentHandlerRouter(repo, &config.Config{}, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/samples", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"pricing_source":"usage_logs.actual_cost"`)
}

func TestGenerationContentHandlerAdoptionValidatesAndNormalizes(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{name: "invalid task", path: "/adoption/0", body: `{"adoption_status":"adopted"}`, wantStatus: http.StatusBadRequest},
		{name: "invalid status", path: "/adoption/42", body: `{"adoption_status":"done"}`, wantStatus: http.StatusBadRequest},
		{name: "invalid score", path: "/adoption/42", body: `{"adoption_status":"pending","quality_score":1.1}`, wantStatus: http.StatusBadRequest},
		{name: "valid", path: "/adoption/42", body: `{"adoption_status":" ADOPTED ","quality_score":0.8,"notes":" used "}`, wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &generationContentHandlerRepo{}
			router := newGenerationContentHandlerRouter(repo, &config.Config{Gateway: config.GatewayConfig{ContentCapture: config.ContentCaptureConfig{Enabled: true}}}, nil)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, test.path, bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			require.Equal(t, test.wantStatus, recorder.Code, recorder.Body.String())
			if test.name == "valid" {
				require.Equal(t, int64(42), repo.adoptionInput.TaskID)
				require.Equal(t, "adopted", repo.adoptionInput.AdoptionStatus)
				require.Equal(t, "used", repo.adoptionInput.Notes)
			}
		})
	}
}
