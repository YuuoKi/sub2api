package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/shopspring/decimal"
)

func TestInternalRealReservationIdempotentAndNoOverspend(t *testing.T) {
	policy := &ProviderRealAccessPolicy{
		ID:               1,
		Name:             "default",
		Enabled:          true,
		GlobalKillSwitch: false,
		AllowMember:      true,
		ImageDailyCNY:    decimal.NewFromInt(20),
		VideoDailyCNY:    decimal.NewFromInt(20),
		MonthlyCNY:       decimal.NewFromInt(20),
	}
	repo := NewMemoryProviderRealAccessPolicyRepo(policy)
	ctx := context.Background()

	// Idempotent same OperationID.
	first := ProviderRealAccessReservation{
		OperationID: "video:1001",
		UserID:      7,
		Kind:        "video",
		ReservedCNY: decimal.NewFromInt(10),
	}
	if err := repo.ReserveInTx(ctx, first); err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if err := repo.ReserveInTx(ctx, first); err != nil {
		t.Fatalf("idempotent reserve: %v", err)
	}

	var accepted atomic.Int64
	var rejected atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := repo.ReserveInTx(ctx, ProviderRealAccessReservation{
				OperationID: "video:race:" + decimal.NewFromInt(int64(i)).String(),
				UserID:      7,
				Kind:        "video",
				ReservedCNY: decimal.NewFromInt(10),
			})
			if err == nil {
				accepted.Add(1)
			} else {
				rejected.Add(1)
			}
		}(i)
	}
	wg.Wait()
	// Already reserved 10; limit 20 => at most one more 10-CNY reservation.
	if accepted.Load() != 1 {
		t.Fatalf("accepted=%d, want 1 additional reservation at limit", accepted.Load())
	}
	if rejected.Load() < 30 {
		t.Fatalf("rejected=%d, want most concurrent attempts rejected", rejected.Load())
	}
	used, err := repo.SumReservedCNY(ctx, 7, "video", policy.UpdatedAt)
	if err != nil {
		t.Fatalf("sum: %v", err)
	}
	if used.GreaterThan(decimal.NewFromInt(20)) {
		t.Fatalf("used=%s over limit 20", used.String())
	}
}
