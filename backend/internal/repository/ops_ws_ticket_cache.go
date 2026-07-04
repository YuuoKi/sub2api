package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const opsWSTicketKeyPrefix = "ops_ws_ticket:"

type opsWSTicketCache struct {
	rdb *redis.Client
}

func NewOpsWSTicketCache(rdb *redis.Client) service.OpsWSTicketCache {
	return &opsWSTicketCache{rdb: rdb}
}

func opsWSTicketKey(ticket string) string {
	return opsWSTicketKeyPrefix + strings.TrimSpace(ticket)
}

func (c *opsWSTicketCache) StoreTicket(ctx context.Context, ticket string, userID int64, ttl time.Duration) error {
	if c == nil || c.rdb == nil || strings.TrimSpace(ticket) == "" || userID <= 0 {
		return fmt.Errorf("invalid ops ws ticket store request")
	}
	if ttl <= 0 {
		ttl = service.OpsWSTicketTTL
	}
	return c.rdb.Set(ctx, opsWSTicketKey(ticket), strconv.FormatInt(userID, 10), ttl).Err()
}

func (c *opsWSTicketCache) ConsumeTicket(ctx context.Context, ticket string) (int64, error) {
	if c == nil || c.rdb == nil || strings.TrimSpace(ticket) == "" {
		return 0, service.ErrOpsWSTicketInvalid
	}
	key := opsWSTicketKey(ticket)
	userIDRaw, err := c.rdb.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, service.ErrOpsWSTicketInvalid
		}
		return 0, err
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(userIDRaw), 10, 64)
	if err != nil || userID <= 0 {
		return 0, service.ErrOpsWSTicketInvalid
	}
	return userID, nil
}
