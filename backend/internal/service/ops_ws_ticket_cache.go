package service

import (
	"context"
	"errors"
	"time"
)

var ErrOpsWSTicketInvalid = errors.New("ops websocket ticket invalid or expired")

// OpsWSTicketCache stores single-use short-lived tickets for admin ops WebSocket auth.
type OpsWSTicketCache interface {
	StoreTicket(ctx context.Context, ticket string, userID int64, ttl time.Duration) error
	ConsumeTicket(ctx context.Context, ticket string) (int64, error)
}

const OpsWSTicketTTL = 60 * time.Second
