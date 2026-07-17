package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *videoGatewayRepository) GetOrCreateMockProviderAccount(ctx context.Context) (*service.VideoProviderAccount, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	const selectQuery = `
SELECT id, COALESCE(group_id, 0), provider, display_name, enabled, encrypted_api_key, masked_key, base_url, default_model
FROM video_provider_accounts
WHERE provider = $1
ORDER BY id
LIMIT 1`
	item, err := scanMockProvider(db.QueryRowContext(ctx, selectQuery, service.VideoProviderMock))
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	const insertQuery = `
INSERT INTO video_provider_accounts (
    provider, display_name, enabled, encrypted_api_key, masked_key, base_url, default_model, metadata_json
) VALUES ($1, $2, TRUE, '', '', '', $3, '{"pricing_source":"internal_simulation","pricing_version":"simulation-v1"}'::jsonb)
ON CONFLICT (provider, display_name) DO NOTHING
RETURNING id, COALESCE(group_id, 0), provider, display_name, enabled, encrypted_api_key, masked_key, base_url, default_model`
	item, err = scanMockProvider(db.QueryRowContext(ctx, insertQuery,
		service.VideoProviderMock, "Internal Mock Video", service.VideoModelMockVideoV1))
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	item, err = scanMockProvider(db.QueryRowContext(ctx, selectQuery, service.VideoProviderMock))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoProviderNotFound
	}
	return item, err
}

func scanMockProvider(row *sql.Row) (*service.VideoProviderAccount, error) {
	var item service.VideoProviderAccount
	err := row.Scan(&item.ID, &item.GroupID, &item.Provider, &item.DisplayName, &item.Enabled,
		&item.EncryptedAPIKey, &item.MaskedKey, &item.BaseURL, &item.DefaultModel)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *videoGatewayRepository) CreateSimulationTask(ctx context.Context, task *service.VideoTask) (bool, error) {
	db, err := r.requireDB()
	if err != nil {
		return false, err
	}
	if task == nil {
		return false, fmt.Errorf("video simulation task is required")
	}
	if task.CreationKey != "" {
		existing, getErr := scanVideoTask(db.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE creation_key = $1`, task.CreationKey))
		if getErr == nil && existing != nil {
			if !simulationCreationKeyOwnedByCaller(existing, task) {
				return false, service.ErrVideoSimulationCreationKeyConflict
			}
			*task = *existing
			return false, nil
		}
		if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
			return false, getErr
		}
	}
	if task.Provider == "" {
		task.Provider = service.VideoProviderMock
	}
	if task.Model == "" {
		task.Model = service.VideoModelMockVideoV1
	}
	if task.TaskType == "" {
		task.TaskType = service.VideoSimulationTaskTypeTextToVideo
	}
	if task.Status == "" {
		task.Status = service.VideoStatusQueued
	}
	if task.DurationSeconds == 0 {
		task.DurationSeconds = service.VideoSimulationDurationSeconds
	}
	if task.Resolution == "" {
		task.Resolution = service.VideoSimulationResolution
	}
	if task.Currency == "" {
		task.Currency = "USD"
	}
	if task.PricingSource == "" {
		task.PricingSource = service.VideoPricingSourceInternalSimulation
	}
	if task.PricingVersion == "" {
		task.PricingVersion = service.VideoPricingVersionSimulationV1
	}
	if task.ReservationState == "" {
		task.ReservationState = service.VideoReservationNone
	}
	if task.DispatchState == "" {
		task.DispatchState = "pending"
	}
	err = db.QueryRowContext(ctx, `INSERT INTO video_tasks
		(provider_account_id, provider, model, task_type, prompt, status, creation_key, created_by,
		 api_key_id, group_id, duration_seconds, resolution, cost_amount, reserved_cost_usd, reservation_state,
		 currency, pricing_source, pricing_version,
		 pricing_cny_per_million_completion_tokens, pricing_usd_cny_exchange_rate, pricing_maximum_cny,
		 dispatch_state)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10,$11,$12,0,0,$13,$14,NULLIF($15,''),NULLIF($16,''),NULL,NULL,NULL,$17)
		RETURNING id, version, created_at, updated_at`,
		task.ProviderAccountID, task.Provider, task.Model, task.TaskType, task.Prompt,
		task.Status, task.CreationKey, task.CreatedBy, task.APIKeyID, task.GroupID,
		task.DurationSeconds, task.Resolution, task.ReservationState, task.Currency,
		task.PricingSource, task.PricingVersion, task.DispatchState,
	).Scan(&task.ID, &task.Version, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if task.CreationKey != "" {
			existing, getErr := scanVideoTask(db.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE creation_key = $1`, task.CreationKey))
			if getErr == nil && existing != nil {
				if !simulationCreationKeyOwnedByCaller(existing, task) {
					return false, service.ErrVideoSimulationCreationKeyConflict
				}
				*task = *existing
				return false, nil
			}
		}
		return false, err
	}
	task.CostAmount = 0
	task.ReservedCostUSD = 0
	task.PricingCNYPerMillionCompletionTokens = nil
	task.PricingUSDCNYExchangeRate = nil
	task.PricingMaximumCNY = nil
	return true, nil
}

