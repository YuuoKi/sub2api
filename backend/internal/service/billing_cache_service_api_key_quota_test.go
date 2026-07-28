package service

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type apiKeyQuotaEligibilityCacheStub struct {
	billingCacheWorkerStub
	balance float64
}

func (s *apiKeyQuotaEligibilityCacheStub) GetUserBalance(context.Context, int64) (float64, error) {
	return s.balance, nil
}

type apiKeyQuotaUsageLoaderStub struct {
	key *APIKey
	err error
}

func (s *apiKeyQuotaUsageLoaderStub) GetByID(context.Context, int64) (*APIKey, error) {
	return s.key, s.err
}

type apiKeyQuotaAuthCacheStub struct {
	invalidated atomic.Bool
	lastKey     atomic.Value
}

func (s *apiKeyQuotaAuthCacheStub) InvalidateAuthCacheByKey(_ context.Context, key string) error {
	s.invalidated.Store(true)
	s.lastKey.Store(key)
	return nil
}

func (s *apiKeyQuotaAuthCacheStub) InvalidateAuthCacheByUserID(context.Context, int64) error {
	return nil
}

func (s *apiKeyQuotaAuthCacheStub) InvalidateAuthCacheByGroupID(context.Context, int64) error {
	return nil
}

func TestCheckBillingEligibility_RejectsExhaustedAPIKeyQuotaFromDB(t *testing.T) {
	cache := &apiKeyQuotaEligibilityCacheStub{balance: 100}
	cfg := &config.Config{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	authCache := &apiKeyQuotaAuthCacheStub{}
	svc.authCacheInvalidator = authCache
	svc.apiKeyRepo = &apiKeyQuotaUsageLoaderStub{
		key: &APIKey{ID: 42, Key: "sk-live", Quota: 10, QuotaUsed: 10},
	}

	// Auth snapshot looks fine (stale QuotaUsed), but DB says exhausted.
	apiKey := &APIKey{ID: 42, Key: "sk-stale", Quota: 10, QuotaUsed: 1}
	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1}, apiKey, nil, nil, "")
	require.ErrorIs(t, err, ErrAPIKeyQuotaExhausted)
	require.True(t, authCache.invalidated.Load(), "exhausted key should invalidate auth cache when possible")
	require.Equal(t, "sk-live", authCache.lastKey.Load())
}

func TestCheckBillingEligibility_AllowsAPIKeyWhenDBQuotaAvailable(t *testing.T) {
	cache := &apiKeyQuotaEligibilityCacheStub{balance: 100}
	cfg := &config.Config{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	svc.apiKeyRepo = &apiKeyQuotaUsageLoaderStub{
		key: &APIKey{ID: 42, Key: "sk-live", Quota: 10, QuotaUsed: 3},
	}

	apiKey := &APIKey{ID: 42, Key: "sk-stale", Quota: 10, QuotaUsed: 3}
	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1}, apiKey, nil, nil, "")
	require.NoError(t, err)
}
