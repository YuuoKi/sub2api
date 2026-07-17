package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type generationContentRoutesRepo struct{ statsCalls int }

type generationContentRoutesSettingRepo struct{ acknowledged bool }

func (r *generationContentRoutesSettingRepo) Get(ctx context.Context, key string) (*service.Setting, error) {
	value, err := r.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &service.Setting{Key: key, Value: value}, nil
}
func (r *generationContentRoutesSettingRepo) GetValue(context.Context, string) (string, error) {
	if !r.acknowledged {
		return "", service.ErrSettingNotFound
	}
	payload, _ := json.Marshal(service.AdminComplianceAcknowledgement{Version: service.AdminComplianceVersion, AdminUserID: 1})
	return string(payload), nil
}
func (*generationContentRoutesSettingRepo) Set(context.Context, string, string) error { return nil }
func (*generationContentRoutesSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (*generationContentRoutesSettingRepo) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (*generationContentRoutesSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (*generationContentRoutesSettingRepo) Delete(context.Context, string) error { return nil }

func (r *generationContentRoutesRepo) Create(context.Context, *service.GenerationContent) error {
	return nil
}
func (r *generationContentRoutesRepo) GetCaptureStats(context.Context) (*service.GenerationContentStats, error) {
	r.statsCalls++
	return &service.GenerationContentStats{}, nil
}
func (r *generationContentRoutesRepo) GetRecent(context.Context, int) ([]service.GenerationContentSample, error) {
	return nil, nil
}
func (r *generationContentRoutesRepo) UpdateTaskAdoption(context.Context, service.GenerationContentAdoptionInput) (*service.GenerationContentAdoption, error) {
	return &service.GenerationContentAdoption{}, nil
}
func (r *generationContentRoutesRepo) GetWeeklyReport(_ context.Context, start, end time.Time) (*service.GenerationContentWeeklyReport, error) {
	return &service.GenerationContentWeeklyReport{PeriodStart: start, PeriodEnd: end}, nil
}

func newGenerationContentAdminRouter(auth middleware.AdminAuthMiddleware, repo *generationContentRoutesRepo, settings *service.SettingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	content := adminhandler.NewGenerationContentHandler(repo, &config.Config{}, nil)
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{GenerationContent: content}}
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, auth, settings)
	return router
}

func generationContentTestAdminAuth(c *gin.Context) {
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})
	c.Next()
}

func TestGenerationContentAdminRoutesRegistered(t *testing.T) {
	repo := &generationContentRoutesRepo{}
	settings := service.NewSettingService(&generationContentRoutesSettingRepo{acknowledged: true}, &config.Config{})
	router := newGenerationContentAdminRouter(middleware.AdminAuthMiddleware(generationContentTestAdminAuth), repo, settings)
	tests := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/admin/generation-content/stats", ""},
		{http.MethodGet, "/api/v1/admin/generation-content/samples", ""},
		{http.MethodGet, "/api/v1/admin/generation-content/weekly-report", ""},
		{http.MethodPost, "/api/v1/admin/generation-content/42/adoption", `{"adoption_status":"pending"}`},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		require.NotEqual(t, http.StatusNotFound, recorder.Code, "%s %s", test.method, test.path)
	}
}

func TestGenerationContentAdminRoutesRequireAdminAuth(t *testing.T) {
	repo := &generationContentRoutesRepo{}
	router := newGenerationContentAdminRouter(middleware.AdminAuthMiddleware(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	}), repo, service.NewSettingService(&generationContentRoutesSettingRepo{acknowledged: true}, &config.Config{}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/admin/generation-content/stats", nil))
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Zero(t, repo.statsCalls)
}

func TestGenerationContentAdminRoutesRequireComplianceAcknowledgement(t *testing.T) {
	repo := &generationContentRoutesRepo{}
	settings := service.NewSettingService(&generationContentRoutesSettingRepo{}, &config.Config{})
	router := newGenerationContentAdminRouter(middleware.AdminAuthMiddleware(generationContentTestAdminAuth), repo, settings)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/admin/generation-content/stats", nil))
	require.Equal(t, http.StatusLocked, recorder.Code)
	require.Contains(t, recorder.Body.String(), "ADMIN_COMPLIANCE_ACK_REQUIRED")
	require.Zero(t, repo.statsCalls)
}
