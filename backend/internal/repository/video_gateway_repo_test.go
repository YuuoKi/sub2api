package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestVideoGatewayRepositoryReservesBudgetAndCreatesTaskInOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()
	task := &service.VideoTask{APIKeyID: 11, GroupID: 12, ProviderAccountID: 7, Provider: "seedance", Model: service.SeedanceModel, TaskType: "text_to_video", Prompt: "test prompt", Status: service.VideoStatusQueued, CreatedBy: 9, CreationKey: "creation-1", DurationSeconds: 4, Resolution: "720p"}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status, balance, NOW\\(\\) FROM users").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"status", "balance", "now"}).AddRow("active", 10.0, now))
	mock.ExpectQuery("FROM api_keys").WithArgs(int64(11)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id", "status", "expires_at", "quota", "quota_used", "rate_limit_5h", "rate_limit_1d", "rate_limit_7d", "usage_5h", "usage_1d", "usage_7d", "window_5h_start", "window_1d_start", "window_7d_start"}).AddRow(int64(9), int64(12), "active", nil, 1.0, 0.1, 1.0, 1.0, 1.0, 0.1, 0.1, 0.1, now, now, now))
	mock.ExpectQuery("SELECT status, subscription_type FROM groups").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"status", "subscription_type"}).AddRow("active", "standard"))
	mock.ExpectQuery("SELECT provider, default_model, enabled FROM video_provider_accounts").WithArgs(int64(7), int64(12)).WillReturnRows(sqlmock.NewRows([]string{"provider", "default_model", "enabled"}).AddRow("seedance", service.SeedanceModel, true))
	mock.ExpectExec("UPDATE users SET balance=balance").WithArgs(0.2, int64(9)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE api_keys SET quota_used").WithArgs(0.2, 0.1, 0.1, 0.1, now, now, now, int64(11), service.StatusAPIKeyQuotaExhausted).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO video_tasks")).WillReturnRows(sqlmock.NewRows([]string{"id", "version", "created_at", "updated_at"}).AddRow(int64(31), int64(1), now, now))
	mock.ExpectCommit()
	require.NoError(t, repo.ReserveAndCreateTask(context.Background(), task, 0.2))
	require.EqualValues(t, 31, task.ID)
	require.Equal(t, service.VideoReservationReserved, task.ReservationState)
	require.NotNil(t, task.BalanceBeforeUSD)
	require.Equal(t, 10.0, *task.BalanceBeforeUSD)

	mock.ExpectQuery(regexp.QuoteMeta("FROM video_tasks")).WithArgs(int64(31)).WillReturnRows(videoTaskRows(now).AddRow(int64(31), int64(11), int64(12), int64(7), "seedance", service.SeedanceModel, "text_to_video", "test prompt", "queued", "", "", "", 4, "720p", nil, 0, "USD", 0, "", "", "", "creation-1", int64(1), "pending", int64(9), now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil))
	stored, err := repo.GetTask(context.Background(), 31)
	require.NoError(t, err)
	require.Equal(t, task.Prompt, stored.Prompt)
	require.Equal(t, task.CreationKey, stored.CreationKey)
	require.Equal(t, task.CreatedBy, stored.CreatedBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryClaimsRunnableTasks(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()
	mock.ExpectQuery("FOR UPDATE SKIP LOCKED").WithArgs(2, 90).WillReturnRows(videoTaskRows(now).AddRow(int64(4), int64(21), int64(22), int64(2), "seedance", "doubao-seedance-2-0-260128", "text_to_video", "prompt", "queued", "", "", "", 4, "720p", nil, 0, "USD", 0, "", "", "", "claim-4", int64(1), "pending", int64(13), now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil))
	tasks, err := repo.ClaimRunnableTasks(context.Background(), 2, 90*time.Second)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.EqualValues(t, 4, tasks[0].ID)
	require.Equal(t, "claim-4", tasks[0].CreationKey)
	require.EqualValues(t, 13, tasks[0].CreatedBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryNilDatabaseFailsExplicitly(t *testing.T) {
	repo := NewVideoGatewayRepository(nil)
	ctx := context.Background()

	require.ErrorContains(t, repo.ReserveAndCreateTask(ctx, &service.VideoTask{}, 1), "database is required")
	_, err := repo.GetTask(ctx, 1)
	require.ErrorContains(t, err, "database is required")
	_, err = repo.ClaimRunnableTasks(ctx, 1, time.Second)
	require.ErrorContains(t, err, "database is required")
	_, err = repo.FinalizeTask(ctx, service.VideoTaskFinalization{Status: service.VideoStatusSucceeded})
	require.ErrorContains(t, err, "database is required")
}

func TestVideoGatewayRepositoryNilReceiverFailsExplicitly(t *testing.T) {
	var repo *videoGatewayRepository
	ctx := context.Background()

	require.ErrorContains(t, repo.ReserveAndCreateTask(ctx, &service.VideoTask{}, 1), "database is required")
	_, err := repo.GetTask(ctx, 1)
	require.ErrorContains(t, err, "database is required")
	_, err = repo.ClaimRunnableTasks(ctx, 1, time.Second)
	require.ErrorContains(t, err, "database is required")
	_, err = repo.FinalizeTask(ctx, service.VideoTaskFinalization{Status: service.VideoStatusSucceeded})
	require.ErrorContains(t, err, "database is required")
}

func TestVideoGatewayRepositoryTerminalFinalizationIsIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()

	tokens := int64(10)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM video_tasks WHERE id=\\$1 FOR UPDATE").WithArgs(int64(8)).WillReturnRows(videoTaskRows(now).AddRow(int64(8), int64(11), int64(12), int64(7), "seedance", service.SeedanceModel, "text_to_video", "prompt", "running", "up-8", "", "", 4, "720p", nil, 0, "USD", 1, "", "", "", "creation-8", int64(1), "accepted", int64(9), now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil))
	mock.ExpectQuery("INSERT INTO usage_billing_dedup").WithArgs("video:8", int64(11), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectQuery("SELECT request_fingerprint FROM usage_billing_dedup_archive").WithArgs("video:8", int64(11)).WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("UPDATE users SET balance=balance").WithArgs(0.1, 0.2, int64(9)).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(9.9))
	mock.ExpectExec("UPDATE api_keys SET quota_used").WithArgs(0.1, now, now, now, int64(11), service.StatusAPIKeyQuotaExhausted, service.StatusAPIKeyActive).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("UPDATE video_tasks SET status").WithArgs(int64(8), int64(1), service.VideoStatusSucceeded,
		"https://assets.invalid/result.mp4", "", "", sqlmock.AnyArg(), 0.1, "USD", "", "", now,
		service.VideoReservationCaptured, 0.1, nil, nil, nil, nil, nil, nil, 9.9, sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(int64(2)))
	mock.ExpectExec("INSERT INTO video_usage_logs").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	input := service.VideoTaskFinalization{TaskID: 8, ExpectedVersion: 1, Status: service.VideoStatusSucceeded, ResultURL: "https://assets.invalid/result.mp4", UsageTotalTokens: &tokens, CostAmount: 0.1, ProviderActualCostUSD: 0.1, Currency: "USD", Settlement: service.VideoSettlementCaptureActual, CompletedAt: now}
	result, err := repo.FinalizeTask(context.Background(), input)
	require.NoError(t, err)
	require.True(t, result.Applied)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM video_tasks WHERE id=\\$1 FOR UPDATE").WithArgs(int64(8)).WillReturnRows(videoTaskRows(now).AddRow(int64(8), int64(11), int64(12), int64(7), "seedance", service.SeedanceModel, "text_to_video", "prompt", "succeeded", "up-8", "https://assets.invalid/result.mp4", "", 4, "720p", tokens, 0.1, "USD", 1, "", "", "", "creation-8", int64(2), "accepted", int64(9), now, now, now, 0.2, "captured", now, now, now, now, 0.1, nil, nil, nil, nil, nil, nil, 10.0, 9.9, -0.1, nil, nil))
	mock.ExpectRollback()
	replay, err := repo.FinalizeTask(context.Background(), input)
	require.NoError(t, err)
	require.True(t, replay.Idempotent)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryCancelRejectsDispatchedTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRuntimeRepository(db)
	scope := service.VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 3}
	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM video_tasks").WithArgs(int64(9), int64(1), int64(2), int64(3)).WillReturnRows(videoTaskRows(now).AddRow(int64(9), int64(2), int64(3), int64(7), "seedance", service.SeedanceModel, "text_to_video", "prompt", "submitted", "up-9", "", "", 4, "720p", nil, 0, "USD", 1, "", "", "", "creation-9", int64(2), "accepted", int64(1), now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil))
	mock.ExpectRollback()
	_, err = repo.CancelTaskForScope(context.Background(), 9, scope)
	require.ErrorIs(t, err, service.ErrVideoCancelConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestValidateVideoSettlementRejectsMismatchedActualCost(t *testing.T) {
	tokens := int64(10)
	task := &service.VideoTask{ReservedCostUSD: 0.2, ReservationState: service.VideoReservationReserved}
	_, _, err := validateVideoSettlement(task, service.VideoTaskFinalization{
		Status: service.VideoStatusSucceeded, ResultURL: "https://assets.invalid/result.mp4", UsageTotalTokens: &tokens,
		CostAmount: 0.1, ProviderActualCostUSD: 0.09, Settlement: service.VideoSettlementCaptureActual,
	})
	require.ErrorContains(t, err, "provider actual cost")
}

func videoTaskRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "api_key_id", "group_id", "provider_account_id", "provider", "model", "task_type", "prompt", "status", "upstream_task_id", "result_url", "last_frame_url", "duration_seconds", "resolution", "usage_total_tokens", "cost_amount", "currency", "real_dispatch_count", "provider_error_code", "provider_error_message", "error_message", "creation_key", "version", "dispatch_state", "created_by", "created_at", "updated_at", "completed_at", "reserved_cost_usd", "reservation_state", "reserved_at", "reservation_window_5h_start", "reservation_window_1d_start", "reservation_window_7d_start", "provider_actual_cost_usd", "upstream_model", "upstream_duration_seconds", "upstream_resolution", "billing_model", "billing_duration_seconds", "billing_resolution", "balance_before_usd", "balance_after_usd", "balance_delta_usd", "authorization_consumed_at", "authorization_consumed_by"})
}
