package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func invalidateVideoCaches(ctx context.Context, auth VideoAuthCacheInvalidator, billing VideoBillingCacheInvalidator, userID, apiKeyID int64) {
	if auth != nil {
		auth.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if billing == nil {
		return
	}
	if err := billing.InvalidateUserBalance(ctx, userID); err != nil {
		logger.L().Error("video_gateway.balance_cache_invalidation_failed", zap.String("component", "service.video_gateway"), zap.Int64("user_id", userID), zap.Error(err))
	}
	if err := billing.InvalidateAPIKeyRateLimit(ctx, apiKeyID); err != nil {
		logger.L().Error("video_gateway.rate_cache_invalidation_failed", zap.String("component", "service.video_gateway"), zap.Int64("api_key_id", apiKeyID), zap.Error(err))
	}
}
