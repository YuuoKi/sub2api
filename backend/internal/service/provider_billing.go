package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	providerBillingStoragePrefix        = "billing/provider_statements/"
	providerBillingMaxFileBytes         = 10 << 20 // 10 MiB
	providerBillingMaxUncompressedBytes = 50 << 20 // 50 MiB
	providerBillingMaxRows              = 100_000
	providerBillingMaxSheets            = 5
	providerBillingMaxCellBytes         = 8 << 10 // 8 KiB
)

// ProviderBillingMaxUploadBytes exposes the hard upload ceiling to HTTP handlers.
func ProviderBillingMaxUploadBytes() int {
	return providerBillingMaxFileBytes
}

var (
	ErrProviderBillingInvalidCurrency         = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_INVALID_CURRENCY", "currency must be CNY or USD")
	ErrProviderBillingNegativeAmount          = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_NEGATIVE_AMOUNT", "amounts and usage must not be negative")
	ErrProviderBillingFileTooLarge            = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_FILE_TOO_LARGE", "billing file exceeds size limit")
	ErrProviderBillingCellTooLarge            = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_CELL_TOO_LARGE", "billing cell exceeds length limit")
	ErrProviderBillingTooManyRows             = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_TOO_MANY_ROWS", "billing file exceeds row limit")
	ErrProviderBillingTooManySheets           = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_TOO_MANY_SHEETS", "billing workbook exceeds sheet limit")
	ErrProviderBillingZipBomb                 = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_ZIP_BOMB", "billing workbook uncompressed size exceeds limit")
	ErrProviderBillingFormulaCell             = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_FORMULA_CELL", "formula cells are not allowed")
	ErrProviderBillingExternalLink            = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_EXTERNAL_LINK", "hidden external links are not allowed")
	ErrProviderBillingDuplicateFileSHA256     = infraerrors.New(http.StatusConflict, "PROVIDER_BILLING_DUPLICATE_FILE", "identical file SHA-256 already imported")
	ErrProviderBillingDuplicateExternalLineID = infraerrors.New(http.StatusConflict, "PROVIDER_BILLING_DUPLICATE_EXTERNAL_LINE", "provider external_line_id already exists")
	ErrProviderBillingInvalidHeader           = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_INVALID_HEADER", "billing import header is invalid")
	ErrProviderBillingInvalidFile             = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_INVALID_FILE", "billing file could not be parsed")
	ErrProviderBillingNotFound                = infraerrors.New(http.StatusNotFound, "PROVIDER_BILLING_NOT_FOUND", "billing import not found")
	ErrProviderBillingPathUnsafe              = infraerrors.New(http.StatusBadRequest, "PROVIDER_BILLING_PATH_UNSAFE", "billing storage path is unsafe")
)

type ProviderBillingMatchStatus string

const (
	ProviderBillingMatchMatched         ProviderBillingMatchStatus = "matched"
	ProviderBillingMatchAmountMismatch  ProviderBillingMatchStatus = "amount_mismatch"
	ProviderBillingMatchUsageMismatch   ProviderBillingMatchStatus = "usage_mismatch"
	ProviderBillingMatchInternalOnly    ProviderBillingMatchStatus = "internal_only"
	ProviderBillingMatchProviderOnly    ProviderBillingMatchStatus = "provider_only"
	ProviderBillingMatchAdjustment      ProviderBillingMatchStatus = "adjustment"
)

type ProviderBillingMatchMode string

const (
	ProviderBillingMatchTaskID         ProviderBillingMatchMode = "task_id"
	ProviderBillingMatchAggregateOnly  ProviderBillingMatchMode = "aggregate_only"
)

type ProviderBillingImportHeader struct {
	Provider           string
	ProviderAccountID  string
	BillingPeriodStart time.Time
	BillingPeriodEnd   time.Time
	Timezone           string
	OriginalCurrency   string
	SourceType         string
	InvoiceNumber      string
}

type ProviderBillingNormalizedLine struct {
	ExternalLineID   string          `json:"external_line_id"`
	UpstreamTaskID   string          `json:"upstream_task_id"`
	Model            string          `json:"model"`
	SKU              string          `json:"sku"`
	UsageQuantity    decimal.Decimal `json:"usage_quantity"`
	UsageUnit        string          `json:"usage_unit"`
	NetAmount        decimal.Decimal `json:"net_amount"`
	TaxAmount        decimal.Decimal `json:"tax_amount"`
	GrossAmount      decimal.Decimal `json:"gross_amount"`
	Currency         string          `json:"currency"`
	OccurredAt       time.Time       `json:"occurred_at"`
	OccurredTimezone string          `json:"-"`
}

