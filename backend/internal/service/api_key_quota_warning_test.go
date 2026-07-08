package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuotaUsagePercentBoundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		used    float64
		limit   float64
		percent float64
		level   string
	}{
		{name: "unlimited", used: 100, limit: 0, percent: 0, level: QuotaWarningNone},
		{name: "negative_limit", used: 10, limit: -1, percent: 0, level: QuotaWarningNone},
		{name: "79_percent", used: 7.9, limit: 10, percent: 79, level: QuotaWarningNone},
		{name: "80_percent", used: 8, limit: 10, percent: 80, level: QuotaWarningWarn},
		{name: "99_percent", used: 9.9, limit: 10, percent: 99, level: QuotaWarningWarn},
		{name: "100_percent", used: 10, limit: 10, percent: 100, level: QuotaWarningCritical},
		{name: "over_100", used: 12, limit: 10, percent: 120, level: QuotaWarningCritical},
		{name: "zero_used", used: 0, limit: 10, percent: 0, level: QuotaWarningNone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			percent, level := QuotaUsagePercent(tc.used, tc.limit)
			require.Equal(t, tc.level, level)
			require.InDelta(t, tc.percent, percent, 0.01)
		})
	}
}
