//go:build unit

package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type stubOpsWSTicketCache struct {
	mu      sync.Mutex
	tickets map[string]int64
}

func newStubOpsWSTicketCache() *stubOpsWSTicketCache {
	return &stubOpsWSTicketCache{tickets: make(map[string]int64)}
}

func (s *stubOpsWSTicketCache) StoreTicket(_ context.Context, ticket string, userID int64, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tickets[ticket] = userID
	return nil
}

func (s *stubOpsWSTicketCache) ConsumeTicket(_ context.Context, ticket string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	userID, ok := s.tickets[ticket]
	if !ok {
		return 0, service.ErrOpsWSTicketInvalid
	}
	delete(s.tickets, ticket)
	return userID, nil
}