func (l ProviderBillingNormalizedLine) MarshalJSON() ([]byte, error) {
	type wire struct {
		ExternalLineID string `json:"external_line_id"`
		UpstreamTaskID string `json:"upstream_task_id"`
		Model          string `json:"model"`
		SKU            string `json:"sku"`
		UsageQuantity  string `json:"usage_quantity"`
		UsageUnit      string `json:"usage_unit"`
		NetAmount      string `json:"net_amount"`
		TaxAmount      string `json:"tax_amount"`
		GrossAmount    string `json:"gross_amount"`
		Currency       string `json:"currency"`
		OccurredAt     string `json:"occurred_at"`
	}
	return json.Marshal(wire{
		ExternalLineID: l.ExternalLineID,
		UpstreamTaskID: l.UpstreamTaskID,
		Model:          l.Model,
		SKU:            l.SKU,
		UsageQuantity:  l.UsageQuantity.String(),
		UsageUnit:      l.UsageUnit,
		NetAmount:      l.NetAmount.String(),
		TaxAmount:      l.TaxAmount.String(),
		GrossAmount:    l.GrossAmount.String(),
		Currency:       l.Currency,
		OccurredAt:     l.OccurredAt.UTC().Format(time.RFC3339),
	})
}

type ProviderBillingParseResult struct {
	Header   ProviderBillingImportHeader
	FileSHA256 string
	Lines    []ProviderBillingNormalizedLine
}

type ProviderBillingImportRecord struct {
	ID                 int64
	Provider           string
	ProviderAccountID  string
	BillingPeriodStart time.Time
	BillingPeriodEnd   time.Time
	Timezone           string
	OriginalCurrency   string
	SourceType         string
	InvoiceNumber      string
	FileSHA256         string
	StorageKey         string
	OriginalFilename   string
	ByteSize           int64
	Status             string
	LineCount          int
	CreatedBy          int64
	CreatedAt          time.Time
	Lines              []ProviderBillingNormalizedLine
}

type ProviderBillingInternalTask struct {
	RefType        string
	RefID          string
	UpstreamTaskID string
	Amount         decimal.Decimal
	Usage          decimal.Decimal
	Currency       string
	Model          string
	SKU            string
	AccountDay     time.Time
}

type ProviderBillingMatchResult struct {
	ID               int64
	ImportID         int64
	BillingLineID    int64
	ExternalLineID   string
	MatchStatus      ProviderBillingMatchStatus
	MatchMode        ProviderBillingMatchMode
	InternalRefType  string
	InternalRefID    string
	ProviderAmount   decimal.Decimal
	InternalAmount   decimal.Decimal
	ProviderUsage    decimal.Decimal
	InternalUsage    decimal.Decimal
	Currency         string
	Model            string
	SKU              string
	AccountDay       *time.Time
	DiffJSON         map[string]any
}

type ProviderBillingPeriodSummary struct {
	Provider           string `json:"provider"`
	ProviderAccountID  string `json:"provider_account_id"`
	BillingPeriodStart string `json:"billing_period_start"`
	BillingPeriodEnd   string `json:"billing_period_end"`
	ImportCount        int    `json:"import_count"`
	Matched            int    `json:"matched"`
	HasDiff            int    `json:"has_diff"`
	ProviderOnly       int    `json:"provider_only"`
	InternalOnly       int    `json:"internal_only"`
	Conclusion         string `json:"conclusion"` // reconciled | has_diff | not_uploaded
}

type ProviderBillingStore interface {
	HasFileSHA256(ctx context.Context, sha256hex string) (bool, error)
	HasProviderExternalLineID(ctx context.Context, provider, externalLineID string) (bool, error)
	CreateImport(ctx context.Context, rec *ProviderBillingImportRecord) (*ProviderBillingImportRecord, error)
	GetImport(ctx context.Context, id int64) (*ProviderBillingImportRecord, error)
	ListImports(ctx context.Context, provider string, limit int) ([]ProviderBillingImportRecord, error)
	ReplaceMatches(ctx context.Context, importID int64, matches []ProviderBillingMatchResult) error
	ListMatches(ctx context.Context, importID int64, status string) ([]ProviderBillingMatchResult, error)
	FindVideoTaskByUpstreamID(ctx context.Context, upstreamTaskID string) (*ProviderBillingInternalTask, error)
	FindBatchImageJobByProviderJobName(ctx context.Context, providerJobName string) (*ProviderBillingInternalTask, error)
	ListInternalTasksForPeriod(ctx context.Context, provider string, accountID string, start, end time.Time) ([]ProviderBillingInternalTask, error)
	PeriodSummary(ctx context.Context, start, end time.Time) ([]ProviderBillingPeriodSummary, error)
	BossConclusions(ctx context.Context) ([]ProviderBillingPeriodSummary, error)
}

