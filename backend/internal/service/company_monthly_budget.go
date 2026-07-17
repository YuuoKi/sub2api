package service

import (
	"context"
	"math"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrInvalidCompanyMonthlyBudget = infraerrors.BadRequest(
	"INVALID_COMPANY_MONTHLY_BUDGET",
	"company monthly budget must be a finite non-negative number",
)

// MonthlyBudgetUsagePercent returns spend/budget*100. An unset budget returns zero.
func MonthlyBudgetUsagePercent(spendCNY, budgetCNY float64) float64 {
	if budgetCNY <= 0 || math.IsNaN(budgetCNY) || math.IsInf(budgetCNY, 0) {
		return 0
	}
	if spendCNY < 0 || math.IsNaN(spendCNY) || math.IsInf(spendCNY, 0) {
		spendCNY = 0
	}
	return spendCNY / budgetCNY * 100
}

// GetCompanyMonthlyBudgetCNY returns the persisted budget, or zero when unset/invalid.
func (s *SettingService) GetCompanyMonthlyBudgetCNY(ctx context.Context) float64 {
	if s == nil || s.settingRepo == nil {
		return 0
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyCompanyMonthlyBudgetCNY)
	if err != nil {
		return 0
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

// SetCompanyMonthlyBudgetCNY persists a finite non-negative budget. Zero clears it.
func (s *SettingService) SetCompanyMonthlyBudgetCNY(ctx context.Context, budgetCNY float64) error {
	if budgetCNY < 0 || math.IsNaN(budgetCNY) || math.IsInf(budgetCNY, 0) {
		return ErrInvalidCompanyMonthlyBudget
	}
	if s == nil || s.settingRepo == nil {
		return ErrSettingNotFound
	}
	if budgetCNY == 0 {
		return s.settingRepo.Delete(ctx, SettingKeyCompanyMonthlyBudgetCNY)
	}
	return s.settingRepo.Set(ctx, SettingKeyCompanyMonthlyBudgetCNY, strconv.FormatFloat(budgetCNY, 'f', -1, 64))
}
