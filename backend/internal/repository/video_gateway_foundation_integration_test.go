//go:build integration

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
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

func TestVideoGatewayFoundationRepositoryLifecycle(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UTC().UnixNano()
	var userID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, status, balance, concurrency)
		VALUES ($1, 'integration-only', 'user', 'active', 0, 1)
		RETURNING id`, fmt.Sprintf("video-foundation-%d@invalid.test", suffix)).Scan(&userID))
	var providerID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO video_provider_accounts (provider, display_name)
		VALUES ('mock', $1) RETURNING id`, fmt.Sprintf("integration-%d", suffix)).Scan(&providerID))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM video_tasks WHERE created_by = $1", userID)
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM video_provider_accounts WHERE id = $1", providerID)
		_, _ = integrationDB.ExecContext(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})

	repo := NewVideoGatewayRepository(integrationDB)
	creationKey := fmt.Sprintf("video-foundation-%d", suffix)
	task := &service.VideoTask{
		ProviderAccountID: providerID,
		Provider:          "mock",
		Model:             "mock-v1",
		TaskType:          "text_to_video",
		Prompt:            "integration prompt",
		Status:            service.VideoStatusQueued,
		CreationKey:       creationKey,
		CreatedBy:         userID,
	}
	require.NoError(t, repo.CreateTask(ctx, task))
	require.NotZero(t, task.ID)

	stored, err := repo.GetTask(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, creationKey, stored.CreationKey)
	require.Equal(t, userID, stored.CreatedBy)
	require.Equal(t, task.Prompt, stored.Prompt)

	duplicate := *task
	duplicate.ID = 0
	err = repo.CreateTask(ctx, &duplicate)
	requirePostgresUniqueViolation(t, err)

	claimed, err := repo.ClaimRunnableTasks(ctx, 1, time.Minute)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, task.ID, claimed[0].ID)
	require.Equal(t, creationKey, claimed[0].CreationKey)
	require.Equal(t, userID, claimed[0].CreatedBy)
	var leaseActive bool
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT worker_claimed_until > NOW() FROM video_tasks WHERE id = $1", task.ID).Scan(&leaseActive))
	require.True(t, leaseActive)
	claimedAgain, err := repo.ClaimRunnableTasks(ctx, 1, time.Minute)
	require.NoError(t, err)
	require.Empty(t, claimedAgain)

	completedAt := time.Now().UTC()
	finalized, err := repo.FinalizeTask(ctx, service.VideoTaskFinalization{
		TaskID:          task.ID,
		ExpectedVersion: task.Version,
		Status:          service.VideoStatusSucceeded,
		ResultURL:       "https://assets.invalid/integration-result.mp4",
		CompletedAt:     completedAt,
	})
	require.NoError(t, err)
	require.True(t, finalized.Applied)
	require.EqualValues(t, 2, finalized.Version)

	replay, err := repo.FinalizeTask(ctx, service.VideoTaskFinalization{
		TaskID:          task.ID,
		ExpectedVersion: task.Version,
		Status:          service.VideoStatusSucceeded,
		ResultURL:       "https://assets.invalid/integration-result.mp4",
		CompletedAt:     completedAt,
	})
	require.NoError(t, err)
	require.True(t, replay.Idempotent)

	_, err = repo.FinalizeTask(ctx, service.VideoTaskFinalization{
		TaskID:          task.ID,
		ExpectedVersion: task.Version,
		Status:          service.VideoStatusFailed,
		CompletedAt:     completedAt,
	})
	require.ErrorIs(t, err, service.ErrVideoTaskTerminalConflict)

	_, err = integrationDB.ExecContext(ctx, `
		INSERT INTO video_usage_logs (video_task_id, provider, model, status)
		VALUES ($1, 'mock', 'mock-v1', 'succeeded')`, task.ID)
	requirePostgresUniqueViolation(t, err)
}

func requirePostgresUniqueViolation(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var pqErr *pq.Error
	require.True(t, errors.As(err, &pqErr), "expected PostgreSQL error, got %T", err)
	require.Equal(t, pq.ErrorCode("23505"), pqErr.Code)
}