type ProviderBillingService struct {
	store    ProviderBillingStore
	dataDir  string
}

func NewProviderBillingService(store ProviderBillingStore, dataDir string) *ProviderBillingService {
	if strings.TrimSpace(dataDir) == "" {
		dataDir = videoAssetDataDir()
	}
	return &ProviderBillingService{store: store, dataDir: dataDir}
}

func ParseProviderBillingFile(header ProviderBillingImportHeader, filename string, raw []byte) (*ProviderBillingParseResult, error) {
	if err := validateProviderBillingHeader(header); err != nil {
		return nil, err
	}
	if len(raw) > providerBillingMaxFileBytes {
		return nil, ErrProviderBillingFileTooLarge
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])

	sourceType := strings.ToLower(strings.TrimSpace(header.SourceType))
	ext := strings.ToLower(filepath.Ext(filename))
	var lines []ProviderBillingNormalizedLine
	var err error
	switch {
	case sourceType == "csv" || ext == ".csv":
		lines, err = parseProviderBillingCSV(header, raw)
	case sourceType == "xlsx" || ext == ".xlsx":
		lines, err = parseProviderBillingXLSX(header, raw)
	default:
		return nil, ErrProviderBillingInvalidFile
	}
	if err != nil {
		return nil, err
	}
	if len(lines) > providerBillingMaxRows {
		return nil, ErrProviderBillingTooManyRows
	}
	if err := ensureUniqueExternalLineIDs(lines); err != nil {
		return nil, err
	}
	return &ProviderBillingParseResult{Header: header, FileSHA256: hash, Lines: lines}, nil
}

func (s *ProviderBillingService) PreviewRawFile(ctx context.Context, header ProviderBillingImportHeader, filename string, raw []byte) (*ProviderBillingParseResult, error) {
	_ = ctx
	return ParseProviderBillingFile(header, filename, raw)
}

func (s *ProviderBillingService) ImportRawFile(ctx context.Context, header ProviderBillingImportHeader, filename string, raw []byte, createdBy int64) (*ProviderBillingImportRecord, error) {
	parsed, err := ParseProviderBillingFile(header, filename, raw)
	if err != nil {
		return nil, err
	}
	exists, err := s.store.HasFileSHA256(ctx, parsed.FileSHA256)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProviderBillingDuplicateFileSHA256
	}
	for _, line := range parsed.Lines {
		dup, err := s.store.HasProviderExternalLineID(ctx, header.Provider, line.ExternalLineID)
		if err != nil {
			return nil, err
		}
		if dup {
			return nil, ErrProviderBillingDuplicateExternalLineID
		}
	}

	storageKey, err := s.persistRawFile(parsed.FileSHA256, filename, raw)
	if err != nil {
		return nil, err
	}

	rec := &ProviderBillingImportRecord{
		Provider:           header.Provider,
		ProviderAccountID:  header.ProviderAccountID,
		BillingPeriodStart: header.BillingPeriodStart,
		BillingPeriodEnd:   header.BillingPeriodEnd,
		Timezone:           header.Timezone,
		OriginalCurrency:   header.OriginalCurrency,
		SourceType:         strings.ToLower(header.SourceType),
		InvoiceNumber:      header.InvoiceNumber,
		FileSHA256:         parsed.FileSHA256,
		StorageKey:         storageKey,
		OriginalFilename:   filepath.Base(filename),
		ByteSize:           int64(len(raw)),
		Status:             "imported",
		LineCount:          len(parsed.Lines),
		CreatedBy:          createdBy,
		Lines:              parsed.Lines,
	}
	return s.store.CreateImport(ctx, rec)
}

