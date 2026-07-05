package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// VideoBudgetGuard is the VA1 per-call budget cap gate for
// the video gateway. It is intentionally a narrow interface so the gateway can
// enforce a budget without depending on the concrete balance/credits backend.
// Real balance deduction is wired separately through UserRepository so the static
// guard stays a cap/interception primitive rather than a billing backend.
type VideoBudgetGuard interface {
	// CheckBudget is the FAIL-CLOSED pre-call gate. It MUST return a non-nil error
	// when the subject cannot afford estimatedCost OR when affordability cannot be
	// determined; the gateway rejects the create (no row persisted, no provider
	// dispatch) on ANY error.
	CheckBudget(ctx context.Context, userID int64, estimatedCost float64) error
	// Charge is retained as a legacy post-success hook for tests/telemetry.
	// StaticBudgetGuard implements it as a no-op; real balance charging must not
	// live behind this method.
	Charge(ctx context.Context, userID int64, cost float64, taskID int64) error
}

// SetBudgetGuard wires the VA1 budget guard into the service. Passing nil disables
// the pre-call cap gate; real post-delivery balance billing is wired separately.
func (s *VideoGatewayService) SetBudgetGuard(g VideoBudgetGuard) {
	s.budget = g
}

// SetBalanceBillingDependencies wires the real user-balance billing path used
// after terminal successful video tasks. Passing nil leaves that dependency inert.
func (s *VideoGatewayService) SetBalanceBillingDependencies(userRepo UserRepository, settingService *SettingService, billingCacheService *BillingCacheService) {
	s.userRepo = userRepo
	s.settingService = settingService
	s.billingCacheService = billingCacheService
}

// estimateVideoCost estimates a task's monetary cost for the budget gate. Seedance
// returns no cost field, so it is duration 脳 the configured per-second rate. With
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
	duration := task.Duration
	if duration == -1 {
		duration = 5
	}
	if rate == 0 {
		rate = s.estimateSeedanceCostPerSecond(task)
	}
	return rate * float64(duration)
}

// calculateVideoActualCost converts provider-reported usage tokens to the official
// Seedance CNY cost. Failed tasks and tasks without provider usage cost 0.
func (s *VideoGatewayService) calculateVideoActualCost(task *VideoTask) float64 {
	if task == nil || task.Status != VideoStatusSucceeded || task.UsageTotalTokens == nil || *task.UsageTotalTokens <= 0 {
		return 0
	}
	rate, _, ok := s.videoPricingCatalog().RateCNYPerMillionTokens(task)
	if !ok || rate <= 0 {
		return 0
	}
	return float64(*task.UsageTotalTokens) / videoTokensPerMillion * rate
}

func (s *VideoGatewayService) estimateSeedanceCostPerSecond(task *VideoTask) float64 {
	if task == nil || task.Provider != VideoProviderSeedance {
		return 0
	}
	model := normalizeVideoPricingModel(task.Model)
	resolution := strings.ToLower(strings.TrimSpace(task.Resolution))
	if resolution == "" {
		resolution = "720p"
	}
	var base float64
	var noVideoTokenRate float64
	var actualTokenRate float64
	if strings.Contains(model, "seedance-2-0-fast") {
		noVideoTokenRate = 37
		actualTokenRate = 37
		if task.HasVideoInput {
			actualTokenRate = 22
		}
		switch resolution {
		case "480p":
			base = 0.4
		case "720p", "1080p":
			base = 0.8
		}
	} else if strings.Contains(model, "seedance-2-0") {
		noVideoTokenRate = 46
		actualTokenRate = 46
		if task.HasVideoInput {
			actualTokenRate = 28
		}
		switch resolution {
		case "480p":
			base = 0.5
		case "720p":
			base = 1.0
		case "1080p":
			base = 2.5
		}
	}
	if base <= 0 {
		return 0
	}
	if noVideoTokenRate > 0 && actualTokenRate > 0 && actualTokenRate != noVideoTokenRate {
		base *= actualTokenRate / noVideoTokenRate
	}
	return base
}

func (s *VideoGatewayService) chargeableVideoCost(task *VideoTask) float64 {
	if task == nil || task.Status != VideoStatusSucceeded {
		return 0
	}
	if task.Provider == VideoProviderSeedance {
		return s.calculateVideoActualCost(task)
	}
	if task.CostEstimate > 0 {
		return task.CostEstimate
	}
	return s.estimateVideoCost(task)
}

// chargeForVideo deducts the USD user balance for a delivered video exactly once.
// Seedance provider-usage prices are stored in CNY and converted through the
// configured usd_cny_rate (default 7.20) before deducting users.balance.
// Billing errors are logged, never fatal (the video is already delivered).
func (s *VideoGatewayService) chargeForVideo(ctx context.Context, task *VideoTask) {
	if task == nil || task.Status != VideoStatusSucceeded {
		return
	}
	cost := s.chargeableVideoCost(task)
	if cost <= 0 {
		return
	}

	claimedAt, claimed, err := s.repo.ClaimVideoBalanceCharge(ctx, task.ID)
	if err != nil {
		slog.Warn("video_gateway: claim balance charge failed", "task_id", task.ID, "error", err)
		return
	}
	if !claimed {
		return
	}

	if s.userRepo != nil {
		rate := DefaultUSDCNYRate
		if s.settingService != nil {
			rate = s.settingService.GetUSDCNYRate(ctx)
		}
		amountUSD := ConvertBillingAmount(cost, task.Currency, BillingCurrencyUSD, rate)
		if amountUSD > 0 {
			if err := s.userRepo.DeductBalance(ctx, task.CreatedBy, amountUSD); err != nil {
				slog.Warn("video_gateway: user balance deduction failed", "task_id", task.ID, "user_id", task.CreatedBy, "error", err)
				s.releaseVideoBalanceClaim(ctx, task.ID, claimedAt)
				return
			}
			if s.billingCacheService != nil {
				s.billingCacheService.QueueDeductBalance(task.CreatedBy, amountUSD)
			}
		}
	}

	if s.budget != nil {
		if err := s.budget.Charge(ctx, task.CreatedBy, cost, task.ID); err != nil {
			slog.Warn("video_gateway: post-success budget hook failed", "task_id", task.ID, "error", err)
		}
	}
}

func (s *VideoGatewayService) releaseVideoBalanceClaim(ctx context.Context, taskID int64, claimedAt time.Time) {
	if s == nil || s.repo == nil || taskID <= 0 || claimedAt.IsZero() {
		return
	}
	cleared, err := s.repo.ClearVideoBalanceChargeIfClaimedAt(ctx, taskID, claimedAt)
	if err != nil {
		slog.Warn("video_gateway: release balance charge claim failed", "task_id", taskID, "error", err)
		return
	}
	if !cleared {
		slog.Warn("video_gateway: balance charge claim was not released", "task_id", taskID)
	}
}
