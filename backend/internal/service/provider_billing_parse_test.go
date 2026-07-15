package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestProviderBillingParseCSV_UsesDecimalNotFloat(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "provider_billing", "seedance_statement.csv"))
	require.NoError(t, err)

	header := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-seedance-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC),
		Timezone:           "Asia/Shanghai",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}

	parsed, err := ParseProviderBillingFile(header, "seedance_statement.csv", raw)
	require.NoError(t, err)
	require.NotEmpty(t, parsed.Lines)

	rounding := findLineByExternalID(t, parsed.Lines, "sd-line-002")
	require.Equal(t, "9.995", rounding.NetAmount.String())
	require.True(t, rounding.NetAmount.Equal(decimal.RequireFromString("9.995")))
	require.IsType(t, decimal.Decimal{}, rounding.NetAmount)
	require.IsType(t, decimal.Decimal{}, rounding.UsageQuantity)
}

func TestProviderBillingParseCSV_NormalizesTimezoneKeepsOriginal(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "provider_billing", "seedance_statement.csv"))
	require.NoError(t, err)

	header := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-seedance-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC),
		Timezone:           "Asia/Shanghai",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}

	parsed, err := ParseProviderBillingFile(header, "seedance_statement.csv", raw)
	require.NoError(t, err)

	line := findLineByExternalID(t, parsed.Lines, "sd-line-001")
	require.Equal(t, "Asia/Shanghai", line.OccurredTimezone)
	require.True(t, line.OccurredAt.Equal(time.Date(2026, 7, 1, 2, 0, 0, 0, time.UTC)))
	require.Equal(t, time.UTC, line.OccurredAt.Location())
}

func TestProviderBillingParse_RejectsIllegalCurrencyAndNegatives(t *testing.T) {
	header := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}

	badCurrency := "external_line_id,upstream_task_id,model,sku,usage_quantity,usage_unit,net_amount,tax_amount,gross_amount,currency,occurred_at\n" +
		"x1,,m,s,1,u,1.00,0,1.00,EUR,2026-07-01T00:00:00Z\n"
	_, err := ParseProviderBillingFile(header, "bad.csv", []byte(badCurrency))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingInvalidCurrency)

	negative := "external_line_id,upstream_task_id,model,sku,usage_quantity,usage_unit,net_amount,tax_amount,gross_amount,currency,occurred_at\n" +
		"x2,,m,s,1,u,-1.00,0,1.00,USD,2026-07-01T00:00:00Z\n"
	_, err = ParseProviderBillingFile(header, "bad.csv", []byte(negative))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingNegativeAmount)
}

func TestProviderBillingParse_RejectsOversizedFileAndCell(t *testing.T) {
	header := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}

	oversized := bytes.Repeat([]byte("a"), providerBillingMaxFileBytes+1)
	_, err := ParseProviderBillingFile(header, "big.csv", oversized)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingFileTooLarge)

	longCell := strings.Repeat("x", providerBillingMaxCellBytes+1)
	csv := "external_line_id,upstream_task_id,model,sku,usage_quantity,usage_unit,net_amount,tax_amount,gross_amount,currency,occurred_at\n" +
		longCell + ",,m,s,1,u,1.00,0,1.00,USD,2026-07-01T00:00:00Z\n"
	_, err = ParseProviderBillingFile(header, "cell.csv", []byte(csv))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingCellTooLarge)
}

func TestProviderBillingParse_RejectsDuplicateExternalLineIDInFile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "provider_billing", "gemini_statement.csv"))
	require.NoError(t, err)

	header := ProviderBillingImportHeader{
		Provider:           "gemini",
		ProviderAccountID:  "acct-gemini-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}

	_, err = ParseProviderBillingFile(header, "gemini_statement.csv", raw)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingDuplicateExternalLineID)
}

