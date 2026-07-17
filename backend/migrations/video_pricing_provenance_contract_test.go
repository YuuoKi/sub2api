package migrations

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoPricingProvenanceMigrationIsForwardOnlyAndKeepsHistoryUnknown(t *testing.T) {
	b, err := FS.ReadFile("184_video_task_pricing_provenance.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(b))

	for _, required := range []string{
		"pricing_source",
		"pricing_version",
		"pricing_cny_per_million_completion_tokens",
		"pricing_usd_cny_exchange_rate",
		"pricing_maximum_cny",
	} {
		require.Contains(t, sql, required)
		require.NotRegexp(t, regexp.MustCompile(required+`[^,;]*not\s+null`), sql)
		require.NotRegexp(t, regexp.MustCompile(required+`[^,;]*default`), sql)
	}
	require.NotRegexp(t, regexp.MustCompile(`(?m)^\s*(update|delete|drop|truncate)\b|insert\s+into[\s\S]+select\s`), sql)
}
