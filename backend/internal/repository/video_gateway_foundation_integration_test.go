//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoGatewayFoundationSchemaSmoke(t *testing.T) {
	for _, table := range []string{
		"video_provider_accounts",
		"video_tasks",
		"video_task_events",
		"video_usage_logs",
		"video_daily_trial_reservations",
	} {
		var regclass sql.NullString
		require.NoError(t, integrationDB.QueryRowContext(context.Background(), "SELECT to_regclass('public.' || $1)", table).Scan(&regclass))
		require.True(t, regclass.Valid, "expected table %s", table)
	}

	for _, index := range []string{"uq_video_tasks_creation_key", "uq_video_usage_logs_video_task_id"} {
		var unique bool
		require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
			SELECT i.indisunique
			FROM pg_class c JOIN pg_index i ON i.indexrelid = c.oid
			WHERE c.relname = $1`, index).Scan(&unique))
		require.True(t, unique, "expected unique index %s", index)
	}
}
