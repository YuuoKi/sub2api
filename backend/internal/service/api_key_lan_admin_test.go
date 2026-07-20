//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLANAdminAPIKeyRequestsForceUnlimitedBusinessLimits(t *testing.T) {
	cfg := &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin}
	expiresInDays := 30
	create := CreateAPIKeyRequest{Quota: 99, ExpiresInDays: &expiresInDays, RateLimit5h: 1, RateLimit1d: 2, RateLimit7d: 3}

	normalizeLANAdminCreateAPIKeyRequest(cfg, &create)

	require.Zero(t, create.Quota)
	require.Nil(t, create.ExpiresInDays)
	require.Zero(t, create.RateLimit5h)
	require.Zero(t, create.RateLimit1d)
	require.Zero(t, create.RateLimit7d)

	quota := 99.0
	rate5h := 1.0
	rate1d := 2.0
	rate7d := 3.0
	update := UpdateAPIKeyRequest{
		Quota:       &quota,
		ExpiresAt:   timePointer(time.Now().UTC().Add(24 * time.Hour)),
		RateLimit5h: &rate5h,
		RateLimit1d: &rate1d,
		RateLimit7d: &rate7d,
	}

	normalizeLANAdminUpdateAPIKeyRequest(cfg, &update)

	require.Equal(t, 0.0, *update.Quota)
	require.Nil(t, update.ExpiresAt)
	require.True(t, update.ClearExpiration)
	require.Equal(t, 0.0, *update.RateLimit5h)
	require.Equal(t, 0.0, *update.RateLimit1d)
	require.Equal(t, 0.0, *update.RateLimit7d)
}

func timePointer(value time.Time) *time.Time { return &value }

func TestLANAdminAPIKeyCannotBindSubscriptionGroup(t *testing.T) {
	svc := &APIKeyService{cfg: &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin}}

	allowed := svc.canUserBindGroup(
		context.Background(),
		&User{ID: 42},
		&Group{ID: 7, SubscriptionType: SubscriptionTypeSubscription},
	)

	require.False(t, allowed)
}
