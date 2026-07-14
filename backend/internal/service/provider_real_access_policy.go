package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// ProviderRealAccessPolicy constrains internal_real creates (not review session).
type ProviderRealAccessPolicy struct {
	ID               int64
	Name             string
	Enabled          bool
	GlobalKillSwitch bool
	AllowMember      bool
	AllowGroup       bool
	ImageDailyCNY    decimal.Decimal
	VideoDailyCNY    decimal.Decimal
	MonthlyCNY       decimal.Decimal
	EnabledAt        *time.Time
	DisabledAt       *time.Time
	AuditActorID     *int64
	AuditActorEmail  string
	UpdatedAt        time.Time
}

type ProviderRealAccessReservation struct {
	OperationID string
	UserID      int64
	Kind        string // image | video
	ReservedCNY decimal.Decimal
	Status      string // reserved | settled | released
	SettledCNY  *decimal.Decimal
	PolicyID    *int64
}

type ProviderRealAccessPolicyRepository interface {
	GetPolicy(ctx context.Context, name string) (*ProviderRealAccessPolicy, error)
	SavePolicy(ctx context.Context, policy *ProviderRealAccessPolicy) error
	ReserveInTx(ctx context.Context, reservation ProviderRealAccessReservation) error
	Settle(ctx context.Context, operationID string, settledCNY decimal.Decimal) error
	Release(ctx context.Context, operationID string) error
	SumReservedCNY(ctx context.Context, userID int64, kind string, since time.Time) (decimal.Decimal, error)
}

// MemoryProviderRealAccessPolicyRepo is used by unit/concurrency tests.
type MemoryProviderRealAccessPolicyRepo struct {
	mu           sync.Mutex
	policy       *ProviderRealAccessPolicy
	reservations map[string]ProviderRealAccessReservation
}

func NewMemoryProviderRealAccessPolicyRepo(policy *ProviderRealAccessPolicy) *MemoryProviderRealAccessPolicyRepo {
	if policy == nil {
		policy = &ProviderRealAccessPolicy{
			ID:               1,
			Name:             "default",
			Enabled:          true,
			GlobalKillSwitch: false,
			AllowMember:      true,
			ImageDailyCNY:    decimal.NewFromInt(100),
			VideoDailyCNY:    decimal.NewFromInt(100),
			MonthlyCNY:       decimal.NewFromInt(1000),
		}
	}
	return &MemoryProviderRealAccessPolicyRepo{
		policy:       policy,
		reservations: make(map[string]ProviderRealAccessReservation),
	}
}

func (r *MemoryProviderRealAccessPolicyRepo) GetPolicy(_ context.Context, name string) (*ProviderRealAccessPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.policy == nil {
		return nil, fmt.Errorf("policy not found")
	}
	if name != "" && r.policy.Name != name {
		return nil, fmt.Errorf("policy not found")
	}
	cp := *r.policy
	return &cp, nil
}

func (r *MemoryProviderRealAccessPolicyRepo) SavePolicy(_ context.Context, policy *ProviderRealAccessPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if policy == nil {
		return fmt.Errorf("policy required")
	}
	cp := *policy
	r.policy = &cp
	return nil
}

func (r *MemoryProviderRealAccessPolicyRepo) ReserveInTx(_ context.Context, reservation ProviderRealAccessReservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.policy == nil || !r.policy.Enabled || r.policy.GlobalKillSwitch || !r.policy.AllowMember {
		return ErrInternalRealPolicyDenied
	}
	opID := strings.TrimSpace(reservation.OperationID)
	if opID == "" {
		return fmt.Errorf("operation id required")
	}
	if existing, ok := r.reservations[opID]; ok {
		if existing.ReservedCNY.Equal(reservation.ReservedCNY) && existing.Kind == reservation.Kind && existing.UserID == reservation.UserID {
			return nil
		}
		return fmt.Errorf("INTERNAL_REAL_IDEMPOTENCY_MISMATCH")
	}
	limit := r.policy.MonthlyCNY
	switch reservation.Kind {
	case "image":
		if r.policy.ImageDailyCNY.IsPositive() && r.policy.ImageDailyCNY.LessThan(limit) {
			limit = r.policy.ImageDailyCNY
		}
	case "video":
		if r.policy.VideoDailyCNY.IsPositive() && r.policy.VideoDailyCNY.LessThan(limit) {
			limit = r.policy.VideoDailyCNY
		}
	}
	used := decimal.Zero
	for _, item := range r.reservations {
		if item.UserID != reservation.UserID || item.Status != "reserved" {
			continue
		}
		if reservation.Kind != "" && item.Kind != reservation.Kind {
			continue
		}
		used = used.Add(item.ReservedCNY)
	}
	if used.Add(reservation.ReservedCNY).GreaterThan(limit) {
		return ErrInternalRealBudgetExceeded
	}
	reservation.Status = "reserved"
	r.reservations[opID] = reservation
	return nil
}

func (r *MemoryProviderRealAccessPolicyRepo) Settle(_ context.Context, operationID string, settledCNY decimal.Decimal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.reservations[operationID]
	if !ok {
		return nil
	}
	item.Status = "settled"
	item.SettledCNY = &settledCNY
	r.reservations[operationID] = item
	return nil
}

func (r *MemoryProviderRealAccessPolicyRepo) Release(_ context.Context, operationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.reservations[operationID]
	if !ok {
		return nil
	}
	item.Status = "released"
	r.reservations[operationID] = item
	return nil
}

func (r *MemoryProviderRealAccessPolicyRepo) SumReservedCNY(_ context.Context, userID int64, kind string, _ time.Time) (decimal.Decimal, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sum := decimal.Zero
	for _, item := range r.reservations {
		if item.UserID != userID || item.Status != "reserved" {
			continue
		}
		if kind != "" && item.Kind != kind {
			continue
		}
		sum = sum.Add(item.ReservedCNY)
	}
	return sum, nil
}
