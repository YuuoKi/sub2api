//go:build integration

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestVideoGatewayFoundationSchemaSmoke(t *testing.T) {
	for _, table := range []string{"video_provider_accounts", "video_tasks", "video_task_events", "video_usage_logs", "video_daily_trial_reservations", "video_single_smoke_consumptions"} {
		var regclass sql.NullString
		require.NoError(t, integrationDB.QueryRowContext(context.Background(), "SELECT to_regclass('public.' || $1)", table).Scan(&regclass))
		require.True(t, regclass.Valid, "expected table %s", table)
	}
	for _, column := range []string{"reserved_cost_usd", "reservation_state", "provider_actual_cost_usd"} {
		var exists bool
		require.NoError(t, integrationDB.QueryRowContext(context.Background(), `SELECT EXISTS(
			SELECT 1 FROM information_schema.columns WHERE table_name='video_tasks' AND column_name=$1)`, column).Scan(&exists))
		require.True(t, exists, "expected video_tasks.%s", column)
	}
	for column, expectedType := range map[string]string{
		"pricing_source":  "text",
		"pricing_version": "text",
		"pricing_cny_per_million_completion_tokens": "numeric",
		"pricing_usd_cny_exchange_rate":             "numeric",
		"pricing_maximum_cny":                       "numeric",
	} {
		var dataType, nullable string
		require.NoError(t, integrationDB.QueryRowContext(context.Background(), `SELECT data_type,is_nullable
			FROM information_schema.columns WHERE table_schema='public' AND table_name='video_tasks' AND column_name=$1`, column).
			Scan(&dataType, &nullable))
		require.Equal(t, expectedType, dataType, "video_tasks.%s type", column)
		require.Equal(t, "YES", nullable, "video_tasks.%s must preserve historical unknowns", column)
	}
}

type videoIntegrationFixture struct {
	repo       service.VideoGatewayRuntimeRepository
	userID     int64
	groupID    int64
	apiKeyID   int64
	providerID int64
	task       *service.VideoTask
}

func newVideoIntegrationFixture(t *testing.T, balance, quota, rate float64, groupType string, reserve bool) *videoIntegrationFixture {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	var f videoIntegrationFixture
	if groupType == "" {
		groupType = "standard"
	}
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO users
		(email,password_hash,role,status,balance,concurrency) VALUES ($1,'x','user','active',$2,1) RETURNING id`,
		fmt.Sprintf("video-%d@invalid.test", suffix), balance).Scan(&f.userID))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO groups
		(name,platform,status,rate_multiplier,subscription_type) VALUES ($1,'openai','active',1,$2) RETURNING id`,
		fmt.Sprintf("video-group-%d", suffix), groupType).Scan(&f.groupID))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO api_keys
		(user_id,key,name,group_id,status,quota,rate_limit_5h,rate_limit_1d,rate_limit_7d)
		VALUES ($1,$2,'video',$3,'active',$4,$5,$5,$5) RETURNING id`, f.userID, fmt.Sprintf("video-key-%d", suffix), f.groupID, quota, rate).Scan(&f.apiKeyID))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO video_provider_accounts
		(provider,display_name,enabled,group_id,default_model,base_url,tiny_real_authorized_at,tiny_real_authorized_by)
		VALUES ('seedance',$1,true,$2,$3,$4,NOW(),$5) RETURNING id`,
		fmt.Sprintf("provider-%d", suffix), f.groupID, service.SeedanceModel, service.SeedanceBaseURL, f.userID).Scan(&f.providerID))
	f.repo = NewVideoGatewayRuntimeRepository(integrationDB)
	if reserve {
		f.task = newVideoIntegrationTask(&f, "integration", fmt.Sprintf("video-create-%d", suffix))
		require.NoError(t, f.repo.ReserveAndCreateTask(ctx, f.task, 0.2))
	}
	t.Cleanup(func() {
		_, _ = integrationDB.Exec(`DELETE FROM video_single_smoke_consumptions WHERE video_task_id IN (SELECT id FROM video_tasks WHERE created_by=$1)`, f.userID)
		_, _ = integrationDB.Exec(`DELETE FROM usage_billing_dedup WHERE api_key_id=$1`, f.apiKeyID)
		_, _ = integrationDB.Exec(`DELETE FROM video_tasks WHERE created_by=$1`, f.userID)
		_, _ = integrationDB.Exec(`DELETE FROM video_provider_accounts WHERE id=$1`, f.providerID)
		_, _ = integrationDB.Exec(`DELETE FROM api_keys WHERE id=$1`, f.apiKeyID)
		_, _ = integrationDB.Exec(`DELETE FROM groups WHERE id=$1`, f.groupID)
		_, _ = integrationDB.Exec(`DELETE FROM users WHERE id=$1`, f.userID)
	})
	return &f
}

