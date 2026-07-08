package service

import "math"

const (
	QuotaWarningNone     = "none"
	QuotaWarningWarn     = "warn"
	QuotaWarningCritical = "critical"

	quotaWarnThreshold     = 0.80
	quotaCriticalThreshold = 1.00
)

// QuotaUsagePercent returns usage percent (0-100) and warning level for a key quota.
// limit <= 0 means unlimited → percent 0, level none.
func QuotaUsagePercent(used, limit float64) (percent float64, level string) {
	if limit <= 0 || math.IsNaN(limit) || math.IsInf(limit, 0) {
		return 0, QuotaWarningNone
	}
	if math.IsNaN(used) || math.IsInf(used, 0) || used < 0 {
		used = 0
	}
	ratio := used / limit
	percent = math.Round(ratio*10000) / 100 // 2 decimal places
	if percent < 0 {
		percent = 0
	}
	switch {
	case ratio >= quotaCriticalThreshold:
		return percent, QuotaWarningCritical
	case ratio >= quotaWarnThreshold:
		return percent, QuotaWarningWarn
	default:
		return percent, QuotaWarningNone
	}
}

// APIKeyQuotaWarningItem is one key approaching or at quota limit.
type APIKeyQuotaWarningItem struct {
	ID                 int64   `json:"id"`
	UserID             int64   `json:"user_id"`
	Name               string  `json:"name"`
	Quota              float64 `json:"quota"`
	QuotaUsed          float64 `json:"quota_used"`
	QuotaUsagePercent  float64 `json:"quota_usage_percent"`
	QuotaWarningLevel  string  `json:"quota_warning_level"`
	Username           string  `json:"username,omitempty"`
	Email              string  `json:"email,omitempty"`
}

// APIKeyQuotaWarningsSummary aggregates warn/critical key counts for the console overview.
type APIKeyQuotaWarningsSummary struct {
	WarnCount     int                     `json:"warn_count"`
	CriticalCount int                     `json:"critical_count"`
	TopItems      []APIKeyQuotaWarningItem `json:"top_items"`
}
