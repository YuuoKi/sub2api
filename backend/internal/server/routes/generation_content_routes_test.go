package routes

import (
	"context"
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

func newGenerationContentAdminRouter(auth middleware.AdminAuthMiddleware, repo *generationContentRoutesRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	content := adminhandler.NewGenerationContentHandler(repo, &config.Config{}, nil)
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{GenerationContent: content}}
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, auth, nil)
	return router
}

func TestGenerationContentAdminRoutesRegistered(t *testing.T) {
	repo := &generationContentRoutesRepo{}
	router := newGenerationContentAdminRouter(middleware.AdminAuthMiddleware(func(c *gin.Context) { c.Next() }), repo)
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
	}), repo)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/admin/generation-content/stats", nil))
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Zero(t, repo.statsCalls)
}