func TestProviderBillingSHA256_CannotReimportSameRawFile(t *testing.T) {
	raw := []byte("external_line_id,upstream_task_id,model,sku,usage_quantity,usage_unit,net_amount,tax_amount,gross_amount,currency,occurred_at\n" +
		"uniq-1,task-1,m,s,1,u,1.00,0,1.00,USD,2026-07-01T00:00:00Z\n")
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])

	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	store := newMemoryProviderBillingStore()
	svc := NewProviderBillingService(store, dataDir)

	header := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}

	first, err := svc.ImportRawFile(t.Context(), header, "a.csv", raw, 1)
	require.NoError(t, err)
	require.Equal(t, hash, first.FileSHA256)
	require.NotEmpty(t, first.StorageKey)
	require.True(t, strings.HasPrefix(first.StorageKey, providerBillingStoragePrefix))

	_, err = svc.ImportRawFile(t.Context(), header, "a-again.csv", raw, 1)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingDuplicateFileSHA256)

	// DB stores hash + storage_key only; raw path exists under controlled DATA_DIR.
	abs, err := ResolveProviderBillingAbsPath(first.StorageKey)
	require.NoError(t, err)
	_, statErr := os.Stat(abs)
	require.NoError(t, statErr)

	preview, err := svc.PreviewRawFile(t.Context(), header, "a-preview.csv", raw)
	require.NoError(t, err)
	require.True(t, preview.Duplicate, "preview must flag already-imported SHA-256")
	require.Equal(t, hash, preview.FileSHA256)
	require.NotEmpty(t, preview.Lines)
}

func TestProviderBillingImport_RejectsDuplicateProviderExternalLineID(t *testing.T) {
	store := newMemoryProviderBillingStore()
	store.existingExternalIDs["seedance|ext-existing"] = true
	svc := NewProviderBillingService(store, t.TempDir())

	raw := []byte("external_line_id,upstream_task_id,model,sku,usage_quantity,usage_unit,net_amount,tax_amount,gross_amount,currency,occurred_at\n" +
		"ext-existing,task-1,m,s,1,u,1.00,0,1.00,USD,2026-07-01T00:00:00Z\n")
	header := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}

	_, err := svc.ImportRawFile(t.Context(), header, "dup.csv", raw, 1)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProviderBillingDuplicateExternalLineID)
}