func newVideoIntegrationTask(f *videoIntegrationFixture, prompt, creationKey string) *service.VideoTask {
	price, rate, maximum := 2.0, 7.0, 1.4
	return &service.VideoTask{APIKeyID: f.apiKeyID, GroupID: f.groupID, ProviderAccountID: f.providerID,
		Provider: "seedance", Model: service.SeedanceModel, TaskType: "text_to_video", Prompt: prompt,
		Status: service.VideoStatusQueued, CreationKey: creationKey, CreatedBy: f.userID,
		DurationSeconds: 4, Resolution: "720p", Currency: "USD",
		PricingSource: service.VideoPricingSourceConfig, PricingVersion: service.VideoPricingVersionSeedanceCompletionTokensUSDV1,
		PricingCNYPerMillionCompletionTokens: &price, PricingUSDCNYExchangeRate: &rate, PricingMaximumCNY: &maximum}
}

func TestVideoGatewayHistoricalPricingProvenanceRemainsNullable(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", false)
	var id int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO video_tasks
		(provider_account_id,provider,model,task_type,prompt,status,created_by,api_key_id,group_id,duration_seconds,resolution)
		VALUES ($1,'seedance',$2,'text_to_video','historical','queued',$3,$4,$5,4,'720p') RETURNING id`,
		f.providerID, service.SeedanceModel, f.userID, f.apiKeyID, f.groupID).Scan(&id))

	task, err := f.repo.GetTask(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "USD", task.Currency)
	require.Empty(t, task.PricingSource)
	require.Empty(t, task.PricingVersion)
	require.Nil(t, task.PricingCNYPerMillionCompletionTokens)
	require.Nil(t, task.PricingUSDCNYExchangeRate)
	require.Nil(t, task.PricingMaximumCNY)
}

func TestVideoGatewayInvalidPricingSnapshotFailsBeforeLedgerInsertAndCounters(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", false)
	task := newVideoIntegrationTask(f, "invalid pricing", fmt.Sprintf("invalid-pricing-%d", time.Now().UnixNano()))
	task.PricingVersion = ""

	var balanceBefore, frozenBefore, quotaBefore, usage5hBefore, usage1dBefore, usage7dBefore float64
	var taskCountBefore, dispatchCountBefore int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, f.userID).Scan(&balanceBefore, &frozenBefore))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT quota_used,usage_5h,usage_1d,usage_7d FROM api_keys WHERE id=$1`, f.apiKeyID).
		Scan(&quotaBefore, &usage5hBefore, &usage1dBefore, &usage7dBefore))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_tasks WHERE created_by=$1`, f.userID).Scan(&taskCountBefore))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_single_smoke_consumptions`).Scan(&dispatchCountBefore))

	err := f.repo.ReserveAndCreateTask(ctx, task, 0.2)
	require.ErrorIs(t, err, service.ErrVideoPricingSnapshotInvalid)

	var balanceAfter, frozenAfter, quotaAfter, usage5hAfter, usage1dAfter, usage7dAfter float64
	var taskCountAfter, dispatchCountAfter int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, f.userID).Scan(&balanceAfter, &frozenAfter))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT quota_used,usage_5h,usage_1d,usage_7d FROM api_keys WHERE id=$1`, f.apiKeyID).
		Scan(&quotaAfter, &usage5hAfter, &usage1dAfter, &usage7dAfter))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_tasks WHERE created_by=$1`, f.userID).Scan(&taskCountAfter))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_single_smoke_consumptions`).Scan(&dispatchCountAfter))
	require.Equal(t, []float64{balanceBefore, frozenBefore, quotaBefore, usage5hBefore, usage1dBefore, usage7dBefore},
		[]float64{balanceAfter, frozenAfter, quotaAfter, usage5hAfter, usage1dAfter, usage7dAfter})
	require.Equal(t, taskCountBefore, taskCountAfter)
	require.Equal(t, dispatchCountBefore, dispatchCountAfter)
	require.Zero(t, task.ID)
	require.Zero(t, task.RealDispatchCount)
}

func TestVideoGatewayReservationChecksBalanceQuotaRateAndGroup(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name                 string
		balance, quota, rate float64
		groupType            string
	}{
		{name: "balance", balance: 0.1, quota: 1, rate: 1},
		{name: "quota", balance: 1, quota: 0.1, rate: 1},
		{name: "rate", balance: 1, quota: 1, rate: 0.1},
		{name: "subscription_group", balance: 1, quota: 1, rate: 1, groupType: "subscription"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newVideoIntegrationFixture(t, tc.balance, tc.quota, tc.rate, tc.groupType, false)
			task := newVideoIntegrationTask(f, "x", fmt.Sprintf("reject-%s-%d", tc.name, time.Now().UnixNano()))
			err := f.repo.ReserveAndCreateTask(ctx, task, 0.2)
			require.ErrorIs(t, err, service.ErrVideoBudgetRejected)
			var count int
			require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_tasks WHERE created_by=$1`, f.userID).Scan(&count))
			require.Zero(t, count)
		})
	}
}

