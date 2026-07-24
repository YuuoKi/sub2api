package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type pricingVideoAdminRepo struct {
	task      service.VideoTask
	deleteErr error
}

func (r *pricingVideoAdminRepo) ListVideoProviders(context.Context) ([]service.VideoProviderAccount, error) {
	return nil, nil
}
func (r *pricingVideoAdminRepo) CreateVideoProvider(context.Context, service.VideoProviderAccount) (*service.VideoProviderAccount, error) {
	return nil, nil
}
func (r *pricingVideoAdminRepo) UpdateVideoProvider(context.Context, int64, service.VideoProviderAdminUpdate) (*service.VideoProviderAccount, error) {
	return nil, nil
}
func (r *pricingVideoAdminRepo) DeleteVideoProvider(context.Context, int64) error { return r.deleteErr }
func (r *pricingVideoAdminRepo) AuthorizeTinyReal(context.Context, int64, int64) (*service.VideoProviderAccount, error) {
	return nil, nil
}
func (r *pricingVideoAdminRepo) ListVideoTasks(context.Context, service.VideoAdminTaskFilter) ([]service.VideoTask, int64, error) {
	return []service.VideoTask{r.task}, 1, nil
}
func (r *pricingVideoAdminRepo) GetVideoTaskAdmin(context.Context, int64) (*service.VideoTask, error) {
	return &r.task, nil
}
func (r *pricingVideoAdminRepo) VideoSystemCheck(context.Context) (service.VideoSystemCheck, error) {
	return service.VideoSystemCheck{}, nil
}

func TestVideoAdminListAndDetailExposePricingProvenance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	price, rate, maximum := 2.0, 7.0, 1.4
	repo := &pricingVideoAdminRepo{task: service.VideoTask{
		ID: 41, Currency: "USD", PricingSource: service.VideoPricingSourceConfig,
		PricingVersion:                       service.VideoPricingVersionSeedanceCompletionTokensUSDV1,
		PricingCNYPerMillionCompletionTokens: &price, PricingUSDCNYExchangeRate: &rate, PricingMaximumCNY: &maximum,
	}}
	handler := &VideoHandler{service: service.NewVideoAdminService(repo, nil)}
	router := gin.New()
	router.GET("/tasks", handler.ListTasks)
	router.GET("/tasks/:id", handler.GetTask)

	for _, path := range []string{"/tasks", "/tasks/41"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, recorder.Code, path)
		require.Contains(t, recorder.Body.String(), `"currency":"USD"`, path)
		require.Contains(t, recorder.Body.String(), `"pricing_source":"config.video_gateway"`, path)
		require.Contains(t, recorder.Body.String(), `"pricing_version":"seedance_completion_tokens_usd_v1"`, path)
	}
}

func TestVideoAdminDeleteProviderResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &pricingVideoAdminRepo{}
	handler := &VideoHandler{service: service.NewVideoAdminService(repo, nil)}
	router := gin.New()
	router.DELETE("/providers/:id", handler.DeleteProvider)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/providers/7", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Video provider deleted successfully")

	repo.deleteErr = service.ErrVideoProviderNotFound
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/providers/99", nil))
	require.Equal(t, http.StatusNotFound, recorder.Code)

	repo.deleteErr = fmt.Errorf("%w: 该通道仍被视频任务引用，不能删除", service.ErrVideoAdminConflict)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/providers/7", nil))
	require.Equal(t, http.StatusConflict, recorder.Code)

	repo.deleteErr = nil
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/providers/0", nil))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestVideoAdminContractExposesPlatformRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &VideoHandler{}
	router := gin.New()
	router.GET("/contract", handler.Contract)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/contract", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, `"provider":"seedance"`)
	require.Contains(t, body, `"platforms"`)
	require.Contains(t, body, `"provider":"jimeng"`)
	require.Contains(t, body, `"adapter_ready":true`)
	require.Contains(t, body, `"adapter_ready":false`)
}

func TestVideoTaskAdminResponseIncludesPersistedSpecificationEvidence(t *testing.T) {
	tokens := int64(321)
	price, rate, maximum := 2.0, 7.0, 1.4
	reservedAt := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)
	response := videoTaskAdminResponse(&service.VideoTask{
		ID:                                   41,
		APIKeyID:                             12,
		GroupID:                              13,
		Model:                                service.SeedanceModel,
		DurationSeconds:                      4,
		Resolution:                           "720p",
		UsageTotalTokens:                     &tokens,
		ReservedCostUSD:                      0.2,
		ReservationState:                     service.VideoReservationCaptured,
		ReservedAt:                           &reservedAt,
		CostAmount:                           0.1,
		ProviderActualCostUSD:                0.08,
		Currency:                             "USD",
		PricingSource:                        service.VideoPricingSourceConfig,
		PricingVersion:                       service.VideoPricingVersionSeedanceCompletionTokensUSDV1,
		PricingCNYPerMillionCompletionTokens: &price,
		PricingUSDCNYExchangeRate:            &rate,
		PricingMaximumCNY:                    &maximum,
	})

	require.Equal(t, int64(12), response["api_key_id"])
	require.Equal(t, int64(13), response["group_id"])
	require.Equal(t, 4, response["duration_seconds"])
	require.Equal(t, "720p", response["resolution"])
	require.Equal(t, &tokens, response["usage_total_tokens"])
	require.Equal(t, 0.2, response["reserved_cost_usd"])
	require.Equal(t, service.VideoReservationCaptured, response["reservation_state"])
	require.Equal(t, &reservedAt, response["reserved_at"])
	require.Equal(t, 0.1, response["cost_amount"])
	require.Equal(t, 0.08, response["provider_actual_cost_usd"])
	require.Equal(t, "USD", response["currency"])
	require.Equal(t, service.VideoPricingSourceConfig, response["pricing_source"])
	require.Equal(t, service.VideoPricingVersionSeedanceCompletionTokensUSDV1, response["pricing_version"])
	require.Equal(t, &price, response["pricing_cny_per_million_completion_tokens"])
	require.Equal(t, &rate, response["pricing_usd_cny_exchange_rate"])
	require.Equal(t, &maximum, response["pricing_maximum_cny"])
	require.Equal(t, service.SeedanceModel, response["request_model"])
	require.Nil(t, response["upstream_model"])
	require.Nil(t, response["billing_model"])
	require.Nil(t, response["balance_before_usd"])
	require.Nil(t, response["balance_after_usd"])
	require.Nil(t, response["authorization_consumed_at"])
}

func TestVideoTaskAdminResponseExposesLocalAssetContractWithoutPathLeak(t *testing.T) {
	savedAt := time.Now().UTC()
	response := videoTaskAdminResponse(&service.VideoTask{ID: 51, Status: service.VideoStatusSucceeded, ResultURL: "https://assets.example.test/result.mp4", LocalAssetPath: "assets/video/51/result.mp4", LocalAssetSavedAt: &savedAt})
	require.Equal(t, true, response["local_asset_available"])
	require.Equal(t, "/api/v1/admin/video/tasks/51/local-asset", response["local_asset_download_url"])
	require.Equal(t, &savedAt, response["local_asset_saved_at"])
	require.NotContains(t, response, "local_asset_path")
}
