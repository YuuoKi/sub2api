package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type providerBillingRepository struct {
	db *sql.DB
}

var _ service.ProviderBillingStore = (*providerBillingRepository)(nil)

func NewProviderBillingRepository(db *sql.DB) service.ProviderBillingStore {
	return &providerBillingRepository{db: db}
}

func (r *providerBillingRepository) HasFileSHA256(ctx context.Context, sha256hex string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM provider_billing_imports WHERE file_sha256 = $1)`, sha256hex).Scan(&exists)
	return exists, err
}

func (r *providerBillingRepository) HasProviderExternalLineID(ctx context.Context, provider, externalLineID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM provider_billing_lines
			WHERE provider = $1 AND external_line_id = $2
		)`, provider, externalLineID).Scan(&exists)
	return exists, err
}

func (r *providerBillingRepository) CreateImport(ctx context.Context, rec *service.ProviderBillingImportRecord) (*service.ProviderBillingImportRecord, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var invoice any
	if strings.TrimSpace(rec.InvoiceNumber) != "" {
		invoice = rec.InvoiceNumber
	}
	var createdBy any
	if rec.CreatedBy > 0 {
		createdBy = rec.CreatedBy
	}

	row := tx.QueryRowContext(ctx, `
		INSERT INTO provider_billing_imports (
			provider, provider_account_id, billing_period_start, billing_period_end,
			timezone, original_currency, source_type, invoice_number,
			file_sha256, storage_key, original_filename, byte_size, status, line_count, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, created_at`,
		rec.Provider, rec.ProviderAccountID, rec.BillingPeriodStart, rec.BillingPeriodEnd,
		rec.Timezone, rec.OriginalCurrency, rec.SourceType, invoice,
		rec.FileSHA256, rec.StorageKey, rec.OriginalFilename, rec.ByteSize, rec.Status, rec.LineCount, createdBy,
	)
	if err := row.Scan(&rec.ID, &rec.CreatedAt); err != nil {
		if isProviderBillingUniqueViolation(err) {
			return nil, service.ErrProviderBillingDuplicateFileSHA256
		}
		return nil, err
	}

	for _, line := range rec.Lines {
		normJSON, err := json.Marshal(line)
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO provider_billing_lines (
				import_id, provider, external_line_id, upstream_task_id, model, sku,
				usage_quantity, usage_unit, net_amount, tax_amount, gross_amount,
				currency, occurred_at, occurred_timezone, normalized_json
			) VALUES ($1,$2,$3,$4,$5,$6,$7::numeric,$8,$9::numeric,$10::numeric,$11::numeric,$12,$13,$14,$15::jsonb)`,
			rec.ID, rec.Provider, line.ExternalLineID, line.UpstreamTaskID, line.Model, line.SKU,
			line.UsageQuantity.String(), line.UsageUnit, line.NetAmount.String(), line.TaxAmount.String(), line.GrossAmount.String(),
			line.Currency, line.OccurredAt.UTC(), line.OccurredTimezone, string(normJSON),
		)
		if err != nil {
			if isProviderBillingUniqueViolation(err) {
				return nil, service.ErrProviderBillingDuplicateExternalLineID
			}
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *providerBillingRepository) GetImport(ctx context.Context, id int64) (*service.ProviderBillingImportRecord, error) {
	rec := &service.ProviderBillingImportRecord{}
	var invoice sql.NullString
	var createdBy sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, provider, provider_account_id, billing_period_start, billing_period_end,
		       timezone, original_currency, source_type, invoice_number,
		       file_sha256, storage_key, original_filename, byte_size, status, line_count,
		       COALESCE(created_by, 0), created_at
		FROM provider_billing_imports WHERE id = $1`, id).Scan(
		&rec.ID, &rec.Provider, &rec.ProviderAccountID, &rec.BillingPeriodStart, &rec.BillingPeriodEnd,
		&rec.Timezone, &rec.OriginalCurrency, &rec.SourceType, &invoice,
		&rec.FileSHA256, &rec.StorageKey, &rec.OriginalFilename, &rec.ByteSize, &rec.Status, &rec.LineCount,
		&createdBy, &rec.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if invoice.Valid {
		rec.InvoiceNumber = invoice.String
	}
	if createdBy.Valid {
		rec.CreatedBy = createdBy.Int64
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT external_line_id, upstream_task_id, model, sku,
		       usage_quantity::text, usage_unit, net_amount::text, tax_amount::text, gross_amount::text,
		       currency, occurred_at, occurred_timezone
		FROM provider_billing_lines WHERE import_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var line service.ProviderBillingNormalizedLine
		var usage, net, tax, gross string
		if err := rows.Scan(
			&line.ExternalLineID, &line.UpstreamTaskID, &line.Model, &line.SKU,
			&usage, &line.UsageUnit, &net, &tax, &gross,
			&line.Currency, &line.OccurredAt, &line.OccurredTimezone,
		); err != nil {
			return nil, err
		}
		line.UsageQuantity = decimal.RequireFromString(usage)
		line.NetAmount = decimal.RequireFromString(net)
		line.TaxAmount = decimal.RequireFromString(tax)
		line.GrossAmount = decimal.RequireFromString(gross)
		rec.Lines = append(rec.Lines, line)
	}
	return rec, rows.Err()
}

func (r *providerBillingRepository) ListImports(ctx context.Context, provider string, limit int) ([]service.ProviderBillingImportRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, provider, provider_account_id, billing_period_start, billing_period_end,
		       timezone, original_currency, source_type, COALESCE(invoice_number, ''),
		       file_sha256, storage_key, original_filename, byte_size, status, line_count,
		       COALESCE(created_by, 0), created_at
		FROM provider_billing_imports`
	args := []any{}
	if strings.TrimSpace(provider) != "" {
		query += ` WHERE provider = $1`
		args = append(args, provider)
		query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d`, len(args)+1)
		args = append(args, limit)
	} else {
		query += ` ORDER BY created_at DESC LIMIT $1`
		args = append(args, limit)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]service.ProviderBillingImportRecord, 0)
	for rows.Next() {
		var rec service.ProviderBillingImportRecord
		if err := rows.Scan(
			&rec.ID, &rec.Provider, &rec.ProviderAccountID, &rec.BillingPeriodStart, &rec.BillingPeriodEnd,
			&rec.Timezone, &rec.OriginalCurrency, &rec.SourceType, &rec.InvoiceNumber,
			&rec.FileSHA256, &rec.StorageKey, &rec.OriginalFilename, &rec.ByteSize, &rec.Status, &rec.LineCount,
			&rec.CreatedBy, &rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *providerBillingRepository) ReplaceMatches(ctx context.Context, importID int64, matches []service.ProviderBillingMatchResult) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM provider_reconciliation_matches WHERE import_id = $1`, importID); err != nil {
		return err
	}

	for _, match := range matches {
		var diff any
		if match.DiffJSON != nil {
			raw, err := json.Marshal(match.DiffJSON)
			if err != nil {
				return err
			}
			diff = string(raw)
		}
		var accountDay any
		if match.AccountDay != nil {
			accountDay = match.AccountDay.UTC().Format("2006-01-02")
		}
		var billingLineID any
		if match.BillingLineID > 0 {
			billingLineID = match.BillingLineID
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO provider_reconciliation_matches (
				import_id, billing_line_id, external_line_id, match_status, match_mode,
				internal_ref_type, internal_ref_id,
				provider_amount, internal_amount, provider_usage, internal_usage,
				currency, model, sku, account_day, diff_json
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::numeric,$9::numeric,$10::numeric,$11::numeric,$12,$13,$14,$15,$16::jsonb)`,
			importID, billingLineID, match.ExternalLineID, string(match.MatchStatus), string(match.MatchMode),
			nullIfEmptyBilling(match.InternalRefType), nullIfEmptyBilling(match.InternalRefID),
			match.ProviderAmount.String(), match.InternalAmount.String(),
			match.ProviderUsage.String(), match.InternalUsage.String(),
			nullIfEmptyBilling(match.Currency), match.Model, match.SKU, accountDay, diff,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *providerBillingRepository) ListMatches(ctx context.Context, importID int64, status string) ([]service.ProviderBillingMatchResult, error) {
	query := `
		SELECT m.id, m.import_id, COALESCE(m.billing_line_id, 0), COALESCE(m.external_line_id, ''),
		       m.match_status, m.match_mode, COALESCE(m.internal_ref_type, ''), COALESCE(m.internal_ref_id, ''),
		       COALESCE(m.provider_amount::text, '0'), COALESCE(m.internal_amount::text, '0'),
		       COALESCE(m.provider_usage::text, '0'), COALESCE(m.internal_usage::text, '0'),
		       COALESCE(m.currency, ''), m.model, m.sku, m.account_day, m.diff_json
		FROM provider_reconciliation_matches m
		WHERE m.import_id = $1`
	args := []any{importID}
	if strings.TrimSpace(status) != "" {
		query += ` AND m.match_status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY m.id`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]service.ProviderBillingMatchResult, 0)
	for rows.Next() {
		var match service.ProviderBillingMatchResult
		var providerAmount, internalAmount, providerUsage, internalUsage string
		var accountDay sql.NullTime
		var diffJSON []byte
		if err := rows.Scan(
			&match.ID, &match.ImportID, &match.BillingLineID, &match.ExternalLineID,
			&match.MatchStatus, &match.MatchMode, &match.InternalRefType, &match.InternalRefID,
			&providerAmount, &internalAmount, &providerUsage, &internalUsage,
			&match.Currency, &match.Model, &match.SKU, &accountDay, &diffJSON,
		); err != nil {
			return nil, err
		}
		match.ProviderAmount = decimal.RequireFromString(providerAmount)
		match.InternalAmount = decimal.RequireFromString(internalAmount)
		match.ProviderUsage = decimal.RequireFromString(providerUsage)
		match.InternalUsage = decimal.RequireFromString(internalUsage)
		if accountDay.Valid {
			day := accountDay.Time.UTC()
			match.AccountDay = &day
		}
		if len(diffJSON) > 0 {
			_ = json.Unmarshal(diffJSON, &match.DiffJSON)
		}
		out = append(out, match)
	}
	return out, rows.Err()
}

