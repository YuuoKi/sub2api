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
)

type videoGatewayRepository struct {
	db *sql.DB
}

func NewVideoGatewayRepository(db *sql.DB) service.VideoGatewayRepository {
	return &videoGatewayRepository{db: db}
}

const insertVideoUsageLogSQL = `
	WITH usage_log_insert_lock AS (
		SELECT pg_advisory_xact_lock($1)
	)
	INSERT INTO video_usage_logs (video_task_id, provider, model, status, cost_estimate, duration)
	SELECT $1,$2,$3,$4,$5,$6
	FROM usage_log_insert_lock
	WHERE NOT EXISTS (
		SELECT 1
		FROM video_usage_logs
		WHERE video_task_id = $1
	)
`

const videoTaskClaimLeaseSeconds = 120

const videoTaskSelectColumns = `
		vt.id, vt.provider_account_id, vt.provider, vt.model, vt.task_type, vt.prompt, vt.negative_prompt,
		vt.reference_image_url, vt.reference_video_url, vt.aspect_ratio, vt.duration, vt.resolution,
		vt.status, vt.upstream_task_id, vt.result_url, vt.error_message, vt.cost_estimate, vt.poll_count,
		vt.created_by, vt.created_at, vt.updated_at, vt.completed_at,
		COALESCE(vpa.display_name, ''), COALESCE(u.email, ''), COALESCE(u.username, '')
`

const videoTaskJoinSQL = `
		FROM video_tasks vt
		LEFT JOIN video_provider_accounts vpa ON vpa.id = vt.provider_account_id
		LEFT JOIN users u ON u.id = vt.created_by
`

const createVideoTaskSQL = `
	INSERT INTO video_tasks (
		provider_account_id, provider, model, task_type, prompt, negative_prompt,
		reference_image_url, reference_video_url, aspect_ratio, duration, resolution,
		status, created_by
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	RETURNING id, created_at, updated_at
`