func (s *ProviderBillingService) ReconcileImport(ctx context.Context, importID int64) ([]ProviderBillingMatchResult, error) {
	imp, err := s.store.GetImport(ctx, importID)
	if err != nil {
		return nil, err
	}
	if imp == nil {
		return nil, ErrProviderBillingNotFound
	}

	matches := make([]ProviderBillingMatchResult, 0, len(imp.Lines)+8)
	claimedInternal := map[string]bool{}

	for _, line := range imp.Lines {
		match := ProviderBillingMatchResult{
			ImportID:       importID,
			ExternalLineID: line.ExternalLineID,
			ProviderAmount: line.NetAmount,
			ProviderUsage:  line.UsageQuantity,
			Currency:       line.Currency,
			Model:          line.Model,
			SKU:            line.SKU,
		}

		taskID := strings.TrimSpace(line.UpstreamTaskID)
		if taskID == "" {
			match.MatchMode = ProviderBillingMatchAggregateOnly
			match.MatchStatus = ProviderBillingMatchProviderOnly
			match.DiffJSON = map[string]any{
				"reason": "no_task_id_aggregate_only",
				"note":   "never claim per-task consistency without upstream task id",
			}
			day := line.OccurredAt.UTC().Truncate(24 * time.Hour)
			match.AccountDay = &day
			matches = append(matches, match)
			continue
		}

		match.MatchMode = ProviderBillingMatchTaskID
		internal, err := s.lookupInternalTask(ctx, imp.Provider, taskID)
		if err != nil {
			return nil, err
		}
		if internal == nil {
			match.MatchStatus = ProviderBillingMatchProviderOnly
			matches = append(matches, match)
			continue
		}

		claimedInternal[internal.RefType+"|"+internal.RefID] = true
		match.InternalRefType = internal.RefType
		match.InternalRefID = internal.RefID
		match.InternalAmount = internal.Amount
		match.InternalUsage = internal.Usage

		amountOK := line.NetAmount.Equal(internal.Amount)
		usageOK := line.UsageQuantity.Equal(internal.Usage)
		switch {
		case amountOK && usageOK:
			match.MatchStatus = ProviderBillingMatchMatched
		case !amountOK:
			match.MatchStatus = ProviderBillingMatchAmountMismatch
			match.DiffJSON = map[string]any{
				"provider_amount": line.NetAmount.String(),
				"internal_amount": internal.Amount.String(),
			}
		default:
			match.MatchStatus = ProviderBillingMatchUsageMismatch
			match.DiffJSON = map[string]any{
				"provider_usage": line.UsageQuantity.String(),
				"internal_usage": internal.Usage.String(),
			}
		}
		matches = append(matches, match)
	}

	internals, err := s.store.ListInternalTasksForPeriod(ctx, imp.Provider, imp.ProviderAccountID, imp.BillingPeriodStart, imp.BillingPeriodEnd)
	if err != nil {
		return nil, err
	}
	for _, internal := range internals {
		key := internal.RefType + "|" + internal.RefID
		if claimedInternal[key] {
			continue
		}
		if strings.TrimSpace(internal.UpstreamTaskID) == "" {
			continue
		}
		day := internal.AccountDay
		matches = append(matches, ProviderBillingMatchResult{
			ImportID:        importID,
			MatchStatus:     ProviderBillingMatchInternalOnly,
			MatchMode:       ProviderBillingMatchTaskID,
			InternalRefType: internal.RefType,
			InternalRefID:   internal.UpstreamTaskID,
			InternalAmount:  internal.Amount,
			InternalUsage:   internal.Usage,
			Currency:        internal.Currency,
			Model:           internal.Model,
			SKU:             internal.SKU,
			AccountDay:      &day,
		})
	}

	if err := s.store.ReplaceMatches(ctx, importID, matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func (s *ProviderBillingService) lookupInternalTask(ctx context.Context, provider, taskID string) (*ProviderBillingInternalTask, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "seedance":
		return s.store.FindVideoTaskByUpstreamID(ctx, taskID)
	case "gemini":
		return s.store.FindBatchImageJobByProviderJobName(ctx, taskID)
	default:
		// Try both keys for unknown providers.
		if vt, err := s.store.FindVideoTaskByUpstreamID(ctx, taskID); err != nil {
			return nil, err
		} else if vt != nil {
			return vt, nil
		}
		return s.store.FindBatchImageJobByProviderJobName(ctx, taskID)
	}
}

func (s *ProviderBillingService) GetPeriodSummary(ctx context.Context, start, end time.Time) ([]ProviderBillingPeriodSummary, error) {
	return s.store.PeriodSummary(ctx, start, end)
}

func (s *ProviderBillingService) GetBossConclusions(ctx context.Context) ([]ProviderBillingPeriodSummary, error) {
	return s.store.BossConclusions(ctx)
}

func (s *ProviderBillingService) ListMatches(ctx context.Context, importID int64, status string) ([]ProviderBillingMatchResult, error) {
	return s.store.ListMatches(ctx, importID, status)
}

func (s *ProviderBillingService) GetImport(ctx context.Context, id int64) (*ProviderBillingImportRecord, error) {
	return s.store.GetImport(ctx, id)
}

func (s *ProviderBillingService) ListImports(ctx context.Context, provider string, limit int) ([]ProviderBillingImportRecord, error) {
	return s.store.ListImports(ctx, provider, limit)
}

func (s *ProviderBillingService) ExportMatchesCSV(ctx context.Context, importID int64) ([]byte, error) {
	matches, err := s.store.ListMatches(ctx, importID, "")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{
		"external_line_id", "match_status", "match_mode", "internal_ref_type", "internal_ref_id",
		"provider_amount", "internal_amount", "provider_usage", "internal_usage", "currency", "model", "sku",
	})
	for _, m := range matches {
		_ = w.Write([]string{
			m.ExternalLineID,
			string(m.MatchStatus),
			string(m.MatchMode),
			m.InternalRefType,
			m.InternalRefID,
			m.ProviderAmount.String(),
			m.InternalAmount.String(),
			m.ProviderUsage.String(),
			m.InternalUsage.String(),
			m.Currency,
			m.Model,
			m.SKU,
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func (s *ProviderBillingService) persistRawFile(sha256hex, filename string, raw []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	rel := providerBillingStoragePrefix + sha256hex + ext
	abs, err := resolveProviderBillingAbsPathWithRoot(s.dataDir, rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return "", err
	}
	tmp := abs + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o640); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return rel, nil
}

func ResolveProviderBillingAbsPath(rel string) (string, error) {
	return resolveProviderBillingAbsPathWithRoot(videoAssetDataDir(), rel)
}

func resolveProviderBillingAbsPathWithRoot(root, rel string) (string, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		return "", ErrProviderBillingPathUnsafe
	}
	if !strings.HasPrefix(rel, providerBillingStoragePrefix) {
		return "", ErrProviderBillingPathUnsafe
	}
	root = filepath.Clean(root)
	abs := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	if !isPathWithinBase(abs, root) {
		return "", ErrProviderBillingPathUnsafe
	}
	return abs, nil
}

func validateProviderBillingHeader(header ProviderBillingImportHeader) error {
	if strings.TrimSpace(header.Provider) == "" || strings.TrimSpace(header.ProviderAccountID) == "" {
		return ErrProviderBillingInvalidHeader
	}
	if strings.TrimSpace(header.Timezone) == "" {
		return ErrProviderBillingInvalidHeader
	}
	if err := validateProviderBillingCurrency(header.OriginalCurrency); err != nil {
		return err
	}
	st := strings.ToLower(strings.TrimSpace(header.SourceType))
	if st != "csv" && st != "xlsx" {
		return ErrProviderBillingInvalidHeader
	}
	if header.BillingPeriodEnd.Before(header.BillingPeriodStart) {
		return ErrProviderBillingInvalidHeader
	}
	return nil
}

func validateProviderBillingCurrency(currency string) error {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "CNY", "USD":
		return nil
	default:
		return ErrProviderBillingInvalidCurrency
	}
}

func ensureUniqueExternalLineIDs(lines []ProviderBillingNormalizedLine) error {
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		id := strings.TrimSpace(line.ExternalLineID)
		if id == "" {
			return ErrProviderBillingInvalidFile
		}
		if _, ok := seen[id]; ok {
			return ErrProviderBillingDuplicateExternalLineID
		}
		seen[id] = struct{}{}
	}
	return nil
}

