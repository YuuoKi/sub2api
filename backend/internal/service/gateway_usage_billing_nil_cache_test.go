package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type nilCachePlatformQuotaRepoStub struct {
	incrCalls int
}

func (s *nilCachePlatformQuotaRepoStub) GetByUserPlatform(context.Context, int64, string) (*UserPlatformQuotaRecord, error) {
	return nil, nil
}

func (s *nilCachePlatformQuotaRepoStub) BulkInsertInitial(context.Context, []UserPlatformQuotaRecord) error {
	return nil
}

func (s *nilCachePlatformQuotaRepoStub) IncrementUsageWithReset(context.Context, int64, string, float64, time.Time) error {
	s.incrCalls++
	return nil
}

func (s *nilCachePlatformQuotaRepoStub) ListByUser(context.Context, int64) ([]UserPlatformQuotaRecord, error) {
	return nil, nil
}

func (s *nilCachePlatformQuotaRepoStub) UpsertForUser(context.Context, int64, []UserPlatformQuotaRecord) error {
	return nil
}

func (s *nilCachePlatformQuotaRepoStub) ResetExpiredWindow(context.Context, int64, string, string, time.Time) error {
	return nil
}

func (s *nilCachePlatformQuotaRepoStub) BatchSnapshotUsage(context.Context, []UserPlatformQuotaSnapshot, time.Time) error {
	return nil
}

type nilCacheUserRepoStub struct{}

func (s *nilCacheUserRepoStub) Create(context.Context, *User) error { return nil }
func (s *nilCacheUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	return &User{ID: 7, Balance: 100}, nil
}
func (s *nilCacheUserRepoStub) GetByIDIncludeDeleted(context.Context, int64) (*User, error) {
	return &User{ID: 7, Balance: 100}, nil
}
func (s *nilCacheUserRepoStub) GetByEmail(context.Context, string) (*User, error) { return nil, nil }
func (s *nilCacheUserRepoStub) GetFirstAdmin(context.Context) (*User, error)     { return nil, nil }
func (s *nilCacheUserRepoStub) Update(context.Context, *User) error              { return nil }
func (s *nilCacheUserRepoStub) Delete(context.Context, int64) error              { return nil }
func (s *nilCacheUserRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	return nil, nil
}
func (s *nilCacheUserRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	return nil, nil
}
func (s *nilCacheUserRepoStub) DeleteUserAvatar(context.Context, int64) error { return nil }
func (s *nilCacheUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *nilCacheUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *nilCacheUserRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	return nil, nil
}
func (s *nilCacheUserRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	return nil, nil
}
func (s *nilCacheUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	return nil
}
func (s *nilCacheUserRepoStub) UpdateBalance(context.Context, int64, float64) error { return nil }
func (s *nilCacheUserRepoStub) SetBalance(context.Context, int64, float64) error    { return nil }
func (s *nilCacheUserRepoStub) DeductBalance(context.Context, int64, float64) error { return nil }
func (s *nilCacheUserRepoStub) UpdateConcurrency(context.Context, int64, int) error { return nil }
func (s *nilCacheUserRepoStub) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}
func (s *nilCacheUserRepoStub) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}
func (s *nilCacheUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }
func (s *nilCacheUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *nilCacheUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (s *nilCacheUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (s *nilCacheUserRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	return nil, nil
}
func (s *nilCacheUserRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error {
	return nil
}
func (s *nilCacheUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error { return nil }
func (s *nilCacheUserRepoStub) EnableTotp(context.Context, int64) error                { return nil }
func (s *nilCacheUserRepoStub) DisableTotp(context.Context, int64) error               { return nil }

func TestPostUsageBilling_NilBillingCacheService_NoPanic(t *testing.T) {
	quotaRepo := &nilCachePlatformQuotaRepoStub{}
	require.NotPanics(t, func() {
		postUsageBilling(context.Background(), &postUsageBillingParams{
			Cost:     &CostBreakdown{ActualCost: 1.25, TotalCost: 1.25},
			User:     &User{ID: 7},
			Account:  &Account{ID: 9},
			APIKey:   &APIKey{ID: 11},
			Platform: "anthropic",
		}, &billingDeps{
			userRepo:              &nilCacheUserRepoStub{},
			billingCacheService:   nil,
			userPlatformQuotaRepo: quotaRepo,
			cfg:                   &config.Config{},
		})
	})
	require.Equal(t, 0, quotaRepo.incrCalls, "nil billing cache must skip platform quota writes")
}

func TestFinalizePostUsageBilling_NilBillingCacheService_NoPanic(t *testing.T) {
	quotaRepo := &nilCachePlatformQuotaRepoStub{}
	require.NotPanics(t, func() {
		finalizePostUsageBilling(context.Background(), &postUsageBillingParams{
			Cost:     &CostBreakdown{ActualCost: 1.25, TotalCost: 1.25},
			User:     &User{ID: 7},
			Account:  &Account{ID: 9},
			Platform: "anthropic",
		}, &billingDeps{
			billingCacheService:   nil,
			deferredService:       &DeferredService{},
			userPlatformQuotaRepo: quotaRepo,
			cfg:                   &config.Config{},
		}, nil)
	})
	require.Equal(t, 0, quotaRepo.incrCalls, "nil billing cache must skip platform quota writes")
}
