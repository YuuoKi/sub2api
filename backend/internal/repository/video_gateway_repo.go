package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type videoGatewayRepository struct{ db *sql.DB }

func NewVideoGatewayRepository(db *sql.DB) service.VideoGatewayRepository {
	return &videoGatewayRepository{db: db}
}

func NewVideoGatewayRuntimeRepository(db *sql.DB) service.VideoGatewayRuntimeRepository {
	return &videoGatewayRepository{db: db}
}

func (r *videoGatewayRepository) ListEnabledVideoProviders(ctx context.Context) ([]service.VideoProviderAccount, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id, provider, display_name, enabled, '', masked_key, '', default_model
		FROM video_provider_accounts WHERE enabled=TRUE ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.VideoProviderAccount, 0)
	for rows.Next() {
		var item service.VideoProviderAccount
		if err := rows.Scan(&item.ID, &item.Provider, &item.DisplayName, &item.Enabled, &item.EncryptedAPIKey, &item.MaskedKey, &item.BaseURL, &item.DefaultModel); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *videoGatewayRepository) GetVideoProvider(ctx context.Context, id int64) (*service.VideoProviderAccount, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	var item service.VideoProviderAccount
	err = db.QueryRowContext(ctx, `SELECT id, provider, display_name, enabled, encrypted_api_key, masked_key, base_url, default_model
		FROM video_provider_accounts WHERE id=$1`, id).Scan(&item.ID, &item.Provider, &item.DisplayName, &item.Enabled,
		&item.EncryptedAPIKey, &item.MaskedKey, &item.BaseURL, &item.DefaultModel)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	return &item, err
}

func (r *videoGatewayRepository) BeginRealDispatch(ctx context.Context, id, version int64) (bool, error) {
	db, err := r.requireDB()
	if err != nil {
		return false, err
	}
	result, err := db.ExecContext(ctx, `UPDATE video_tasks SET real_dispatch_count=1, dispatch_state='dispatching',
		version=version+1, updated_at=NOW() WHERE id=$1 AND version=$2 AND status='queued' AND real_dispatch_count=0`, id, version)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (r *videoGatewayRepository) MarkVideoSubmitted(ctx context.Context, id, version int64, upstreamTaskID string) error {
	db, err := r.requireDB()
	if err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `UPDATE video_tasks SET status='submitted', dispatch_state='accepted', upstream_task_id=$3,
		version=version+1, worker_claimed_at=NULL, worker_claimed_until=NULL, updated_at=NOW() WHERE id=$1 AND version=$2`, id, version, upstreamTaskID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return service.ErrVideoTaskTerminalConflict
	}
	return nil
}

func (r *videoGatewayRepository) CreateTask(ctx context.Context, task *service.VideoTask) error {
	db, err := r.requireDB()
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("video task is required")
	}
	if task.DurationSeconds == 0 {
		task.DurationSeconds = 4
	}
	if task.Resolution == "" {
		task.Resolution = "720p"
	}
	return db.QueryRowContext(ctx, `INSERT INTO video_tasks
		(provider_account_id, provider, model, task_type, prompt, status, creation_key, created_by,
		 api_key_id, group_id, duration_seconds, resolution)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,NULLIF($9,0),NULLIF($10,0),$11,$12)
		RETURNING id, version, created_at, updated_at`,
		task.ProviderAccountID, task.Provider, task.Model, task.TaskType, task.Prompt,
		task.Status, task.CreationKey, task.CreatedBy, task.APIKeyID, task.GroupID,
		task.DurationSeconds, task.Resolution,
	).Scan(&task.ID, &task.Version, &task.CreatedAt, &task.UpdatedAt)
}

const videoTaskColumns = `id, COALESCE(api_key_id, 0), COALESCE(group_id, 0), provider_account_id,
	provider, model, task_type, prompt, status, upstream_task_id, result_url, last_frame_url, duration_seconds,
	resolution, usage_total_tokens, cost_amount, currency, real_dispatch_count,
	provider_error_code, provider_error_message, error_message, COALESCE(creation_key, ''),
	version, dispatch_state, created_by, created_at, updated_at, completed_at`

