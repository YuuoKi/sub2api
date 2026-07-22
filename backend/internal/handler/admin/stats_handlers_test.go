package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupHandlerGetStatsReturnsKeyCounts(t *testing.T) {
	router, stub := setupAdminRouter()
	gid := int64(2)
	stub.apiKeys = []service.APIKey{
		{ID: 1, Status: service.StatusActive, GroupID: &gid},
		{ID: 2, Status: service.StatusActive, GroupID: &gid},
		{ID: 3, Status: service.StatusDisabled, GroupID: &gid},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/2/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.EqualValues(t, 3, body.Data["total_api_keys"])
	require.EqualValues(t, 2, body.Data["active_api_keys"])
}

func TestProxyHandlerGetStatsReturnsAccountCount(t *testing.T) {
	router, stub := setupAdminRouter()
	latency := int64(42)
	stub.proxyCounts = []service.ProxyWithAccountCount{{
		Proxy:         service.Proxy{ID: 4, Status: service.StatusActive},
		AccountCount:  1,
		LatencyMs:     &latency,
		LatencyStatus: "success",
	}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/4/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.EqualValues(t, 1, body.Data["total_accounts"])
	require.EqualValues(t, 1, body.Data["active_accounts"])
	require.EqualValues(t, 42, body.Data["average_latency"])
}

func TestRedeemHandlerGetStatsAggregatesCodes(t *testing.T) {
	router, stub := setupAdminRouter()
	past := time.Now().UTC().Add(-time.Hour)
	stub.redeems = []service.RedeemCode{
		{ID: 1, Type: service.RedeemTypeBalance, Value: 10, Status: service.StatusUnused},
		{ID: 2, Type: service.RedeemTypeBalance, Value: 25, Status: service.StatusUsed},
		{ID: 3, Type: service.RedeemTypeConcurrency, Value: 1, Status: service.StatusExpired},
		{ID: 4, Type: service.RedeemTypeSubscription, Value: 30, Status: service.StatusUnused, ExpiresAt: &past},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.EqualValues(t, 4, body.Data["total_codes"])
	require.EqualValues(t, 1, body.Data["active_codes"])
	require.EqualValues(t, 1, body.Data["used_codes"])
	require.EqualValues(t, 2, body.Data["expired_codes"])
	require.EqualValues(t, 25, body.Data["total_value_distributed"])
}
