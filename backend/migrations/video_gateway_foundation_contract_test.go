package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoGatewayFoundationMigrationContract(t *testing.T) {
	sqlBytes, err := FS.ReadFile("174_wujie_video_gateway_foundation.sql")
	require.NoError(t, err)
	sql := string(sqlBytes)

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS video_provider_accounts",
		"CREATE TABLE IF NOT EXISTS video_tasks",
		"CREATE TABLE IF NOT EXISTS video_task_events",
		"CREATE TABLE IF NOT EXISTS video_usage_logs",
		"CREATE TABLE IF NOT EXISTS video_daily_trial_reservations",
		"UNIQUE (provider, created_by, trial_date)",
		"uq_video_tasks_creation_key",
		"uq_video_usage_logs_video_task_id",
		"worker_claimed_until",
		"balance_charged_at",
	} {
		require.Contains(t, sql, fragment)
	}
	require.NotContains(t, strings.ToLower(sql), "insert into video_provider_accounts")
}