func simulationCreationKeyOwnedByCaller(existing, incoming *service.VideoTask) bool {
	if existing == nil || incoming == nil {
		return false
	}
	return existing.Provider == service.VideoProviderMock &&
		existing.CreatedBy == incoming.CreatedBy &&
		existing.APIKeyID == incoming.APIKeyID
}

func (r *videoGatewayRepository) GetSimulationTaskForOwner(ctx context.Context, taskID, userID int64) (*service.VideoTask, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	task, err := scanVideoTask(db.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE id = $1 AND provider = $2`, taskID, service.VideoProviderMock))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	if task.CreatedBy != userID {
		return nil, service.ErrVideoTaskForbidden
	}
	return task, nil
}

func (r *videoGatewayRepository) CancelSimulationTaskForOwner(ctx context.Context, taskID, userID int64) (*service.VideoTask, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	task, err := scanVideoTask(tx.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE id = $1 AND provider = $2 FOR UPDATE`, taskID, service.VideoProviderMock))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	if task.CreatedBy != userID {
		return nil, service.ErrVideoTaskForbidden
	}
	switch task.Status {
	case service.VideoStatusSucceeded, service.VideoStatusFailed, service.VideoStatusCancelled:
		return nil, service.ErrVideoCancelConflict
	}
	updated, err := scanVideoTask(tx.QueryRowContext(ctx, `UPDATE video_tasks
		SET status = $2, version = version + 1, completed_at = NOW(), updated_at = NOW(),
		    worker_claimed_until = NULL
		WHERE id = $1 AND provider = $3
		RETURNING `+videoTaskColumns, taskID, service.VideoStatusCancelled, service.VideoProviderMock))
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO video_task_events (video_task_id, event_type, payload_json)
		VALUES ($1, $2, '{}'::jsonb)`, taskID, service.VideoStatusCancelled); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *videoGatewayRepository) ListSimulationTasksForOwner(ctx context.Context, userID int64) ([]*service.VideoTask, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT `+videoTaskColumns+`
		FROM video_tasks
		WHERE created_by = $1 AND provider = $2
		ORDER BY id DESC`, userID, service.VideoProviderMock)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.VideoTask, 0)
	for rows.Next() {
		task, scanErr := scanVideoTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

// mockSucceededContentReclaimSeconds bounds how long a succeeded mock task
// without ai_generation_content may be reclaimed for fail-open capture retry.
const mockSucceededContentReclaimSeconds = 30 * 60

func (r *videoGatewayRepository) ClaimMockRunnableTasks(ctx context.Context, limit int, lease time.Duration) ([]*service.VideoTask, error) {
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
	// Also reclaim recently-succeeded mock tasks that still lack generation-content
	// so capture fail-open retries survive worker restart. Capture stays fail-open:
	// reclaim never flips succeeded→failed; insert uses ON CONFLICT DO NOTHING.
	rows, err := db.QueryContext(ctx, `WITH candidates AS (
		SELECT vt.id FROM video_tasks vt
		WHERE vt.provider = 'mock'
		  AND (vt.worker_claimed_until IS NULL OR vt.worker_claimed_until <= NOW())
		  AND (
			vt.status IN ('queued','submitted','running')
			OR (
				vt.status = 'succeeded'
				AND vt.completed_at IS NOT NULL
				AND vt.completed_at > NOW() - ($3::int * INTERVAL '1 second')
				AND NOT EXISTS (
					SELECT 1 FROM ai_generation_content gc WHERE gc.task_id = vt.id
				)
			)
		  )
		ORDER BY vt.updated_at, vt.id LIMIT $1 FOR UPDATE SKIP LOCKED
	), claimed AS (
		UPDATE video_tasks task SET worker_claimed_at = NOW(),
		worker_claimed_until = NOW() + ($2::int * INTERVAL '1 second')
		FROM candidates WHERE task.id = candidates.id RETURNING task.*
	) SELECT `+videoTaskColumns+` FROM claimed`, limit, seconds, mockSucceededContentReclaimSeconds)
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

func (r *videoGatewayRepository) TransitionSimulationTask(ctx context.Context, taskID, expectedVersion int64, fromStatus, toStatus string) (*service.VideoTask, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	task, err := scanVideoTask(tx.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE id = $1 AND provider = $2 FOR UPDATE`, taskID, service.VideoProviderMock))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	if task.Version != expectedVersion || task.Status != fromStatus {
		return nil, service.ErrVideoTaskTerminalConflict
	}
	updated, err := scanVideoTask(tx.QueryRowContext(ctx, `UPDATE video_tasks
		SET status = $2, version = version + 1, updated_at = NOW()
		WHERE id = $1 AND provider = $3
		RETURNING `+videoTaskColumns, taskID, toStatus, service.VideoProviderMock))
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO video_task_events (video_task_id, event_type, payload_json)
		VALUES ($1, $2, '{}'::jsonb)`, taskID, toStatus); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *videoGatewayRepository) FinalizeSimulationTask(ctx context.Context, taskID, expectedVersion int64, status, errorMessage string) (service.VideoTaskFinalizationResult, error) {
	db, err := r.requireDB()
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if status != service.VideoStatusSucceeded && status != service.VideoStatusFailed && status != service.VideoStatusCancelled {
		return service.VideoTaskFinalizationResult{}, fmt.Errorf("video simulation finalization requires terminal status")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	task, err := scanVideoTask(tx.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE id = $1 AND provider = $2 FOR UPDATE`, taskID, service.VideoProviderMock))
	if errors.Is(err, sql.ErrNoRows) {
		return service.VideoTaskFinalizationResult{}, service.ErrVideoTaskNotFound
	}
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if task.Status == service.VideoStatusSucceeded || task.Status == service.VideoStatusFailed || task.Status == service.VideoStatusCancelled {
		if task.Status == status {
			return service.VideoTaskFinalizationResult{Applied: false, Idempotent: true, Status: task.Status, Version: task.Version}, nil
		}
		return service.VideoTaskFinalizationResult{}, service.ErrVideoTaskTerminalConflict
	}
	if task.Version != expectedVersion {
		return service.VideoTaskFinalizationResult{}, service.ErrVideoTaskTerminalConflict
	}
	updated, err := scanVideoTask(tx.QueryRowContext(ctx, `UPDATE video_tasks
		SET status = $2, error_message = $3, version = version + 1, completed_at = NOW(), updated_at = NOW(),
		    worker_claimed_until = NULL
		WHERE id = $1 AND provider = $4
		RETURNING `+videoTaskColumns, taskID, status, errorMessage, service.VideoProviderMock))
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO video_task_events (video_task_id, event_type, payload_json)
		VALUES ($1, $2, '{}'::jsonb)`, taskID, status); err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	return service.VideoTaskFinalizationResult{Applied: true, Status: updated.Status, Version: updated.Version}, nil
}

func (r *videoGatewayRepository) InsertVideoTaskEvent(ctx context.Context, taskID int64, eventType string, payload map[string]any) error {
	db, err := r.requireDB()
	if err != nil {
		return err
	}
	raw := []byte("{}")
	if payload != nil {
		encoded, encErr := json.Marshal(payload)
		if encErr != nil {
			return encErr
		}
		raw = encoded
	}
	_, err = db.ExecContext(ctx, `INSERT INTO video_task_events (video_task_id, event_type, payload_json)
		VALUES ($1, $2, $3::jsonb)`, taskID, eventType, string(raw))
	return err
}

func (r *videoGatewayRepository) ListVideoTaskEvents(ctx context.Context, taskID int64) ([]service.VideoTaskEvent, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT video_task_id, event_type, created_at
		FROM video_task_events WHERE video_task_id = $1 ORDER BY created_at, id`, taskID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.VideoTaskEvent, 0)
	for rows.Next() {
		var ev service.VideoTaskEvent
		if scanErr := rows.Scan(&ev.TaskID, &ev.EventType, &ev.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (r *videoGatewayRepository) CaptureTaskLinkedContent(ctx context.Context, task *service.VideoTask) error {
	if r == nil || task == nil || task.ID <= 0 {
		return nil
	}
	db, err := r.requireDB()
	if err != nil {
		return err
	}
	var apiKeyID, userID, groupID any
	if task.APIKeyID > 0 {
		apiKeyID = task.APIKeyID
	}
	if task.CreatedBy > 0 {
		userID = task.CreatedBy
	}
	if task.GroupID > 0 {
		groupID = task.GroupID
	}
	requestID := fmt.Sprintf("simulation-task-%d", task.ID)
	const query = `
INSERT INTO ai_generation_content (
    request_id, api_key_id, user_id, group_id, task_id, model,
    prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, FALSE, 1)
ON CONFLICT (task_id) WHERE task_id IS NOT NULL DO NOTHING`
	prompt := task.Prompt
	response := "模拟视频结果"
	_, err = db.ExecContext(ctx, query,
		requestID, apiKeyID, userID, groupID, task.ID, task.Model,
		prompt, response, len(prompt), len(response),
	)
	if err != nil {
		return fmt.Errorf("capture simulation generation content: %w", err)
	}
	return nil
}