func (r *videoGatewayRepository) GetTask(ctx context.Context, id int64) (*service.VideoTask, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	task, err := scanVideoTask(db.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	return task, err
}

func (r *videoGatewayRepository) GetTaskForScope(ctx context.Context, id int64, scope service.VideoTaskScope) (*service.VideoTask, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	task, err := scanVideoTask(db.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks
		WHERE id=$1 AND created_by=$2 AND api_key_id=$3 AND group_id=$4`, id, scope.UserID, scope.APIKeyID, scope.GroupID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	return task, err
}

func (r *videoGatewayRepository) CancelTaskForScope(ctx context.Context, id int64, scope service.VideoTaskScope) (*service.VideoTask, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	task, err := scanVideoTask(db.QueryRowContext(ctx, `UPDATE video_tasks SET status='cancelled', completed_at=NOW(),
		version=version+1, updated_at=NOW() WHERE id=$1 AND created_by=$2 AND api_key_id=$3 AND group_id=$4
		AND status IN ('queued','submitted','running') RETURNING `+videoTaskColumns, id, scope.UserID, scope.APIKeyID, scope.GroupID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	return task, err
}

func (r *videoGatewayRepository) ClaimRunnableTasks(ctx context.Context, limit int, lease time.Duration) ([]*service.VideoTask, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	seconds := int(lease / time.Second)
	if seconds <= 0 {
		return nil, fmt.Errorf("video task claim lease must be positive")
	}
	rows, err := db.QueryContext(ctx, `WITH candidates AS (
		SELECT id FROM video_tasks
		WHERE status IN ('queued','submitted','running')
		  AND (worker_claimed_until IS NULL OR worker_claimed_until <= NOW())
		ORDER BY updated_at, id LIMIT $1 FOR UPDATE SKIP LOCKED
	), claimed AS (
		UPDATE video_tasks task SET worker_claimed_at = NOW(),
		worker_claimed_until = NOW() + ($2::int * INTERVAL '1 second')
		FROM candidates WHERE task.id = candidates.id RETURNING task.*
	) SELECT `+videoTaskColumns+` FROM claimed`, limit, seconds)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var tasks []*service.VideoTask
	for rows.Next() {
		task, scanErr := scanVideoTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *videoGatewayRepository) FinalizeTask(ctx context.Context, input service.VideoTaskFinalization) (service.VideoTaskFinalizationResult, error) {
	db, err := r.requireDB()
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if input.Status != service.VideoStatusSucceeded && input.Status != service.VideoStatusFailed && input.Status != service.VideoStatusCancelled {
		return service.VideoTaskFinalizationResult{}, fmt.Errorf("video finalization requires terminal status")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var result service.VideoTaskFinalizationResult
	err = tx.QueryRowContext(ctx, `UPDATE video_tasks SET status=$3, result_url=$4, last_frame_url=$5,
		error_message=$6, usage_total_tokens=$7, cost_amount=$8, currency=$9,
		provider_error_code=$10, provider_error_message=$11, completed_at=$12, version=version+1,
		worker_claimed_at=NULL, worker_claimed_until=NULL, updated_at=NOW()
		WHERE id=$1 AND version=$2 AND status NOT IN ('succeeded','failed','cancelled')
		RETURNING status, version`, input.TaskID, input.ExpectedVersion, input.Status,
		input.ResultURL, input.LastFrameURL, input.ErrorMessage, input.UsageTotalTokens,
		input.CostAmount, input.Currency, input.ProviderErrorCode, input.ProviderErrorMessage,
		input.CompletedAt).Scan(&result.Status, &result.Version)
	if errors.Is(err, sql.ErrNoRows) {
		if readErr := tx.QueryRowContext(ctx, `SELECT status, version FROM video_tasks WHERE id=$1`, input.TaskID).Scan(&result.Status, &result.Version); readErr != nil {
			if errors.Is(readErr, sql.ErrNoRows) {
				return service.VideoTaskFinalizationResult{}, service.ErrVideoTaskNotFound
			}
			return service.VideoTaskFinalizationResult{}, readErr
		}
		if result.Status == input.Status {
			result.Idempotent = true
			return result, nil
		}
		return service.VideoTaskFinalizationResult{}, service.ErrVideoTaskTerminalConflict
	}
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO video_usage_logs (video_task_id, provider, model, status)
		SELECT id, provider, model, status FROM video_tasks WHERE id=$1
		ON CONFLICT (video_task_id) DO NOTHING`, input.TaskID); err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	result.Applied = true
	return result, nil
}

type videoRowScanner interface{ Scan(...any) error }

func (r *videoGatewayRepository) requireDB() (*sql.DB, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("video gateway repository database is required")
	}
	return r.db, nil
}

func scanVideoTask(scanner videoRowScanner) (*service.VideoTask, error) {
	task := &service.VideoTask{}
	var completed sql.NullTime
	var usage sql.NullInt64
	if err := scanner.Scan(&task.ID, &task.APIKeyID, &task.GroupID, &task.ProviderAccountID,
		&task.Provider, &task.Model, &task.TaskType, &task.Prompt, &task.Status, &task.UpstreamTaskID, &task.ResultURL,
		&task.LastFrameURL, &task.DurationSeconds, &task.Resolution, &usage, &task.CostAmount,
		&task.Currency, &task.RealDispatchCount, &task.ProviderErrorCode, &task.ProviderErrorMessage,
		&task.ErrorMessage,
		&task.CreationKey, &task.Version, &task.DispatchState, &task.CreatedBy,
		&task.CreatedAt, &task.UpdatedAt, &completed); err != nil {
		return nil, err
	}
	if usage.Valid {
		task.UsageTotalTokens = &usage.Int64
	}
	if completed.Valid {
		task.CompletedAt = &completed.Time
	}
	return task, nil
}