func parseProviderBillingCSV(header ProviderBillingImportHeader, raw []byte) ([]ProviderBillingNormalizedLine, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
	}
	if len(records) < 2 {
		return nil, ErrProviderBillingInvalidFile
	}
	for _, record := range records {
		for _, cell := range record {
			if len(cell) > providerBillingMaxCellBytes {
				return nil, ErrProviderBillingCellTooLarge
			}
		}
	}
	col := mapCSVHeader(records[0])
	required := []string{
		"external_line_id", "upstream_task_id", "model", "sku", "usage_quantity",
		"usage_unit", "net_amount", "tax_amount", "gross_amount", "currency", "occurred_at",
	}
	for _, key := range required {
		if _, ok := col[key]; !ok {
			return nil, fmt.Errorf("%w: missing column %s", ErrProviderBillingInvalidFile, key)
		}
	}

	lines := make([]ProviderBillingNormalizedLine, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		row := records[i]
		if isCSVRowEmpty(row) {
			continue
		}
		line, err := normalizeProviderBillingRow(header, func(name string) string {
			idx := col[name]
			if idx < 0 || idx >= len(row) {
				return ""
			}
			return row[idx]
		})
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	if len(lines) > providerBillingMaxRows {
		return nil, ErrProviderBillingTooManyRows
	}
	return lines, nil
}

