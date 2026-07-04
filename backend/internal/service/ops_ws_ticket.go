package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func IssueOpsWSTicket(ctx context.Context, cache OpsWSTicketCache, userID int64) (string, time.Duration, error) {
	if cache == nil {
		return "", 0, ErrServiceUnavailable
	}
	if userID <= 0 {
		return "", 0, fmt.Errorf("invalid user id")
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", 0, fmt.Errorf("generate ops ws ticket: %w", err)
	}
	ticket := hex.EncodeToString(buf)
	if err := cache.StoreTicket(ctx, ticket, userID, OpsWSTicketTTL); err != nil {
		return "", 0, err
	}
	return ticket, OpsWSTicketTTL, nil
}

func ConsumeOpsWSTicket(ctx context.Context, cache OpsWSTicketCache, ticket string) (int64, error) {
	if cache == nil || ticket == "" {
		return 0, ErrOpsWSTicketInvalid
	}
	return cache.ConsumeTicket(ctx, ticket)
}
