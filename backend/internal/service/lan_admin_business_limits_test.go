//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLANAdminGroupBusinessLimitsAlwaysRemainUnlimited(t *testing.T) {
	settings := NewSettingService(nil, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})
	daily, weekly, monthly := 1.0, 2.0, 3.0
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo, settingService: settings}

	created, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
		Name:            "production",
		Platform:        PlatformOpenAI,
		RateMultiplier:  1,
		DailyLimitUSD:   &daily,
		WeeklyLimitUSD:  &weekly,
		MonthlyLimitUSD: &monthly,
		RPMLimit:        60,
	})
	require.NoError(t, err)
	require.Nil(t, created.DailyLimitUSD)
	require.Nil(t, created.WeeklyLimitUSD)
	require.Nil(t, created.MonthlyLimitUSD)
	require.Zero(t, created.RPMLimit)

	repo.getByID = &Group{ID: 1, Name: "production", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive, DailyLimitUSD: &daily, WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &monthly, RPMLimit: 10}
	newRPM := 120
	updated, err := svc.UpdateGroup(context.Background(), 1, &UpdateGroupInput{
		DailyLimitUSD:   &daily,
		WeeklyLimitUSD:  &weekly,
		MonthlyLimitUSD: &monthly,
		RPMLimit:        &newRPM,
	})
	require.NoError(t, err)
	require.Nil(t, updated.DailyLimitUSD)
	require.Nil(t, updated.WeeklyLimitUSD)
	require.Nil(t, updated.MonthlyLimitUSD)
	require.Zero(t, updated.RPMLimit)
}

func TestLANAdminRejectsSubscriptionGroups(t *testing.T) {
	settings := NewSettingService(nil, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo, settingService: settings}

	_, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
		Name:             "legacy-subscription",
		Platform:         PlatformOpenAI,
		RateMultiplier:   1,
		SubscriptionType: SubscriptionTypeSubscription,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "subscription groups are disabled")
	require.Nil(t, repo.created)

	repo.getByID = &Group{ID: 1, Name: "production", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard}
	_, err = svc.UpdateGroup(context.Background(), 1, &UpdateGroupInput{SubscriptionType: SubscriptionTypeSubscription})
	require.Error(t, err)
	require.ErrorContains(t, err, "subscription groups are disabled")
	require.Nil(t, repo.updated)
}

func TestLANAdminBillingEligibilityDoesNotRequireBalanceOrLocalLimits(t *testing.T) {
	svc := &BillingCacheService{cfg: &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin}}
	err := svc.CheckBillingEligibility(
		context.Background(),
		&User{ID: 42, Balance: 0, RPMLimit: 10},
		&APIKey{ID: 7, Quota: 1, RateLimit1d: 1},
		&Group{ID: 9, RPMLimit: 10, DailyLimitUSD: float64Pointer(1)},
		nil,
		PlatformOpenAI,
	)
	require.NoError(t, err)
}

func float64Pointer(value float64) *float64 { return &value }