func TestVideoGatewayReservationAndCancelReleaseAllHolds(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	var balance, frozen, quotaUsed, usage5h, usage1d, usage7d float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, f.userID).Scan(&balance, &frozen))
	require.Equal(t, 9.8, balance)
	require.Equal(t, 0.2, frozen)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT quota_used,usage_5h,usage_1d,usage_7d FROM api_keys WHERE id=$1`, f.apiKeyID).Scan(&quotaUsed, &usage5h, &usage1d, &usage7d))
	require.Equal(t, []float64{0.2, 0.2, 0.2, 0.2}, []float64{quotaUsed, usage5h, usage1d, usage7d})
	cancelled, err := f.repo.CancelTaskForScope(ctx, f.task.ID, service.VideoTaskScope{UserID: f.userID, APIKeyID: f.apiKeyID, GroupID: f.groupID})
	require.NoError(t, err)
	require.Equal(t, service.VideoReservationReleased, cancelled.ReservationState)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, f.userID).Scan(&balance, &frozen))
	require.Equal(t, 10.0, balance)
	require.Zero(t, frozen)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT quota_used,usage_5h,usage_1d,usage_7d FROM api_keys WHERE id=$1`, f.apiKeyID).Scan(&quotaUsed, &usage5h, &usage1d, &usage7d))
	require.Equal(t, []float64{0.0, 0.0, 0.0, 0.0}, []float64{quotaUsed, usage5h, usage1d, usage7d})
}

