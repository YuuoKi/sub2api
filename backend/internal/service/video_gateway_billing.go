package service

import (
	"context"
	"log/slog"
)

// VideoBudgetGuard is the VA1 per-call budget gate + post-success charge hook for
// the video gateway. It is intentionally a narrow interface so the gateway can
// enforce a budget without depending on the concrete balance/credits backend.
//
// Phase-1 scope (this change): the gate LOGIC and its call sites are wired and
// unit-tested with a mock. Injecting the real per-user-balance backend and a
// non-zero cost rate into DI — i.e. real end-to-end deduction — is deferred to
// phase 2. When no guard is injected (production default for now) the gateway
// behaves exactly as before (no gate, no deduction).
type VideoBudgetGuard interface {
	// CheckBudget is the FAIL-CLOSED pre-call gate. It MUST return a non-nil error
	// when the subject cannot afford estimatedCost OR when affordability cannot be
	// determined; the gateway rejects the create (no row persisted, no provider
	// dispatch) on ANY error.
	CheckBudget(ctx context.Context, userID int64, estimatedCost float64) error
	// Charge deducts the actual cost after a generation reaches terminal success.
	// An error is logged but never rolls back an already-delivered video.
	Charge(ctx context.Context, userID int64, cost float64, taskID int64) error
}

// SetBudgetGuard wires the VA1 budget guard into the service. Passing nil disables
// the gate (current production default until phase-2 real billing).
func (s *VideoGatewayService) SetBudgetGuard(g VideoBudgetGuard) {
	s.budget = g
}

// estimateVideoCost estimates a task's monetary cost for the budget gate. Seedance
// returns no cost field, so it is duration × the configured per-second rate. With
// the default rate of 0 the estimate is 0 (gate inert) until a real rate is set.
func (s *VideoGatewayService) estimateVideoCost(task *VideoTask) float64 {
	if task == nil {
		return 0
	}
	rate := 0.0
	if s.cfg != nil {
		rate = s.cfg.VideoGateway.CostPerSecond
	}
	if rate < 0 {
		rate = 0
	}
	return rate * float64(task.Duration)
}

// chargeForVideo deducts the cost of a delivered video on terminal success. It
// charges the SAME duration-based estimate the pre-call gate authorized, so VA1 is
// internally consistent — a user is never charged more than was gated. Reconciling
// against a real provider-reported cost (task.CostEstimate) is phase-2 scope.
// Billing errors are logged, never fatal (the video is already delivered).
func (s *VideoGatewayService) chargeForVideo(ctx context.Context, task *VideoTask) {
	if s.budget == nil || task == nil {
		return
	}
	if err := s.budget.Charge(ctx, task.CreatedBy, s.estimateVideoCost(task), task.ID); err != nil {
		slog.Warn("video_gateway: post-success charge failed", "task_id", task.ID, "error", err)
	}
}
