package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemporaryCredentialsMigrationIsAdditive(t *testing.T) {
	content, err := FS.ReadFile("180_user_temporary_credentials.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(content))
	require.Contains(t, sql, "must_change_password")
	require.Contains(t, sql, "temporary_password_expires_at")
	require.Contains(t, sql, "add column if not exists")
	require.NotContains(t, sql, "drop column")
}