func TestVideoGatewayAtomicSuccessCaptureAndDedup(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	tokens := int64(100)
	input := service.VideoTaskFinalization{TaskID: f.task.ID, ExpectedVersion: f.task.Version, Status: service.VideoStatusSucceeded,
		ResultURL: "https://cdn.example.test/video.mp4", UsageTotalTokens: &tokens, CostAmount: 0.05,
		ProviderActualCostUSD: 0.05, Currency: "USD", Settlement: service.VideoSettlementCaptureActual, CompletedAt: time.Now().UTC()}
	result, err := f.repo.FinalizeTask(ctx, input)
	require.NoError(t, err)
	require.True(t, result.Applied)
	replay, err := f.repo.FinalizeTask(ctx, input)
	require.NoError(t, err)
	require.True(t, replay.Idempotent)
	var balance, frozen, quotaUsed, usage5h, cost float64
	var state, status, resultURL string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, f.userID).Scan(&balance, &frozen))
	require.Equal(t, 9.95, balance)
	require.Zero(t, frozen)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT quota_used,usage_5h FROM api_keys WHERE id=$1`, f.apiKeyID).Scan(&quotaUsed, &usage5h))
	require.Equal(t, 0.05, quotaUsed)
	require.Equal(t, 0.05, usage5h)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT status,reservation_state,cost_amount,result_url FROM video_tasks WHERE id=$1`, f.task.ID).Scan(&status, &state, &cost, &resultURL))
	require.Equal(t, service.VideoStatusSucceeded, status)
	require.Equal(t, service.VideoReservationCaptured, state)
	require.Equal(t, 0.05, cost)
	require.NotEmpty(t, resultURL)
	var dedup, logs int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id=$1 AND api_key_id=$2`, fmt.Sprintf("video:%d", f.task.ID), f.apiKeyID).Scan(&dedup))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_usage_logs WHERE video_task_id=$1 AND charged_cost_usd=0.05`, f.task.ID).Scan(&logs))
	require.Equal(t, 1, dedup)
	require.Equal(t, 1, logs)
}

func TestVideoGatewayOverBudgetCapturesMaximumAndPreservesAsset(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	tokens := int64(1_000_000)
	input := service.VideoTaskFinalization{TaskID: f.task.ID, ExpectedVersion: f.task.Version, Status: service.VideoStatusFailed,
		ResultURL: "https://cdn.example.test/over.mp4", UsageTotalTokens: &tokens, CostAmount: 0.2, ProviderActualCostUSD: 0.3,
		Currency: "USD", Settlement: service.VideoSettlementCaptureReserved, ProviderErrorCode: "budget_actual_exceeded",
		ProviderErrorMessage: "provider cost exceeded reserved maximum", ErrorMessage: "provider cost exceeded reserved maximum", CompletedAt: time.Now().UTC()}
	_, err := f.repo.FinalizeTask(ctx, input)
	require.NoError(t, err)
	var balance, frozen, quotaUsed, charged, providerCost float64
	var status, state, resultURL string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, f.userID).Scan(&balance, &frozen))
	require.Equal(t, 9.8, balance)
	require.Zero(t, frozen)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT quota_used FROM api_keys WHERE id=$1`, f.apiKeyID).Scan(&quotaUsed))
	require.Equal(t, 0.2, quotaUsed)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT status,reservation_state,cost_amount,provider_actual_cost_usd,result_url FROM video_tasks WHERE id=$1`, f.task.ID).Scan(&status, &state, &charged, &providerCost, &resultURL))
	require.Equal(t, service.VideoStatusFailed, status)
	require.Equal(t, service.VideoReservationCaptured, state)
	require.Equal(t, 0.2, charged)
	require.Equal(t, 0.3, providerCost)
	require.NotEmpty(t, resultURL)
}

