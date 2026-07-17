package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestVideoTaskResponseExposesPricingProvenanceAtTopLevel(t *testing.T) {
	price, rate, maximum := 2.0, 7.0, 1.4
	payload := videoTaskResponse(&service.VideoTask{
		Currency: "USD", PricingSource: service.VideoPricingSourceConfig,
		PricingVersion:                       service.VideoPricingVersionSeedanceCompletionTokensUSDV1,
		PricingCNYPerMillionCompletionTokens: &price, PricingUSDCNYExchangeRate: &rate, PricingMaximumCNY: &maximum,
	})
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(b, &decoded))
	require.Equal(t, "USD", decoded["currency"])
	require.Equal(t, "config.video_gateway", decoded["pricing_source"])
	require.Equal(t, "seedance_completion_tokens_usd_v1", decoded["pricing_version"])
	require.Equal(t, 2.0, decoded["pricing_cny_per_million_completion_tokens"])
	require.Equal(t, 7.0, decoded["pricing_usd_cny_exchange_rate"])
	require.Equal(t, 1.4, decoded["pricing_maximum_cny"])
}

func TestVideoTaskResponseKeepsUnknownHistoricalPricingNullable(t *testing.T) {
	payload := videoTaskResponse(&service.VideoTask{Currency: "USD"})
	require.Nil(t, payload["pricing_source"])
	require.Nil(t, payload["pricing_version"])
}

func TestVideoGatewayHandlerRequiresEmployeeAPIKeyContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		method, path string
		handler      gin.HandlerFunc
	}{
		{http.MethodGet, "/v1/video/providers", (&VideoGatewayHandler{}).Providers},
		{http.MethodPost, "/v1/video/tasks", (&VideoGatewayHandler{}).Create},
		{http.MethodGet, "/v1/video/tasks/1", (&VideoGatewayHandler{}).Get},
	} {
		r := gin.New()
		r.Handle(tc.method, tc.path, tc.handler)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, http.StatusForbidden, w.Code, tc.path)
	}
}
