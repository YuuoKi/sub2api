package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestListEnabledVideoProvidersKeepsSecretsSanitizedButCatalogMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRuntimeRepository(db)
	mock.ExpectQuery("SELECT id, group_id, provider, display_name, enabled, '', masked_key, base_url, default_model").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "group_id", "provider", "display_name", "enabled",
			"encrypted_api_key", "masked_key", "base_url", "default_model",
		}).AddRow(
			int64(7), int64(9), service.HCAtomSeedanceV3Provider, "HC V3", true,
			"", "yh****perx", service.HCAtomSeedanceV3BaseURL, service.HCAtomSeedanceV3PublicModel,
		))

	items, err := repo.ListEnabledVideoProviders(context.Background(), 9)

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.True(t, items[0].APIKeyConfigured)
	require.Empty(t, items[0].EncryptedAPIKey)
	require.Equal(t, service.HCAtomSeedanceV3BaseURL, items[0].BaseURL)
	encoded, err := json.Marshal(items[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), service.HCAtomSeedanceV3BaseURL)
	require.NotContains(t, string(encoded), "encrypted_api_key")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryRejectsInvalidPricingSnapshotBeforeTransaction(t *testing.T) {
	tests := []struct {
		name       string
		maximumUSD float64
		mutate     func(*service.VideoTask)
	}{
		{name: "all provenance missing", maximumUSD: 0.2, mutate: func(task *service.VideoTask) {
			task.Currency, task.PricingSource, task.PricingVersion = "", "", ""
			task.PricingCNYPerMillionCompletionTokens = nil
			task.PricingUSDCNYExchangeRate = nil
			task.PricingMaximumCNY = nil
		}},
		{name: "partial snapshot", maximumUSD: 0.2, mutate: func(task *service.VideoTask) { task.PricingUSDCNYExchangeRate = nil }},
		{name: "unsupported currency", maximumUSD: 0.2, mutate: func(task *service.VideoTask) { task.Currency = "CNY" }},
		{name: "unsupported source", maximumUSD: 0.2, mutate: func(task *service.VideoTask) { task.PricingSource = "request.body" }},
		{name: "unsupported version", maximumUSD: 0.2, mutate: func(task *service.VideoTask) { task.PricingVersion = "seedance_v2" }},
		{name: "zero token price", maximumUSD: 0.2, mutate: func(task *service.VideoTask) { value := 0.0; task.PricingCNYPerMillionCompletionTokens = &value }},
		{name: "nan token price", maximumUSD: 0.2, mutate: func(task *service.VideoTask) { value := math.NaN(); task.PricingCNYPerMillionCompletionTokens = &value }},
		{name: "infinite exchange rate", maximumUSD: 0.2, mutate: func(task *service.VideoTask) { value := math.Inf(1); task.PricingUSDCNYExchangeRate = &value }},
		{name: "nan maximum cny", maximumUSD: 0.2, mutate: func(task *service.VideoTask) { value := math.NaN(); task.PricingMaximumCNY = &value }},
		{name: "non-finite maximum usd", maximumUSD: math.Inf(1), mutate: func(*service.VideoTask) {}},
		{name: "maximum does not match snapshot", maximumUSD: 0.21, mutate: func(*service.VideoTask) {}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			task := validVideoPricingTaskForRepository()
			tc.mutate(task)

			err = NewVideoGatewayRepository(db).ReserveAndCreateTask(context.Background(), task, tc.maximumUSD)
			require.ErrorContains(t, err, "video pricing snapshot")
			require.Zero(t, task.ID)
			require.Zero(t, task.ReservedCostUSD)
			require.Empty(t, task.ReservationState)
			require.Zero(t, task.RealDispatchCount)
			require.NoError(t, mock.ExpectationsWereMet(), "invalid snapshot must fail before transaction or ledger access")
		})
	}
}

func validVideoPricingTaskForRepository() *service.VideoTask {
	price, rate, maximum := 2.0, 7.0, 1.4
	return &service.VideoTask{
		APIKeyID: 11, GroupID: 12, ProviderAccountID: 7, Provider: "seedance", Model: service.SeedanceModel,
		TaskType: "text_to_video", Prompt: "test prompt", Status: service.VideoStatusQueued, CreatedBy: 9,
		CreationKey: "creation-1", DurationSeconds: 4, Resolution: "720p", Currency: "USD",
		PricingSource: service.VideoPricingSourceConfig, PricingVersion: service.VideoPricingVersionSeedanceCompletionTokensUSDV1,
		PricingCNYPerMillionCompletionTokens: &price, PricingUSDCNYExchangeRate: &rate, PricingMaximumCNY: &maximum,
	}
}