func TestVideoGatewaySettlementRollbackKeepsReservationAtomic(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	tokens := int64(100)
	_, err := f.repo.FinalizeTask(ctx, service.VideoTaskFinalization{TaskID: f.task.ID, ExpectedVersion: f.task.Version,
		Status: service.VideoStatusSucceeded, ResultURL: "https://cdn.example.test/video.mp4", UsageTotalTokens: &tokens,
		CostAmount: 0.05, ProviderActualCostUSD: 0.05, Currency: strings.Repeat("X", 9), Settlement: service.VideoSettlementCaptureActual, CompletedAt: time.Now().UTC()})
	require.Error(t, err)
	var balance, frozen, quotaUsed float64
	var status, state string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, f.userID).Scan(&balance, &frozen))
	require.Equal(t, 9.8, balance)
	require.Equal(t, 0.2, frozen)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT quota_used FROM api_keys WHERE id=$1`, f.apiKeyID).Scan(&quotaUsed))
	require.Equal(t, 0.2, quotaUsed)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT status,reservation_state FROM video_tasks WHERE id=$1`, f.task.ID).Scan(&status, &state))
	require.Equal(t, service.VideoStatusQueued, status)
	require.Equal(t, service.VideoReservationReserved, state)
	var dedup int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id=$1`, fmt.Sprintf("video:%d", f.task.ID)).Scan(&dedup))
	require.Zero(t, dedup)
}

func TestVideoGatewaySettlementRollsBackWhenReservedAPIKeyIsSoftDeleted(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `UPDATE api_keys SET deleted_at=NOW() WHERE id=$1 RETURNING id`, f.apiKeyID).Scan(&f.apiKeyID))
	tokens := int64(100)
	_, err := f.repo.FinalizeTask(ctx, service.VideoTaskFinalization{TaskID: f.task.ID, ExpectedVersion: f.task.Version,
		Status: service.VideoStatusSucceeded, ResultURL: "https://cdn.example.test/video.mp4", UsageTotalTokens: &tokens,
		CostAmount: 0.05, ProviderActualCostUSD: 0.05, Currency: "USD", Settlement: service.VideoSettlementCaptureActual, CompletedAt: time.Now().UTC()})
	require.ErrorContains(t, err, "api key reservation")

	var balance, frozen float64
	var status, reservationState string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, f.userID).Scan(&balance, &frozen))
	require.Equal(t, 9.8, balance)
	require.Equal(t, 0.2, frozen)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT status,reservation_state FROM video_tasks WHERE id=$1`, f.task.ID).Scan(&status, &reservationState))
	require.Equal(t, service.VideoStatusQueued, status)
	require.Equal(t, service.VideoReservationReserved, reservationState)
	var dedup, usageLogs int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id=$1`, fmt.Sprintf("video:%d", f.task.ID)).Scan(&dedup))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_usage_logs WHERE video_task_id=$1`, f.task.ID).Scan(&usageLogs))
	require.Zero(t, dedup)
	require.Zero(t, usageLogs)
}

func TestVideoGatewayScopeAndDispatchCancelConcurrency(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	_, err := f.repo.GetTaskForScope(ctx, f.task.ID, service.VideoTaskScope{UserID: f.userID + 1, APIKeyID: f.apiKeyID, GroupID: f.groupID})
	require.ErrorIs(t, err, service.ErrVideoTaskNotFound)
	start := make(chan struct{})
	var wg sync.WaitGroup
	var dispatchOK bool
	var dispatchErr, cancelErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		dispatchOK, dispatchErr = f.repo.BeginRealDispatch(ctx, f.task.ID, f.task.Version)
	}()
	go func() {
		defer wg.Done()
		<-start
		_, cancelErr = f.repo.CancelTaskForScope(ctx, f.task.ID, service.VideoTaskScope{UserID: f.userID, APIKeyID: f.apiKeyID, GroupID: f.groupID})
	}()
	close(start)
	wg.Wait()
	require.NoError(t, dispatchErr)
	if dispatchOK {
		require.ErrorIs(t, cancelErr, service.ErrVideoCancelConflict)
	} else {
		require.NoError(t, cancelErr)
	}
}

