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

func (r *videoGatewayRepository) CreateTask(ctx context.Context, task *service.VideoTask) error {
	db, err := r.requireDB()
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("video task is required")
	}
	return db.QueryRowContext(ctx, `INSERT INTO video_tasks
		(provider_account_id, provider, model, task_type, prompt, status, creation_key, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8)
		RETURNING id, version, created_at, updated_at`,
		task.ProviderAccountID, task.Provider, task.Model, task.TaskType, task.Prompt,
		task.Status, task.CreationKey, task.CreatedBy,
	).Scan(&task.ID, &task.Version, &task.CreatedAt, &task.UpdatedAt)
}

const videoTaskColumns = `id, provider_account_id, provider, model, task_type, prompt,
	status, result_url, error_message, COALESCE(creation_key, ''), version, dispatch_state,
	created_by, created_at, updated_at, completed_at`

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
	err = tx.QueryRowContext(ctx, `UPDATE video_tasks SET status=$3, result_url=$4,
		error_message=$5, completed_at=$6, version=version+1,
		worker_claimed_at=NULL, worker_claimed_until=NULL, updated_at=NOW()
		WHERE id=$1 AND version=$2 AND status NOT IN ('succeeded','failed','cancelled')
		RETURNING status, version`, input.TaskID, input.ExpectedVersion, input.Status,
		input.ResultURL, input.ErrorMessage, input.CompletedAt).Scan(&result.Status, &result.Version)
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
	if err := scanner.Scan(&task.ID, &task.ProviderAccountID, &task.Provider, &task.Model,
		&task.TaskType, &task.Prompt, &task.Status, &task.ResultURL, &task.ErrorMessage,
		&task.CreationKey, &task.Version, &task.DispatchState, &task.CreatedBy,
		&task.CreatedAt, &task.UpdatedAt, &completed); err != nil {
		return nil, err
	}
	if completed.Valid {
		task.CompletedAt = &completed.Time
	}
	return task, nil
}
