package repository

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffiliateUserOverviewSQLIncludesMaturedFrozenQuota(t *testing.T) {
	query := strings.Join(strings.Fields(affiliateUserOverviewSQL), " ")

	require.Contains(t, query, "ua.aff_quota + COALESCE(matured.matured_frozen_quota, 0)")
	require.Contains(t, query, "frozen_until <= NOW()")
}

func TestAffiliateRecordQueriesUseLedgerAuditFields(t *testing.T) {
	source, err := os.ReadFile("affiliate_repo.go")
	require.NoError(t, err)
	content := string(source)

	require.Contains(t, content, "JOIN payment_orders po ON po.id = ual.source_order_id")
	require.Contains(t, content, "ual.amount::double precision")
	require.Contains(t, content, "ual.balance_after::double precision")
	require.NotContains(t, content, "parseAffiliateRebateAmount")
	require.NotContains(t, content, `"current_balance": "u.balance"`)
}

func TestAffiliateAccrueLedgerMigrationAddsPartialUniqueIndex(t *testing.T) {
	source, err := os.ReadFile("../../migrations/148_user_affiliate_ledger_accrue_unique.sql")
	require.NoError(t, err)
	content := string(source)

	require.Contains(t, content, "CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_accrue_order_uniq")
	require.Contains(t, content, "ON user_affiliate_ledger(user_id, source_order_id)")
	require.Contains(t, content, "WHERE action = 'accrue' AND source_order_id IS NOT NULL")
	require.Contains(t, content, "RAISE EXCEPTION")
}
