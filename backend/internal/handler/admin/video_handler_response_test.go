package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestVideoTaskAdminResponseIncludesPersistedSpecificationEvidence(t *testing.T) {
	tokens := int64(321)
	reservedAt := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)
	response := videoTaskAdminResponse(&service.VideoTask{
		ID:                    41,
		APIKeyID:              12,
		GroupID:               13,
		Model:                 service.SeedanceModel,
		DurationSeconds:       4,
		Resolution:            "720p",
		UsageTotalTokens:      &tokens,
		ReservedCostUSD:       0.2,
		ReservationState:      service.VideoReservationCaptured,
		ReservedAt:            &reservedAt,
		CostAmount:            0.1,
		ProviderActualCostUSD: 0.08,
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
	require.Equal(t, service.SeedanceModel, response["request_model"])
	require.Nil(t, response["upstream_model"])
	require.Nil(t, response["billing_model"])
	require.Nil(t, response["balance_before_usd"])
	require.Nil(t, response["balance_after_usd"])
	require.Nil(t, response["authorization_consumed_at"])
}