func mapCSVHeader(header []string) map[string]int {
	out := make(map[string]int, len(header))
	for i, name := range header {
		key := strings.ToLower(strings.TrimSpace(name))
		out[key] = i
	}
	return out
}

func isCSVRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func normalizeProviderBillingRow(header ProviderBillingImportHeader, get func(string) string) (ProviderBillingNormalizedLine, error) {
	currency := strings.ToUpper(strings.TrimSpace(get("currency")))
	if err := validateProviderBillingCurrency(currency); err != nil {
		return ProviderBillingNormalizedLine{}, err
	}

	usage, err := parseNonNegativeDecimal(get("usage_quantity"))
	if err != nil {
		return ProviderBillingNormalizedLine{}, err
	}
	netAmount, err := parseNonNegativeDecimal(get("net_amount"))
	if err != nil {
		return ProviderBillingNormalizedLine{}, err
	}
	taxAmount, err := parseNonNegativeDecimal(get("tax_amount"))
	if err != nil {
		return ProviderBillingNormalizedLine{}, err
	}
	grossAmount, err := parseNonNegativeDecimal(get("gross_amount"))
	if err != nil {
		return ProviderBillingNormalizedLine{}, err
	}

	occurredRaw := strings.TrimSpace(get("occurred_at"))
	occurredAt, occurredTZ, err := normalizeOccurredAt(occurredRaw, header.Timezone)
	if err != nil {
		return ProviderBillingNormalizedLine{}, fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
	}

	externalID := strings.TrimSpace(get("external_line_id"))
	if externalID == "" {
		return ProviderBillingNormalizedLine{}, ErrProviderBillingInvalidFile
	}

	return ProviderBillingNormalizedLine{
		ExternalLineID:   externalID,
		UpstreamTaskID:   strings.TrimSpace(get("upstream_task_id")),
		Model:            strings.TrimSpace(get("model")),
		SKU:              strings.TrimSpace(get("sku")),
		UsageQuantity:    usage,
		UsageUnit:        strings.TrimSpace(get("usage_unit")),
		NetAmount:        netAmount,
		TaxAmount:        taxAmount,
		GrossAmount:      grossAmount,
		Currency:         currency,
		OccurredAt:       occurredAt,
		OccurredTimezone: occurredTZ,
	}, nil
}

func parseNonNegativeDecimal(raw string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return decimal.Zero, fmt.Errorf("%w: %v", ErrProviderBillingInvalidFile, err)
	}
	if value.IsNegative() {
		return decimal.Zero, ErrProviderBillingNegativeAmount
	}
	return value, nil
}

func normalizeOccurredAt(raw, fallbackTZ string) (time.Time, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, "", errors.New("occurred_at required")
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		tzName := fallbackTZ
		if strings.TrimSpace(tzName) == "" {
			tzName = ts.Location().String()
			if tzName == "" || tzName == "Local" {
				_, offset := ts.Zone()
				tzName = offsetToTimezoneLabel(offset)
			}
		} else if loc, locErr := time.LoadLocation(fallbackTZ); locErr == nil {
			// Preserve declared import timezone when the RFC3339 offset matches it.
			_, gotOffset := ts.Zone()
			_, wantOffset := ts.In(loc).Zone()
			if gotOffset != wantOffset && !strings.HasSuffix(raw, "Z") {
				tzName = offsetToTimezoneLabel(gotOffset)
			}
		}
		return ts.UTC(), tzName, nil
	}
	loc, err := time.LoadLocation(fallbackTZ)
	if err != nil {
		return time.Time{}, "", err
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if ts, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return ts.UTC(), fallbackTZ, nil
		}
	}
	return time.Time{}, "", fmt.Errorf("unsupported occurred_at %q", raw)
}

func offsetToTimezoneLabel(offsetSeconds int) string {
	if offsetSeconds == 0 {
		return "UTC"
	}
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	h := offsetSeconds / 3600
	m := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("UTC%s%02d:%02d", sign, h, m)
}