func TestVideoGatewayRepositoryReservesBudgetAndCreatesTaskInOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()
	price, rate, maximum := 2.0, 7.0, 1.4
	task := &service.VideoTask{APIKeyID: 11, GroupID: 12, ProviderAccountID: 7, Provider: "seedance", Model: service.SeedanceModel, TaskType: "text_to_video", Prompt: "test prompt", Status: service.VideoStatusQueued, CreatedBy: 9, CreationKey: "creation-1", DurationSeconds: 4, Resolution: "720p", Currency: "USD", PricingSource: service.VideoPricingSourceConfig, PricingVersion: service.VideoPricingVersionSeedanceCompletionTokensUSDV1, PricingCNYPerMillionCompletionTokens: &price, PricingUSDCNYExchangeRate: &rate, PricingMaximumCNY: &maximum}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status, balance, NOW\\(\\) FROM users").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"status", "balance", "now"}).AddRow("active", 10.0, now))
	mock.ExpectQuery("FROM api_keys").WithArgs(int64(11)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "group_id", "status", "expires_at", "quota", "quota_used", "rate_limit_5h", "rate_limit_1d", "rate_limit_7d", "usage_5h", "usage_1d", "usage_7d", "window_5h_start", "window_1d_start", "window_7d_start"}).AddRow(int64(9), int64(12), "active", nil, 1.0, 0.1, 1.0, 1.0, 1.0, 0.1, 0.1, 0.1, now, now, now))
	mock.ExpectQuery("SELECT status, subscription_type FROM groups").WithArgs(int64(12)).WillReturnRows(sqlmock.NewRows([]string{"status", "subscription_type"}).AddRow("active", "standard"))
	mock.ExpectQuery("SELECT provider, default_model, enabled FROM video_provider_accounts").WithArgs(int64(7), int64(12)).WillReturnRows(sqlmock.NewRows([]string{"provider", "default_model", "enabled"}).AddRow("seedance", service.SeedanceModel, true))
	mock.ExpectExec("UPDATE users SET balance=balance").WithArgs(0.2, int64(9)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE api_keys SET quota_used").WithArgs(0.2, 0.1, 0.1, 0.1, now, now, now, int64(11), service.StatusAPIKeyQuotaExhausted).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO video_tasks[\s\S]+pricing_source[\s\S]+pricing_version[\s\S]+pricing_cny_per_million_completion_tokens[\s\S]+pricing_usd_cny_exchange_rate[\s\S]+pricing_maximum_cny`).
		WithArgs(int64(7), "seedance", service.SeedanceModel, "text_to_video", "test prompt", sqlmock.AnyArg(), service.VideoStatusQueued,
			"creation-1", int64(9), int64(11), int64(12), 4, "720p", 0.2, service.VideoReservationReserved,
			now, now, now, now, 10.0, "USD", service.VideoPricingSourceConfig,
			service.VideoPricingVersionSeedanceCompletionTokensUSDV1, 2.0, 7.0, 1.4).
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "created_at", "updated_at"}).AddRow(int64(31), int64(1), now, now))
	mock.ExpectCommit()
	require.NoError(t, repo.ReserveAndCreateTask(context.Background(), task, 0.2))
	require.EqualValues(t, 31, task.ID)
	require.Equal(t, service.VideoReservationReserved, task.ReservationState)
	require.NotNil(t, task.BalanceBeforeUSD)
	require.Equal(t, 10.0, *task.BalanceBeforeUSD)

	mock.ExpectQuery(regexp.QuoteMeta("FROM video_tasks")).WithArgs(int64(31)).WillReturnRows(videoTaskRows(now).AddRow(int64(31), int64(11), int64(12), int64(7), "seedance", service.SeedanceModel, "text_to_video", "test prompt", "{}", "queued", "", "", "", 4, "720p", nil, 0, "USD", service.VideoPricingSourceConfig, service.VideoPricingVersionSeedanceCompletionTokensUSDV1, 2.0, 7.0, 1.4, 0, "", "", "", "creation-1", int64(1), "pending", int64(9), now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil, nil, nil))
	stored, err := repo.GetTask(context.Background(), 31)
	require.NoError(t, err)
	require.Equal(t, task.Prompt, stored.Prompt)
	require.Equal(t, task.CreationKey, stored.CreationKey)
	require.Equal(t, task.CreatedBy, stored.CreatedBy)
	require.Equal(t, service.VideoPricingSourceConfig, stored.PricingSource)
	require.Equal(t, service.VideoPricingVersionSeedanceCompletionTokensUSDV1, stored.PricingVersion)
	require.Equal(t, 2.0, *stored.PricingCNYPerMillionCompletionTokens)
	require.Equal(t, 7.0, *stored.PricingUSDCNYExchangeRate)
	require.Equal(t, 1.4, *stored.PricingMaximumCNY)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryClaimsRunnableTasks(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()
	mock.ExpectQuery("FOR UPDATE SKIP LOCKED").WithArgs(2, 90).WillReturnRows(videoTaskRows(now).AddRow(int64(4), int64(21), int64(22), int64(2), "seedance", "doubao-seedance-2-0-260128", "text_to_video", "prompt", "{}", "queued", "", "", "", 4, "720p", nil, 0, "USD", nil, nil, nil, nil, nil, 0, "", "", "", "claim-4", int64(1), "pending", int64(13), now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil, nil, nil))
	tasks, err := repo.ClaimRunnableTasks(context.Background(), 2, 90*time.Second)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.EqualValues(t, 4, tasks[0].ID)
	require.Equal(t, "claim-4", tasks[0].CreationKey)
	require.EqualValues(t, 13, tasks[0].CreatedBy)
	require.Nil(t, tasks[0].PricingCNYPerMillionCompletionTokens)
	require.Empty(t, tasks[0].PricingSource)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryGetTaskForOwnerDistinguishesMissingAndForeign(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoGatewayRepository{db: db}
	now := time.Now().UTC()
	row := func(ownerID int64) *sqlmock.Rows {
		return videoTaskRows(now).AddRow(int64(42), int64(11), int64(12), int64(7), "seedance", service.SeedanceModel, "text_to_video", "prompt", "{}", "succeeded", "up-42", "https://assets.example.test/result.mp4", "", 4, "720p", int64(100), 0.1, "USD", nil, nil, nil, nil, nil, 1, "", "", "", "creation-42", int64(2), "accepted", ownerID, now, now, now, 0.2, "captured", now, now, now, now, 0.1, nil, nil, nil, nil, nil, nil, 10.0, 9.9, -0.1, nil, nil, nil, nil)
	}

	mock.ExpectQuery("FROM video_tasks WHERE id = \\$1").WithArgs(int64(42)).WillReturnRows(row(9))
	task, err := repo.GetTaskForOwner(context.Background(), 42, 9)
	require.NoError(t, err)
	require.EqualValues(t, 9, task.CreatedBy)

	mock.ExpectQuery("FROM video_tasks WHERE id = \\$1").WithArgs(int64(42)).WillReturnRows(row(10))
	_, err = repo.GetTaskForOwner(context.Background(), 42, 9)
	require.ErrorIs(t, err, service.ErrVideoTaskForbidden)

	mock.ExpectQuery("FROM video_tasks WHERE id = \\$1").WithArgs(int64(404)).WillReturnError(sql.ErrNoRows)
	_, err = repo.GetTaskForOwner(context.Background(), 404, 9)
	require.ErrorIs(t, err, service.ErrVideoTaskNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositorySetsLocalAssetOnlyForReadyUnarchivedTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoGatewayRepository{db: db}
	savedAt := time.Now().UTC()

	mock.ExpectExec("UPDATE video_tasks SET local_asset_path").
		WithArgs(int64(42), "assets/video/42/result.mp4", savedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.SetTaskLocalAsset(context.Background(), 42, "assets/video/42/result.mp4", savedAt))

	mock.ExpectExec("UPDATE video_tasks SET local_asset_path").
		WithArgs(int64(43), "assets/video/43/result.mp4", savedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	err = repo.SetTaskLocalAsset(context.Background(), 43, "assets/video/43/result.mp4", savedAt)
	require.ErrorIs(t, err, service.ErrVideoTaskTerminalConflict)
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
	mock.ExpectQuery("SELECT .* FROM video_tasks WHERE id=\\$1 FOR UPDATE").WithArgs(int64(8)).WillReturnRows(videoTaskRows(now).AddRow(int64(8), int64(11), int64(12), int64(7), "seedance", service.SeedanceModel, "text_to_video", "prompt", "{}", "running", "up-8", "", "", 4, "720p", nil, 0, "USD", nil, nil, nil, nil, nil, 1, "", "", "", "creation-8", int64(1), "accepted", int64(9), now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil, nil, nil))
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
	mock.ExpectQuery("SELECT .* FROM video_tasks WHERE id=\\$1 FOR UPDATE").WithArgs(int64(8)).WillReturnRows(videoTaskRows(now).AddRow(int64(8), int64(11), int64(12), int64(7), "seedance", service.SeedanceModel, "text_to_video", "prompt", "{}", "succeeded", "up-8", "https://assets.invalid/result.mp4", "", 4, "720p", tokens, 0.1, "USD", nil, nil, nil, nil, nil, 1, "", "", "", "creation-8", int64(2), "accepted", int64(9), now, now, now, 0.2, "captured", now, now, now, now, 0.1, nil, nil, nil, nil, nil, nil, 10.0, 9.9, -0.1, nil, nil, nil, nil))
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
	mock.ExpectQuery("SELECT .* FROM video_tasks").WithArgs(int64(9), int64(1), int64(2), int64(3)).WillReturnRows(videoTaskRows(now).AddRow(int64(9), int64(2), int64(3), int64(7), "seedance", service.SeedanceModel, "text_to_video", "prompt", "{}", "submitted", "up-9", "", "", 4, "720p", nil, 0, "USD", nil, nil, nil, nil, nil, 1, "", "", "", "creation-9", int64(2), "accepted", int64(1), now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil, nil, nil))
	mock.ExpectRollback()
	_, err = repo.CancelTaskForScope(context.Background(), 9, scope)
	require.ErrorIs(t, err, service.ErrVideoCancelConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryBeginHCAtomV3DispatchRequiresFixedEnabledProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewVideoGatewayRuntimeRepository(db)
	mock.ExpectExec("UPDATE video_tasks task SET real_dispatch_count=1").WithArgs(int64(7), int64(3), service.HCAtomSeedanceV3Provider, service.HCAtomSeedanceV3BaseURL, service.HCAtomSeedanceV3PublicModel).WillReturnResult(sqlmock.NewResult(0, 1))
	ok, err := repo.BeginHCAtomV3Dispatch(context.Background(), 7, 3)
	require.NoError(t, err)
	require.True(t, ok)
	mock.ExpectExec("UPDATE video_tasks task SET real_dispatch_count=1").WithArgs(int64(8), int64(3), service.HCAtomSeedanceV3Provider, service.HCAtomSeedanceV3BaseURL, service.HCAtomSeedanceV3PublicModel).WillReturnResult(sqlmock.NewResult(0, 0))
	ok, err = repo.BeginHCAtomV3Dispatch(context.Background(), 8, 3)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryBeginProductionDispatchRequiresLiveBillingAndProviderState(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRuntimeRepository(db)

	mock.ExpectExec("UPDATE video_tasks task SET[\\s\\S]+task.reservation_state=\\$6[\\s\\S]+provider.encrypted_api_key<>''[\\s\\S]+candidate.models_list_config[\\s\\S]+video_price_720p[\\s\\S]+key.status=\\$7").
		WithArgs(
			int64(7), int64(3), service.HCAtomSeedanceV3Provider,
			service.HCAtomSeedanceV3BaseURL, service.HCAtomSeedanceV3PublicModel,
			service.VideoReservationReserved, service.StatusAPIKeyActive,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	ok, err := repo.BeginProductionDispatch(
		context.Background(), 7, 3, service.HCAtomSeedanceV3Provider,
		service.HCAtomSeedanceV3BaseURL, service.HCAtomSeedanceV3PublicModel,
	)

	require.NoError(t, err)
	require.True(t, ok)
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
	return sqlmock.NewRows([]string{"id", "api_key_id", "group_id", "provider_account_id", "provider", "model", "task_type", "prompt", "request_payload", "status", "upstream_task_id", "result_url", "last_frame_url", "duration_seconds", "resolution", "usage_total_tokens", "cost_amount", "currency", "pricing_source", "pricing_version", "pricing_cny_per_million_completion_tokens", "pricing_usd_cny_exchange_rate", "pricing_maximum_cny", "real_dispatch_count", "provider_error_code", "provider_error_message", "error_message", "creation_key", "version", "dispatch_state", "created_by", "created_at", "updated_at", "completed_at", "reserved_cost_usd", "reservation_state", "reserved_at", "reservation_window_5h_start", "reservation_window_1d_start", "reservation_window_7d_start", "provider_actual_cost_usd", "upstream_model", "upstream_duration_seconds", "upstream_resolution", "billing_model", "billing_duration_seconds", "billing_resolution", "balance_before_usd", "balance_after_usd", "balance_delta_usd", "authorization_consumed_at", "authorization_consumed_by", "local_asset_path", "local_asset_saved_at"})
}