func (r *providerBillingRepository) FindVideoTaskByUpstreamID(ctx context.Context, upstreamTaskID string) (*service.ProviderBillingInternalTask, error) {
	var (
		id          int64
		modelName   string
		costText    string
		createdAt   time.Time
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(model, ''), cost_estimate::text, created_at
		FROM video_tasks
		WHERE upstream_task_id = $1
		ORDER BY id DESC LIMIT 1`, upstreamTaskID).Scan(&id, &modelName, &costText, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &service.ProviderBillingInternalTask{
		RefType:        "video_task",
		RefID:          fmt.Sprintf("%d", id),
		UpstreamTaskID: upstreamTaskID,
		Amount:         decimal.RequireFromString(costText),
		Usage:          decimal.NewFromInt(1),
		Currency:       "USD",
		Model:          modelName,
		SKU:            "",
		AccountDay:     createdAt.UTC().Truncate(24 * time.Hour),
	}, nil
}

func (r *providerBillingRepository) FindBatchImageJobByProviderJobName(ctx context.Context, providerJobName string) (*service.ProviderBillingInternalTask, error) {
	var (
		batchID    string
		modelName  string
		costText   sql.NullString
		imageCount int
		createdAt  time.Time
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT batch_id, COALESCE(model, ''), actual_cost::text, COALESCE(success_count, 0), created_at
		FROM batch_image_jobs
		WHERE provider_job_name = $1
		ORDER BY id DESC LIMIT 1`, providerJobName).Scan(&batchID, &modelName, &costText, &imageCount, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	amount := decimal.Zero
	if costText.Valid && strings.TrimSpace(costText.String) != "" {
		amount = decimal.RequireFromString(costText.String)
	}
	usage := decimal.NewFromInt(int64(imageCount))
	if imageCount <= 0 {
		usage = decimal.NewFromInt(1)
	}
	return &service.ProviderBillingInternalTask{
		RefType:        "batch_image_job",
		RefID:          batchID,
		UpstreamTaskID: providerJobName,
		Amount:         amount,
		Usage:          usage,
		Currency:       "USD",
		Model:          modelName,
		SKU:            "image",
		AccountDay:     createdAt.UTC().Truncate(24 * time.Hour),
	}, nil
}

func (r *providerBillingRepository) ListInternalTasksForPeriod(ctx context.Context, provider, _ string, start, end time.Time) ([]service.ProviderBillingInternalTask, error) {
	out := make([]service.ProviderBillingInternalTask, 0)
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "seedance":
		rows, err := r.db.QueryContext(ctx, `
			SELECT id, upstream_task_id, COALESCE(model, ''), cost_estimate::text, created_at
			FROM video_tasks
			WHERE created_at >= $1 AND created_at <= $2
			  AND upstream_task_id <> ''
			ORDER BY id`, start, end)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var upstream, modelName, costText string
			var createdAt time.Time
			if err := rows.Scan(&id, &upstream, &modelName, &costText, &createdAt); err != nil {
				return nil, err
			}
			out = append(out, service.ProviderBillingInternalTask{
				RefType:        "video_task",
				RefID:          fmt.Sprintf("%d", id),
				UpstreamTaskID: upstream,
				Amount:         decimal.RequireFromString(costText),
				Usage:          decimal.NewFromInt(1),
				Currency:       "USD",
				Model:          modelName,
				AccountDay:     createdAt.UTC().Truncate(24 * time.Hour),
			})
		}
		return out, rows.Err()
	case "gemini":
		rows, err := r.db.QueryContext(ctx, `
			SELECT batch_id, COALESCE(provider_job_name, ''), COALESCE(model, ''),
			       COALESCE(actual_cost::text, '0'), COALESCE(success_count, 0), created_at
			FROM batch_image_jobs
			WHERE created_at >= $1 AND created_at <= $2
			  AND provider_job_name IS NOT NULL AND provider_job_name <> ''
			ORDER BY id`, start, end)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var batchID, upstream, modelName, costText string
			var imageCount int
			var createdAt time.Time
			if err := rows.Scan(&batchID, &upstream, &modelName, &costText, &imageCount, &createdAt); err != nil {
				return nil, err
			}
			usage := decimal.NewFromInt(int64(imageCount))
			if imageCount <= 0 {
				usage = decimal.NewFromInt(1)
			}
			out = append(out, service.ProviderBillingInternalTask{
				RefType:        "batch_image_job",
				RefID:          batchID,
				UpstreamTaskID: upstream,
				Amount:         decimal.RequireFromString(costText),
				Usage:          usage,
				Currency:       "USD",
				Model:          modelName,
				SKU:            "image",
				AccountDay:     createdAt.UTC().Truncate(24 * time.Hour),
			})
		}
		return out, rows.Err()
	default:
		return out, nil
	}
}

