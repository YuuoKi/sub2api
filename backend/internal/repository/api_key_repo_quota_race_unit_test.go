package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepositoryUpdateDoesNotRewriteQuotaUsedUnderConcurrentIncrement(t *testing.T) {
	repo, client := newAPIKeyRepoSQLiteNamed(t, "api_key_quota_race")
	ctx := context.Background()

	user := mustCreateAPIKeyRepoUser(t, ctx, client, "quota-race@example.com")
	key := &service.APIKey{
		UserID: user.ID,
		Name:   "race-key",
		Key:    "sk-race-quota-1",
		Status: service.StatusActive,
		Quota:  1000,
	}
	require.NoError(t, repo.Create(ctx, key))

	const n = 40
	const amount = 1.0
	var wg sync.WaitGroup
	var okCount atomic.Int64
	errCh := make(chan error, n*2)

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := repo.IncrementQuotaUsed(ctx, key.ID, amount); err != nil {
				errCh <- err
				return
			}
			okCount.Add(1)
		}()
	}

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			snapshot, err := repo.GetByID(ctx, key.ID)
			if err != nil {
				errCh <- err
				return
			}
			snapshot.Name = fmt.Sprintf("race-key-%d", i)
			snapshot.QuotaUsed = 0 // stale; must be ignored
			if err := repo.Update(ctx, snapshot); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	got, err := repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	require.InDelta(t, float64(okCount.Load())*amount, got.QuotaUsed, 0.0001)
	require.NotEqual(t, "race-key", got.Name)
}

func TestAPIKeyRepositoryResetRateLimitUsageDoesNotTouchQuotaUsed(t *testing.T) {
	repo, client := newAPIKeyRepoSQLiteNamed(t, "api_key_reset_usage")
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "reset-usage@example.com")
	key := &service.APIKey{
		UserID:  user.ID,
		Name:    "reset-key",
		Key:     "sk-reset-usage-1",
		Status:  service.StatusActive,
		Quota:   100,
		Usage5h: 9,
		Usage1d: 8,
		Usage7d: 7,
	}
	require.NoError(t, repo.Create(ctx, key))
	_, err := repo.IncrementQuotaUsed(ctx, key.ID, 12)
	require.NoError(t, err)

	require.NoError(t, repo.ResetRateLimitUsage(ctx, key.ID))
	got, err := repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	require.InDelta(t, 12.0, got.QuotaUsed, 0.0001)
	require.Zero(t, got.Usage5h)
	require.Zero(t, got.Usage1d)
	require.Zero(t, got.Usage7d)
	require.Nil(t, got.Window5hStart)
	require.Nil(t, got.Window1dStart)
	require.Nil(t, got.Window7dStart)
}

func TestAPIKeyRepositoryResetQuotaUsedDoesNotTouchRateLimitUsage(t *testing.T) {
	repo, client := newAPIKeyRepoSQLiteNamed(t, "api_key_reset_quota")
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "reset-quota@example.com")
	key := &service.APIKey{
		UserID:  user.ID,
		Name:    "reset-quota-key",
		Key:     "sk-reset-quota-1",
		Status:  service.StatusActive,
		Quota:   100,
		Usage5h: 9,
		Usage1d: 8,
		Usage7d: 7,
	}
	require.NoError(t, repo.Create(ctx, key))
	_, err := repo.IncrementQuotaUsed(ctx, key.ID, 12)
	require.NoError(t, err)
	_, err = client.APIKey.UpdateOneID(key.ID).
		SetUsage5h(9).
		SetUsage1d(9).
		SetUsage7d(9).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.ResetQuotaUsed(ctx, key.ID))
	got, err := repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	require.Zero(t, got.QuotaUsed)
	require.InDelta(t, 9.0, got.Usage5h, 0.0001)
	require.InDelta(t, 9.0, got.Usage1d, 0.0001)
	require.InDelta(t, 9.0, got.Usage7d, 0.0001)
}
