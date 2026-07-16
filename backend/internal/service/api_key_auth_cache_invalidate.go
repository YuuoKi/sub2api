package service

import (
	"context"
	"errors"
)

// InvalidateAuthCacheByKey 清除指定 API Key 的认证缓存
func (s *APIKeyService) InvalidateAuthCacheByKey(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	cacheKey := s.authCacheKey(key)
	return s.deleteAuthCache(ctx, cacheKey)
}

// InvalidateAuthCacheByUserID 清除用户相关的 API Key 认证缓存
func (s *APIKeyService) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return nil
	}
	keys, err := s.apiKeyRepo.ListKeysByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.deleteAuthCacheByKeys(ctx, keys)
}

// InvalidateAuthCacheByGroupID 清除分组相关的 API Key 认证缓存
func (s *APIKeyService) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) error {
	if groupID <= 0 {
		return nil
	}
	keys, err := s.apiKeyRepo.ListKeysByGroupID(ctx, groupID)
	if err != nil {
		return err
	}
	return s.deleteAuthCacheByKeys(ctx, keys)
}

func (s *APIKeyService) deleteAuthCacheByKeys(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	var errs []error
	for _, key := range keys {
		if key == "" {
			continue
		}
		if err := s.deleteAuthCache(ctx, s.authCacheKey(key)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