func TestVideoGatewayProviderGroupIsolationAndGlobalGate(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	other := newVideoIntegrationFixture(t, 10, 1, 1, "standard", false)
	_, err := f.repo.GetVideoProvider(ctx, other.providerID, f.groupID)
	require.ErrorIs(t, err, service.ErrVideoProviderNotFound)
	second := newVideoIntegrationTask(f, "second", fmt.Sprintf("global-second-%d", time.Now().UnixNano()))
	require.NoError(t, f.repo.ReserveAndCreateTask(ctx, second, 0.2))
	ok, err := f.repo.BeginRealDispatch(ctx, f.task.ID, f.task.Version)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = f.repo.BeginRealDispatch(ctx, second.ID, second.Version)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestVideoGatewayDispatchRequiresProviderGrantWithoutMutation(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	_, err := integrationDB.ExecContext(ctx, `UPDATE video_provider_accounts SET tiny_real_authorized_at=NULL,tiny_real_authorized_by=NULL WHERE id=$1`, f.providerID)
	require.NoError(t, err)
	ok, err := f.repo.BeginRealDispatch(ctx, f.task.ID, f.task.Version)
	require.NoError(t, err)
	require.False(t, ok)
	var dispatchCount, globalCount int64
	var status, dispatchState string
	var consumedAt sql.NullTime
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT status,dispatch_state,real_dispatch_count FROM video_tasks WHERE id=$1`, f.task.ID).Scan(&status, &dispatchState, &dispatchCount))
	require.Equal(t, service.VideoStatusQueued, status)
	require.Equal(t, "pending", dispatchState)
	require.Zero(t, dispatchCount)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT tiny_real_consumed_at FROM video_provider_accounts WHERE id=$1`, f.providerID).Scan(&consumedAt))
	require.False(t, consumedAt.Valid)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_single_smoke_consumptions WHERE video_task_id=$1`, f.task.ID).Scan(&globalCount))
	require.Zero(t, globalCount)
}

func TestVideoGatewayConcurrentDispatchConsumesOneProviderGrant(t *testing.T) {
	ctx := context.Background()
	f := newVideoIntegrationFixture(t, 10, 1, 1, "standard", true)
	second := newVideoIntegrationTask(f, "concurrent", fmt.Sprintf("concurrent-%d", time.Now().UnixNano()))
	require.NoError(t, f.repo.ReserveAndCreateTask(ctx, second, 0.2))
	start := make(chan struct{})
	results := make(chan bool, 2)
	errorsFound := make(chan error, 2)
	var wg sync.WaitGroup
	for _, task := range []*service.VideoTask{f.task, second} {
		wg.Add(1)
		go func(candidate *service.VideoTask) {
			defer wg.Done()
			<-start
			ok, err := f.repo.BeginRealDispatch(ctx, candidate.ID, candidate.Version)
			results <- ok
			errorsFound <- err
		}(task)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		require.NoError(t, err)
	}
	successes := 0
	for ok := range results {
		if ok {
			successes++
		}
	}
	require.Equal(t, 1, successes)
	var totalDispatches, globalCount int64
	var consumedAt sql.NullTime
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COALESCE(SUM(real_dispatch_count),0) FROM video_tasks WHERE id IN ($1,$2)`, f.task.ID, second.ID).Scan(&totalDispatches))
	require.Equal(t, int64(1), totalDispatches)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT tiny_real_consumed_at FROM video_provider_accounts WHERE id=$1`, f.providerID).Scan(&consumedAt))
	require.True(t, consumedAt.Valid)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_single_smoke_consumptions WHERE video_task_id IN ($1,$2)`, f.task.ID, second.ID).Scan(&globalCount))
	require.Equal(t, int64(1), globalCount)
}

func requirePostgresUniqueViolation(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var pqErr *pq.Error
	require.True(t, errors.As(err, &pqErr), "expected PostgreSQL error, got %T", err)
	require.Equal(t, pq.ErrorCode("23505"), pqErr.Code)
}
