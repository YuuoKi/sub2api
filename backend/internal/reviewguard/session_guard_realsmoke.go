//go:build realsmoke

package reviewguard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Compatibility aliases and thin adapters for Form A harnesses.
// Budget/lock/atomic rules live only in session_guard.go.

type Kind = RealCreateKind

const (
	Image Kind = RealCreateImage
	Video Kind = RealCreateVideo

	MaxCNY = 60.0
)

type State struct {
	ImageAttempts int
	VideoAttempts int
	ReservedCNY   float64
}

type Guard struct {
	statePath string
	enabled   bool
	inner     *SessionGuard
}

func New(statePath string, enabled bool) *Guard {
	path := strings.TrimSpace(statePath)
	return &Guard{
		statePath: path,
		enabled:   enabled,
		inner:     NewSessionGuard(path),
	}
}

func (g *Guard) ReserveBefore(kind Kind, estimatedCNY float64, call func()) error {
	if err := g.Reserve(kind, estimatedCNY); err != nil {
		return err
	}
	call()
	return nil
}

func (g *Guard) Reserve(kind Kind, estimatedCNY float64) error {
	if g == nil || !g.enabled {
		return errors.New("REAL_REVIEW_SESSION_DISABLED: explicit review-session guard enablement is required")
	}
	if g.statePath == "" {
		return errors.New("REAL_REVIEW_STATE_PATH_REQUIRED: an explicit absolute state path is required")
	}
	cny := decimal.NewFromFloat(estimatedCNY)
	opID := fmt.Sprintf("realsmoke:%s:%d:%s", kind, time.Now().UnixNano(), cny.String())
	return g.inner.Reserve(context.Background(), RealCreateReservation{
		OperationID:    opID,
		Kind:           kind,
		ReservedCNY:    cny,
		PricingSource:  "realsmoke",
		PricingVersion: "realsmoke",
	})
}

func (g *Guard) LoadState() (State, error) {
	if g == nil || g.inner == nil {
		return State{}, errors.New("REAL_REVIEW_SESSION_DISABLED: explicit review-session guard enablement is required")
	}
	snap, err := g.inner.Snapshot(context.Background())
	if err != nil {
		return State{}, err
	}
	value, _ := snap.ReservedCNY.Float64()
	return State{
		ImageAttempts: snap.ImageUsed,
		VideoAttempts: snap.VideoUsed,
		ReservedCNY:   value,
	}, nil
}