func (r *videoGatewayRepository) CreateProviderAccount(ctx context.Context, account *service.VideoProviderAccount) error {
	metadata, err := marshalJSONMap(account.Metadata)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO video_provider_accounts (
			provider, display_name, enabled, encrypted_api_key, masked_key,
			base_url, default_model, rate_limit_per_minute, metadata_json
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)
		RETURNING id, created_at, updated_at
	`
	err = r.db.QueryRowContext(ctx, q,
		account.Provider,
		account.DisplayName,
		account.Enabled,
		account.EncryptedAPIKey,
		account.MaskedKey,
		account.BaseURL,
		account.DefaultModel,
		account.RateLimitPerMinute,
		string(metadata),
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *videoGatewayRepository) GetProviderAccount(ctx context.Context, id int64) (*service.VideoProviderAccount, error) {
	const q = `
		SELECT id, provider, display_name, enabled, encrypted_api_key, masked_key,
		       base_url, default_model, rate_limit_per_minute, metadata_json, created_at, updated_at
		FROM video_provider_accounts
		WHERE id = $1
	`
	account, err := scanVideoProvider(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrVideoProviderNotFound, nil)
	}
	return account, nil
}

func (r *videoGatewayRepository) ListProviderAccounts(ctx context.Context) ([]*service.VideoProviderAccount, error) {
	const q = `
		SELECT id, provider, display_name, enabled, encrypted_api_key, masked_key,
		       base_url, default_model, rate_limit_per_minute, metadata_json, created_at, updated_at
		FROM video_provider_accounts
		ORDER BY
			CASE provider WHEN 'mock' THEN 1 WHEN 'seedance' THEN 2 WHEN 'kling' THEN 3 ELSE 99 END,
			id
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*service.VideoProviderAccount
	for rows.Next() {
		item, err := scanVideoProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *videoGatewayRepository) UpdateProviderAccount(ctx context.Context, account *service.VideoProviderAccount) error {
	metadata, err := marshalJSONMap(account.Metadata)
	if err != nil {
		return err
	}
	const q = `
		UPDATE video_provider_accounts
		SET display_name = $2,
		    enabled = $3,
		    encrypted_api_key = $4,
		    masked_key = $5,
		    base_url = $6,
		    default_model = $7,
		    rate_limit_per_minute = $8,
		    metadata_json = $9::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	err = r.db.QueryRowContext(ctx, q,
		account.ID,
		account.DisplayName,
		account.Enabled,
		account.EncryptedAPIKey,
		account.MaskedKey,
		account.BaseURL,
		account.DefaultModel,
		account.RateLimitPerMinute,
		string(metadata),
	).Scan(&account.UpdatedAt)
	if err != nil {
		return translatePersistenceError(err, service.ErrVideoProviderNotFound, nil)
	}
	return nil
}

func (r *videoGatewayRepository) CreateTask(ctx context.Context, task *service.VideoTask) error {
	return scanCreatedVideoTask(r.db.QueryRowContext(ctx, createVideoTaskSQL,
		task.ProviderAccountID,
		task.Provider,
		task.Model,
		task.TaskType,
		task.Prompt,
		task.NegativePrompt,
		task.ReferenceImageURL,
		task.ReferenceVideoURL,
		task.AspectRatio,
		task.Duration,
		task.Resolution,
		task.Status,
		task.CreatedBy,
	), task)
}

func (r *videoGatewayRepository) CreateDailyTrialTask(ctx context.Context, task *service.VideoTask, provider string, createdBy int64, trialDate time.Time) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	var reservationID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO video_daily_trial_reservations (provider, created_by, trial_date)
		VALUES ($1,$2,$3)
		ON CONFLICT (provider, created_by, trial_date) DO NOTHING
		RETURNING id
	`, provider, createdBy, trialDate).Scan(&reservationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	if err := scanCreatedVideoTask(tx.QueryRowContext(ctx, createVideoTaskSQL,
		task.ProviderAccountID,
		task.Provider,
		task.Model,
		task.TaskType,
		task.Prompt,
		task.NegativePrompt,
		task.ReferenceImageURL,
		task.ReferenceVideoURL,
		task.AspectRatio,
		task.Duration,
		task.Resolution,
		task.Status,
		task.CreatedBy,
	), task); err != nil {
		return false, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE video_daily_trial_reservations
		SET video_task_id = $2
		WHERE id = $1
	`, reservationID, task.ID); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *videoGatewayRepository) GetTask(ctx context.Context, id int64) (*service.VideoTask, error) {
	q := "SELECT" + videoTaskSelectColumns + videoTaskJoinSQL + " WHERE vt.id = $1"
	task, err := scanVideoTask(r.db.QueryRowContext(ctx, q, id))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrVideoTaskNotFound, nil)
	}
	return task, nil
}

func (r *videoGatewayRepository) ListTasks(ctx context.Context, params service.VideoTaskListParams) ([]*service.VideoTask, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	if strings.TrimSpace(params.Status) != "" {
		args = append(args, params.Status)
		where = append(where, fmt.Sprintf("vt.status = $%d", len(args)))
	}
	if strings.TrimSpace(params.Provider) != "" {
		args = append(args, params.Provider)
		where = append(where, fmt.Sprintf("vt.provider = $%d", len(args)))
	}
	if !params.IsAdmin {
		args = append(args, params.CreatedBy)
		where = append(where, fmt.Sprintf("vt.created_by = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	countQ := "SELECT COUNT(*) FROM video_tasks vt WHERE " + whereSQL
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	args = append(args, pageSize, (page-1)*pageSize)
	q := "SELECT" + videoTaskSelectColumns + videoTaskJoinSQL + " WHERE " + whereSQL + fmt.Sprintf(`
		ORDER BY vt.created_at DESC, vt.id DESC
		LIMIT $%d OFFSET $%d
	`, len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out, err := scanVideoTaskRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *videoGatewayRepository) ListRunnableTasks(ctx context.Context, limit int) ([]*service.VideoTask, error) {
	if limit <= 0 {
		limit = 20
	}
	q := `
		WITH candidate_ids AS (
			SELECT vt.id
			FROM video_tasks vt
			WHERE vt.status IN ('queued', 'submitted', 'running')
			  AND (vt.worker_claimed_until IS NULL OR vt.worker_claimed_until <= NOW())
			ORDER BY vt.updated_at ASC, vt.id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		),
		claimed AS (
			UPDATE video_tasks vt
			SET worker_claimed_at = NOW(),
			    worker_claimed_until = NOW() + ($2::int * INTERVAL '1 second')
			FROM candidate_ids c
			WHERE vt.id = c.id
			RETURNING vt.*
		)
		SELECT` + videoTaskSelectColumns + `
		FROM claimed vt
		LEFT JOIN video_provider_accounts vpa ON vpa.id = vt.provider_account_id
		LEFT JOIN users u ON u.id = vt.created_by
		ORDER BY vt.updated_at ASC, vt.id ASC
	`
	rows, err := r.db.QueryContext(ctx, q, limit, videoTaskClaimLeaseSeconds)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanVideoTaskRows(rows)
}

func (r *videoGatewayRepository) ClaimTaskForSubmit(ctx context.Context, taskID int64) (bool, error) {
	const q = `
		UPDATE video_tasks
		SET status = $2,
		    updated_at = NOW(),
		    worker_claimed_at = COALESCE(worker_claimed_at, NOW()),
		    worker_claimed_until = NOW() + ($4::int * INTERVAL '1 second')
		WHERE id = $1 AND status = $3
		RETURNING id
	`
	var claimedID int64
	err := r.db.QueryRowContext(ctx, q, taskID, service.VideoStatusSubmitted, service.VideoStatusQueued, videoTaskClaimLeaseSeconds).Scan(&claimedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return claimedID > 0, nil
}

func (r *videoGatewayRepository) UpdateTask(ctx context.Context, task *service.VideoTask) error {
	const q = `
		UPDATE video_tasks
		SET status = $2,
		    upstream_task_id = $3,
		    result_url = $4,
		    error_message = $5,
		    cost_estimate = $6,
		    poll_count = $8,
		    worker_claimed_at = NULL,
		    worker_claimed_until = NULL,
		    updated_at = NOW(),
		    completed_at = $7
		WHERE id = $1
		RETURNING updated_at
	`
	var completedAt any
	if task.CompletedAt != nil {
		completedAt = *task.CompletedAt
	}
	if err := r.db.QueryRowContext(ctx, q,
		task.ID,
		task.Status,
		task.UpstreamTaskID,
		task.ResultURL,
		task.ErrorMessage,
		task.CostEstimate,
		completedAt,
		task.PollCount,
	).Scan(&task.UpdatedAt); err != nil {
		return translatePersistenceError(err, service.ErrVideoTaskNotFound, nil)
	}
	return nil
}

func (r *videoGatewayRepository) AddTaskEvent(ctx context.Context, event *service.VideoTaskEvent) error {
	payload, err := marshalJSONMap(event.Payload)
	if err != nil {
		return err
	}
	const q = `
		INSERT INTO video_task_events (video_task_id, event_type, message, payload_json)
		VALUES ($1,$2,$3,$4::jsonb)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, q,
		event.VideoTaskID,
		event.EventType,
		event.Message,
		string(payload),
	).Scan(&event.ID, &event.CreatedAt)
}

func (r *videoGatewayRepository) ListTaskEvents(ctx context.Context, taskID int64, limit int) ([]*service.VideoTaskEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
		SELECT id, video_task_id, event_type, message, payload_json, created_at
		FROM video_task_events
		WHERE video_task_id = $1
		ORDER BY created_at ASC, id ASC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, q, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*service.VideoTaskEvent
	for rows.Next() {
		var payload []byte
		event := &service.VideoTaskEvent{}
		if err := rows.Scan(&event.ID, &event.VideoTaskID, &event.EventType, &event.Message, &payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		event.Payload = unmarshalJSONMap(payload)
		out = append(out, event)
	}
	return out, rows.Err()
}

func (r *videoGatewayRepository) InsertUsageLog(ctx context.Context, task *service.VideoTask) error {
	_, err := r.db.ExecContext(ctx, insertVideoUsageLogSQL,
		task.ID,
		task.Provider,
		task.Model,
		task.Status,
		task.CostEstimate,
		task.Duration,
	)
	return err
}

func (r *videoGatewayRepository) CountTasksSince(ctx context.Context, since time.Time) (map[string]int64, error) {
	const q = `
		SELECT status, COUNT(*)
		FROM video_tasks
		WHERE created_at >= $1
		GROUP BY status
	`
	rows, err := r.db.QueryContext(ctx, q, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, rows.Err()
}

func (r *videoGatewayRepository) CountProviderTasksSince(ctx context.Context, since time.Time) (map[string]map[string]int64, error) {
	const q = `
		SELECT provider, status, COUNT(*)
		FROM video_tasks
		WHERE created_at >= $1
		GROUP BY provider, status
	`
	rows, err := r.db.QueryContext(ctx, q, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]map[string]int64{}
	for rows.Next() {
		var provider, status string
		var count int64
		if err := rows.Scan(&provider, &status, &count); err != nil {
			return nil, err
		}
		if out[provider] == nil {
			out[provider] = map[string]int64{}
		}
		out[provider][status] = count
	}
	return out, rows.Err()
}

func (r *videoGatewayRepository) ProviderAccountTaskStatsSince(ctx context.Context, since time.Time) (map[int64]service.VideoProviderRuntimeStats, error) {
	const statsQ = `
		SELECT provider_account_id,
		       COUNT(*) FILTER (WHERE created_at >= $1),
		       COUNT(*) FILTER (WHERE status IN ('queued', 'submitted', 'running')),
		       COUNT(*) FILTER (WHERE status = 'failed' AND created_at >= $1)
		FROM video_tasks
		GROUP BY provider_account_id
	`
	rows, err := r.db.QueryContext(ctx, statsQ, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]service.VideoProviderRuntimeStats{}
	for rows.Next() {
		var providerAccountID int64
		var item service.VideoProviderRuntimeStats
		if err := rows.Scan(&providerAccountID, &item.TodayTasks, &item.CurrentInflight, &item.TodayFailures); err != nil {
			return nil, err
		}
		out[providerAccountID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	const errorsQ = `
		SELECT DISTINCT ON (provider_account_id) provider_account_id, error_message, updated_at
		FROM video_tasks
		WHERE status = 'failed' AND COALESCE(error_message, '') <> ''
		ORDER BY provider_account_id, updated_at DESC, id DESC
	`
	errorRows, err := r.db.QueryContext(ctx, errorsQ)
	if err != nil {
		return nil, err
	}
	defer func() { _ = errorRows.Close() }()
	for errorRows.Next() {
		var providerAccountID int64
		var lastError string
		var lastErrorAt time.Time
		if err := errorRows.Scan(&providerAccountID, &lastError, &lastErrorAt); err != nil {
			return nil, err
		}
		item := out[providerAccountID]
		item.LastError = lastError
		item.LastErrorAt = &lastErrorAt
		out[providerAccountID] = item
	}
	return out, errorRows.Err()
}

func (r *videoGatewayRepository) ListRecentTasksByStatus(ctx context.Context, status string, limit int) ([]*service.VideoTask, error) {
	if limit <= 0 {
		limit = 8
	}
	q := "SELECT" + videoTaskSelectColumns + videoTaskJoinSQL + `
		WHERE vt.status = $1
		ORDER BY vt.updated_at DESC, vt.id DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, q, status, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanVideoTaskRows(rows)
}

func (r *videoGatewayRepository) UsageSummarySince(ctx context.Context, since time.Time) ([]service.VideoUsageSummary, error) {
	const q = `
		SELECT provider, model, status, COUNT(*), COALESCE(SUM(cost_estimate), 0), COALESCE(SUM(duration), 0)
		FROM video_usage_logs
		WHERE created_at >= $1
		GROUP BY provider, model, status
		ORDER BY COUNT(*) DESC, provider, model
		LIMIT 20
	`
	rows, err := r.db.QueryContext(ctx, q, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.VideoUsageSummary
	for rows.Next() {
		var item service.VideoUsageSummary
		if err := rows.Scan(&item.Provider, &item.Model, &item.Status, &item.Count, &item.CostEstimate, &item.Duration); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCreatedVideoTask(row scanner, task *service.VideoTask) error {
	return row.Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
}

func scanVideoProvider(row scanner) (*service.VideoProviderAccount, error) {
	var metadata []byte
	account := &service.VideoProviderAccount{}
	err := row.Scan(
		&account.ID,
		&account.Provider,
		&account.DisplayName,
		&account.Enabled,
		&account.EncryptedAPIKey,
		&account.MaskedKey,
		&account.BaseURL,
		&account.DefaultModel,
		&account.RateLimitPerMinute,
		&metadata,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	account.Metadata = unmarshalJSONMap(metadata)
	account.APIKeyConfigured = strings.TrimSpace(account.EncryptedAPIKey) != "" || strings.TrimSpace(account.MaskedKey) != ""
	return account, nil
}

func scanVideoTaskRows(rows *sql.Rows) ([]*service.VideoTask, error) {
	var out []*service.VideoTask
	for rows.Next() {
		task, err := scanVideoTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

func scanVideoTask(row scanner) (*service.VideoTask, error) {
	task := &service.VideoTask{}
	var completed sql.NullTime
	err := row.Scan(
		&task.ID,
		&task.ProviderAccountID,
		&task.Provider,
		&task.Model,
		&task.TaskType,
		&task.Prompt,
		&task.NegativePrompt,
		&task.ReferenceImageURL,
		&task.ReferenceVideoURL,
		&task.AspectRatio,
		&task.Duration,
		&task.Resolution,
		&task.Status,
		&task.UpstreamTaskID,
		&task.ResultURL,
		&task.ErrorMessage,
		&task.CostEstimate,
		&task.PollCount,
		&task.CreatedBy,
		&task.CreatedAt,
		&task.UpdatedAt,
		&completed,
		&task.ProviderAccountName,
		&task.CreatedByEmail,
		&task.CreatedByName,
	)
	if err != nil {
		return nil, err
	}
	if completed.Valid {
		task.CompletedAt = &completed.Time
	}
	return task, nil
}

func marshalJSONMap(in map[string]any) ([]byte, error) {
	if in == nil {
		in = map[string]any{}
	}
	return json.Marshal(in)
}

func unmarshalJSONMap(data []byte) map[string]any {
	if len(data) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}