func (r *providerBillingRepository) PeriodSummary(ctx context.Context, start, end time.Time) ([]service.ProviderBillingPeriodSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.provider, i.provider_account_id,
		       i.billing_period_start, i.billing_period_end,
		       COUNT(DISTINCT i.id) AS import_count,
		       COUNT(*) FILTER (WHERE m.match_status = 'matched') AS matched,
		       COUNT(*) FILTER (WHERE m.match_status IS NOT NULL AND m.match_status <> 'matched') AS has_diff,
		       COUNT(*) FILTER (WHERE m.match_status = 'provider_only') AS provider_only,
		       COUNT(*) FILTER (WHERE m.match_status = 'internal_only') AS internal_only
		FROM provider_billing_imports i
		LEFT JOIN provider_reconciliation_matches m ON m.import_id = i.id
		WHERE i.billing_period_start <= $2 AND i.billing_period_end >= $1
		GROUP BY i.provider, i.provider_account_id, i.billing_period_start, i.billing_period_end
		ORDER BY i.billing_period_start DESC`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]service.ProviderBillingPeriodSummary, 0)
	for rows.Next() {
		var s service.ProviderBillingPeriodSummary
		var periodStart, periodEnd time.Time
		if err := rows.Scan(
			&s.Provider, &s.ProviderAccountID, &periodStart, &periodEnd,
			&s.ImportCount, &s.Matched, &s.HasDiff, &s.ProviderOnly, &s.InternalOnly,
		); err != nil {
			return nil, err
		}
		s.BillingPeriodStart = periodStart.UTC().Format(time.RFC3339)
		s.BillingPeriodEnd = periodEnd.UTC().Format(time.RFC3339)
		if s.ImportCount == 0 {
			s.Conclusion = "not_uploaded"
		} else if s.HasDiff > 0 {
			s.Conclusion = "has_diff"
		} else {
			s.Conclusion = "reconciled"
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *providerBillingRepository) BossConclusions(ctx context.Context) ([]service.ProviderBillingPeriodSummary, error) {
	// Latest calendar month window for boss homepage strip.
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	summaries, err := r.PeriodSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return []service.ProviderBillingPeriodSummary{{
			BillingPeriodStart: start.Format(time.RFC3339),
			BillingPeriodEnd:   end.Format(time.RFC3339),
			Conclusion:         "not_uploaded",
		}}, nil
	}
	return summaries, nil
}

func nullIfEmptyBilling(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func isProviderBillingUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}