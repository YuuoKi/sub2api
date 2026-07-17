package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockVideoProviderMigration186SeedsInternalSimulationAccount(t *testing.T) {
	b, err := FS.ReadFile("186_mock_video_provider.sql")
	require.NoError(t, err, "migration 186 must exist for internal mock provider seed")
	sql := strings.ToLower(string(b))

	require.Contains(t, sql, "video_provider_accounts")
	require.Contains(t, sql, "'mock'")
	require.Contains(t, sql, "mock-video-v1")
	require.NotContains(t, sql, "ark.cn-beijing")
	require.NotContains(t, sql, "sk-")
	require.NotRegexp(t, `(?i)\bdrop\b|\btruncate\b`, sql)
}
