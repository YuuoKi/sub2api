package migrations

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoLocalAssetMigrationIsForwardOnlyAndKeepsHistoryUnknown(t *testing.T) {
	b, err := FS.ReadFile("185_video_task_local_asset.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(b))

	for _, column := range []string{"local_asset_path", "local_asset_saved_at"} {
		require.Contains(t, sql, column)
		require.NotRegexp(t, regexp.MustCompile(column+`[^,;]*not\s+null`), sql)
		require.NotRegexp(t, regexp.MustCompile(column+`[^,;]*default`), sql)
	}
	require.NotRegexp(t, regexp.MustCompile(`(?m)^\s*(update|delete|drop|truncate)\b|insert\s+into[\s\S]+select\s`), sql)
}
