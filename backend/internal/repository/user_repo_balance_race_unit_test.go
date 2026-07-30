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

func TestUserRepositoryUpdateDoesNotRewriteBalanceUnderConcurrentDeduct(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()

	const startBalance = 1000.0
	const deductN = 50
	const deductAmount = 1.0

	err := repo.Create(ctx, &service.User{
		Email:        "race-balance@example.com",
		Username:     "race-balance",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      startBalance,
	})
	require.NoError(t, err)

	user, err := repo.GetByEmail(ctx, "race-balance@example.com")
	require.NoError(t, err)

	var wg sync.WaitGroup
	var deductOK atomic.Int64
	errCh := make(chan error, deductN*2)

	wg.Add(deductN)
	for i := 0; i < deductN; i++ {
		go func() {
			defer wg.Done()
			if err := repo.DeductBalance(ctx, user.ID, deductAmount); err != nil {
				errCh <- err
				return
			}
			deductOK.Add(1)
		}()
	}

	wg.Add(deductN)
	for i := 0; i < deductN; i++ {
		go func(i int) {
			defer wg.Done()
			snapshot, err := repo.GetByID(ctx, user.ID)
			if err != nil {
				errCh <- err
				return
			}
			// Stale snapshot mutates a non-billing field and calls Update.
			// Balance must not be rewritten from this snapshot.
			snapshot.Username = fmt.Sprintf("race-balance-%d", i)
			snapshot.Balance = startBalance // stale; must be ignored by Update
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

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	want := startBalance - float64(deductOK.Load())*deductAmount
	require.InDelta(t, want, got.Balance, 0.0001, "concurrent profile Update must not rewind balance")
	require.NotEqual(t, "race-balance", got.Username)
}

func TestUserRepositoryUpdatePreservesBalanceWhenChangingEmail(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()

	err := repo.Create(ctx, &service.User{
		Email:        "email-balance@example.com",
		Username:     "email-balance",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      42.5,
	})
	require.NoError(t, err)

	user, err := repo.GetByEmail(ctx, "email-balance@example.com")
	require.NoError(t, err)
	// Create does not seed total_recharged; establish it via atomic add.
	require.NoError(t, repo.UpdateBalance(ctx, user.ID, 100))
	user, err = repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 142.5, user.Balance, 0.0001)
	require.InDelta(t, 100.0, user.TotalRecharged, 0.0001)

	require.NoError(t, repo.DeductBalance(ctx, user.ID, 2.5))

	snapshot, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	snapshot.Email = "email-balance-updated@example.com"
	snapshot.Balance = 999 // stale; must be ignored
	snapshot.TotalRecharged = 0
	require.NoError(t, repo.Update(ctx, snapshot))

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "email-balance-updated@example.com", got.Email)
	require.InDelta(t, 140.0, got.Balance, 0.0001)
	require.InDelta(t, 100.0, got.TotalRecharged, 0.0001)
}

func TestUserRepositorySetBalanceDoesNotTouchTotalRecharged(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()

	err := repo.Create(ctx, &service.User{
		Email:        "set-balance@example.com",
		Username:     "set-balance",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      10,
	})
	require.NoError(t, err)
	user, err := repo.GetByEmail(ctx, "set-balance@example.com")
	require.NoError(t, err)
	require.NoError(t, repo.UpdateBalance(ctx, user.ID, 55))

	require.NoError(t, repo.SetBalance(ctx, user.ID, 3))
	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 3.0, got.Balance, 0.0001)
	require.InDelta(t, 55.0, got.TotalRecharged, 0.0001)
}