func TestProviderBillingMatch_SeedanceAndGeminiTaskIDs(t *testing.T) {
	store := newMemoryProviderBillingStore()
	store.videoByUpstream = map[string]ProviderBillingInternalTask{
		"seedance-task-aaa": {
			RefType:       "video_task",
			RefID:         "vt-1",
			UpstreamTaskID: "seedance-task-aaa",
			Amount:        decimal.RequireFromString("10.00"),
			Usage:         decimal.RequireFromString("1"),
			Currency:      "USD",
			Model:         "seedance-1.0",
			SKU:           "video-720p",
		},
	}
	store.batchByJobName = map[string]ProviderBillingInternalTask{
		"projects/demo/operations/op-exact": {
			RefType:        "batch_image_job",
			RefID:          "bj-1",
			UpstreamTaskID: "projects/demo/operations/op-exact",
			Amount:         decimal.RequireFromString("0.0400000000"),
			Usage:          decimal.RequireFromString("1"),
			Currency:       "USD",
			Model:          "gemini-2.0-flash-preview-image-generation",
			SKU:            "image",
		},
		"projects/demo/operations/op-amount-mismatch": {
			RefType:        "batch_image_job",
			RefID:          "bj-2",
			UpstreamTaskID: "projects/demo/operations/op-amount-mismatch",
			Amount:         decimal.RequireFromString("0.0400000000"),
			Usage:          decimal.RequireFromString("1"),
			Currency:       "USD",
			Model:          "gemini-2.0-flash-preview-image-generation",
			SKU:            "image",
		},
		"projects/demo/operations/op-usage-mismatch": {
			RefType:        "batch_image_job",
			RefID:          "bj-3",
			UpstreamTaskID: "projects/demo/operations/op-usage-mismatch",
			Amount:         decimal.RequireFromString("0.0800000000"),
			Usage:          decimal.RequireFromString("1"),
			Currency:       "USD",
			Model:          "gemini-2.0-flash-preview-image-generation",
			SKU:            "image",
		},
	}
	store.internalOnly = []ProviderBillingInternalTask{
		{
			RefType:        "video_task",
			RefID:          "vt-missing-on-provider",
			UpstreamTaskID: "seedance-internal-only",
			Amount:         decimal.RequireFromString("3.00"),
			Usage:          decimal.RequireFromString("1"),
			Currency:       "USD",
			Model:          "seedance-1.0",
			SKU:            "video-720p",
			AccountDay:     time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	svc := NewProviderBillingService(store, t.TempDir())

	seedanceRaw, err := os.ReadFile(filepath.Join("testdata", "provider_billing", "seedance_statement.csv"))
	require.NoError(t, err)
	seedanceHeader := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-seedance-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "Asia/Shanghai",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}
	imp, err := svc.ImportRawFile(t.Context(), seedanceHeader, "seedance_statement.csv", seedanceRaw, 1)
	require.NoError(t, err)

	matches, err := svc.ReconcileImport(t.Context(), imp.ID)
	require.NoError(t, err)

	byStatus := groupMatchStatuses(matches)
	require.Contains(t, byStatus["matched"], "sd-line-001")
	require.Contains(t, byStatus["provider_only"], "sd-line-004")
	require.Contains(t, byStatus["internal_only"], "seedance-internal-only")

	// No task ID → aggregate_only; never claim per-task consistency.
	agg := findMatchByExternalLine(matches, "sd-line-003")
	require.Equal(t, ProviderBillingMatchAggregateOnly, agg.MatchMode)
	require.NotEqual(t, ProviderBillingMatchMatched, agg.MatchStatus)

	geminiCSV := dedupeGeminiFixture(t)
	geminiHeader := ProviderBillingImportHeader{
		Provider:           "gemini",
		ProviderAccountID:  "acct-gemini-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}
	gImp, err := svc.ImportRawFile(t.Context(), geminiHeader, "gemini_clean.csv", geminiCSV, 1)
	require.NoError(t, err)
	gMatches, err := svc.ReconcileImport(t.Context(), gImp.ID)
	require.NoError(t, err)

	gByStatus := groupMatchStatuses(gMatches)
	require.Contains(t, gByStatus["matched"], "gm-line-001")
	require.Contains(t, gByStatus["amount_mismatch"], "gm-line-003")
	require.Contains(t, gByStatus["usage_mismatch"], "gm-line-002")
	require.Contains(t, gByStatus["provider_only"], "gm-line-005")

	aggGemini := findMatchByExternalLine(gMatches, "gm-line-004")
	require.Equal(t, ProviderBillingMatchAggregateOnly, aggGemini.MatchMode)
}

func TestProviderBillingMatchEngine_NeverTouchesUserBalance(t *testing.T) {
	store := newMemoryProviderBillingStore()
	svc := NewProviderBillingService(store, t.TempDir())
	raw := []byte("external_line_id,upstream_task_id,model,sku,usage_quantity,usage_unit,net_amount,tax_amount,gross_amount,currency,occurred_at\n" +
		"bal-1,task-bal,m,s,1,u,1.00,0,1.00,USD,2026-07-01T00:00:00Z\n")
	header := ProviderBillingImportHeader{
		Provider:           "seedance",
		ProviderAccountID:  "acct-1",
		BillingPeriodStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		BillingPeriodEnd:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		Timezone:           "UTC",
		OriginalCurrency:   "USD",
		SourceType:         "csv",
	}
	imp, err := svc.ImportRawFile(t.Context(), header, "bal.csv", raw, 1)
	require.NoError(t, err)
	_, err = svc.ReconcileImport(t.Context(), imp.ID)
	require.NoError(t, err)
	require.False(t, store.balanceTouched)
	require.False(t, store.billingTxTouched)
}

func findLineByExternalID(t *testing.T, lines []ProviderBillingNormalizedLine, id string) ProviderBillingNormalizedLine {
	t.Helper()
	for _, line := range lines {
		if line.ExternalLineID == id {
			return line
		}
	}
	t.Fatalf("line %s not found", id)
	return ProviderBillingNormalizedLine{}
}

func groupMatchStatuses(matches []ProviderBillingMatchResult) map[string][]string {
	out := map[string][]string{}
	for _, m := range matches {
		key := string(m.MatchStatus)
		label := m.ExternalLineID
		if label == "" {
			label = m.InternalRefID
		}
		out[key] = append(out[key], label)
	}
	return out
}

func findMatchByExternalLine(matches []ProviderBillingMatchResult, externalLineID string) ProviderBillingMatchResult {
	for _, m := range matches {
		if m.ExternalLineID == externalLineID {
			return m
		}
	}
	return ProviderBillingMatchResult{}
}

func dedupeGeminiFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "provider_billing", "gemini_statement.csv"))
	require.NoError(t, err)
	lines := strings.Split(string(raw), "\n")
	seen := map[string]bool{}
	var out []string
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i == 0 {
			out = append(out, line)
			continue
		}
		id := strings.Split(line, ",")[0]
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n") + "\n")
}
