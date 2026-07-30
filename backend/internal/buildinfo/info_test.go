package buildinfo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFormatsGuangzhouDisplayLabel(t *testing.T) {
	info := New("0.1.151", "abcdef0123456789abcdef0123456789abcdef01", "2026-07-25T08:30:00Z", "source")
	require.Equal(t, "广州内部版 2026.07.25-r151", info.Version)
	require.Equal(t, "abcdef0123456789abcdef0123456789abcdef01", info.BuildCommit)
	require.Equal(t, "2026-07-25T08:30:00Z", info.BuildDate)
}

func TestNormalizeCommitNeverForcePrefixV(t *testing.T) {
	require.Equal(t, "abc1234def", NormalizeCommit("abc1234def"))
	require.Equal(t, "abc1234def", NormalizeCommit("vabc1234def"))
	require.Equal(t, "v1.2.3-release", NormalizeCommit("v1.2.3-release"))
}

func TestFourIdentitySourcesShareSameInfo(t *testing.T) {
	info := New("0.1.151", "deadbeefcafebabe", "2026-07-25", "release")

	cli := info.CLIString()
	require.True(t, strings.HasPrefix(cli, info.Version))
	require.Contains(t, cli, info.BuildCommit)
	require.Contains(t, cli, info.BuildDate)
	require.False(t, strings.Contains(cli, "v"+info.BuildCommit))

	pub := info.PublicMap()
	admin := info.PublicMap()
	injection := info.PublicMap()

	require.Equal(t, pub, admin)
	require.Equal(t, pub, injection)
	require.Equal(t, "广州内部版 2026.07.25-r151", pub["version"])
	require.Equal(t, "deadbeefcafebabe", pub["build_commit"])
	require.Equal(t, "2026-07-25", pub["build_date"])
}
