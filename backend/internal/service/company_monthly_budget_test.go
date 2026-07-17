package service

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type monthlyBudgetSettingRepoStub struct {
	value       string
	setKey      string
	setValue    string
	deletedKey  string
	getValueErr error
}

func (*monthlyBudgetSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}
func (r *monthlyBudgetSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return r.value, r.getValueErr
}
func (r *monthlyBudgetSettingRepoStub) Set(_ context.Context, key, value string) error {
	r.setKey, r.setValue = key, value
	return nil
}
func (*monthlyBudgetSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}
func (*monthlyBudgetSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}
func (*monthlyBudgetSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}
func (r *monthlyBudgetSettingRepoStub) Delete(_ context.Context, key string) error {
	r.deletedKey = key
	return nil
}

func TestSettingServiceCompanyMonthlyBudgetLifecycle(t *testing.T) {
	t.Run("reads persisted finite budget", func(t *testing.T) {
		repo := &monthlyBudgetSettingRepoStub{value: " 1234.56 "}
		svc := NewSettingService(repo, nil)
		require.Equal(t, 1234.56, svc.GetCompanyMonthlyBudgetCNY(context.Background()))
	})

	t.Run("invalid persisted value fails closed to unset", func(t *testing.T) {
		for _, raw := range []string{"", "bad", "-1", "NaN", "+Inf"} {
			repo := &monthlyBudgetSettingRepoStub{value: raw}
			svc := NewSettingService(repo, nil)
			require.Zero(t, svc.GetCompanyMonthlyBudgetCNY(context.Background()), raw)
		}
	})

	t.Run("persists finite non-negative budget", func(t *testing.T) {
		repo := &monthlyBudgetSettingRepoStub{}
		svc := NewSettingService(repo, nil)
		require.NoError(t, svc.SetCompanyMonthlyBudgetCNY(context.Background(), 88.125))
		require.Equal(t, SettingKeyCompanyMonthlyBudgetCNY, repo.setKey)
		require.Equal(t, "88.125", repo.setValue)
	})

	t.Run("zero clears budget", func(t *testing.T) {
		repo := &monthlyBudgetSettingRepoStub{}
		svc := NewSettingService(repo, nil)
		require.NoError(t, svc.SetCompanyMonthlyBudgetCNY(context.Background(), 0))
		require.Equal(t, SettingKeyCompanyMonthlyBudgetCNY, repo.deletedKey)
		require.Empty(t, repo.setKey)
	})

	t.Run("rejects invalid input without writing", func(t *testing.T) {
		for _, value := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
			repo := &monthlyBudgetSettingRepoStub{}
			svc := NewSettingService(repo, nil)
			require.Error(t, svc.SetCompanyMonthlyBudgetCNY(context.Background(), value))
			require.Empty(t, repo.setKey)
			require.Empty(t, repo.deletedKey)
		}
	})
}

func TestMonthlyBudgetUsagePercent(t *testing.T) {
	require.Equal(t, 50.0, MonthlyBudgetUsagePercent(500, 1000))
	require.Equal(t, 150.0, MonthlyBudgetUsagePercent(1500, 1000))
	require.Zero(t, MonthlyBudgetUsagePercent(100, 0))
	require.Zero(t, MonthlyBudgetUsagePercent(math.NaN(), 1000))
}
