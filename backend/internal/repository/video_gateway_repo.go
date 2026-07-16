package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
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

func (r *videoGatewayRepository) ListEnabledVideoProviders(ctx context.Context, groupID int64) ([]service.VideoProviderAccount, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id, group_id, provider, display_name, enabled, '', masked_key, '', default_model
		FROM video_provider_accounts WHERE enabled=TRUE AND group_id=$1 ORDER BY id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.VideoProviderAccount, 0)
	for rows.Next() {
		var item service.VideoProviderAccount
		if err := rows.Scan(&item.ID, &item.GroupID, &item.Provider, &item.DisplayName, &item.Enabled, &item.EncryptedAPIKey, &item.MaskedKey, &item.BaseURL, &item.DefaultModel); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *videoGatewayRepository) GetVideoProvider(ctx context.Context, id, groupID int64) (*service.VideoProviderAccount, error) {
	db, err := r.requireDB()
	if err != nil {
		return nil, err
	}
	var item service.VideoProviderAccount
	err = db.QueryRowContext(ctx, `SELECT id, group_id, provider, display_name, enabled, encrypted_api_key, masked_key, base_url, default_model
		FROM video_provider_accounts WHERE id=$1 AND group_id=$2`, id, groupID).Scan(&item.ID, &item.GroupID, &item.Provider, &item.DisplayName, &item.Enabled,
		&item.EncryptedAPIKey, &item.MaskedKey, &item.BaseURL, &item.DefaultModel)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoProviderNotFound
	}
	return &item, err
}

func (r *videoGatewayRepository) BeginRealDispatch(ctx context.Context, id, version int64) (bool, error) {
	db, err := r.requireDB()
	if err != nil {
		return false, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	providerGrant, err := tx.ExecContext(ctx, `UPDATE video_provider_accounts provider SET tiny_real_consumed_at=NOW(), updated_at=NOW()
		FROM video_tasks task, groups candidate WHERE task.id=$1 AND task.provider_account_id=provider.id
		AND provider.group_id=task.group_id AND candidate.id=provider.group_id
		AND candidate.status='active' AND candidate.subscription_type='standard' AND candidate.deleted_at IS NULL
		AND provider.enabled=TRUE AND provider.provider='seedance' AND provider.default_model=$2 AND provider.base_url=$3
		AND provider.tiny_real_authorized_at IS NOT NULL AND provider.tiny_real_consumed_at IS NULL`, id, service.SeedanceModel, service.SeedanceBaseURL)
	if err != nil {
		return false, err
	}
	grants, err := providerGrant.RowsAffected()
	if err != nil {
		return false, err
	}
	if grants != 1 {
		return false, nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE video_tasks SET real_dispatch_count=1, dispatch_state='dispatching', version=version+1,
		updated_at=NOW() WHERE id=$1 AND version=$2 AND status='queued' AND real_dispatch_count=0`, id, version)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO video_single_smoke_consumptions (gate_key, video_task_id)
		VALUES ('global',$1) ON CONFLICT (gate_key) DO NOTHING`, id); err != nil {
		return false, err
	}
	var owned int64
	if err = tx.QueryRowContext(ctx, `SELECT video_task_id FROM video_single_smoke_consumptions WHERE gate_key='global'`).Scan(&owned); err != nil {
		return false, err
	}
	if owned != id {
		return false, nil
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
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

func (r *videoGatewayRepository) UpdateVideoProgress(ctx context.Context, id, version int64, status string) error {
	if status != service.VideoStatusSubmitted && status != service.VideoStatusRunning {
		return fmt.Errorf("invalid video progress status")
	}
	db, err := r.requireDB()
	if err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `UPDATE video_tasks SET status=$3, version=version+1, worker_claimed_at=NULL,
		worker_claimed_until=NULL, updated_at=NOW() WHERE id=$1 AND version=$2 AND status IN ('submitted','running')`, id, version, status)
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

func (r *videoGatewayRepository) ReserveAndCreateTask(ctx context.Context, task *service.VideoTask, maximumUSD float64) error {
	db, err := r.requireDB()
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("video task is required")
	}
	if maximumUSD <= 0 || task.CreatedBy <= 0 || task.APIKeyID <= 0 || task.GroupID <= 0 {
		return service.ErrVideoBudgetRejected
	}
	if task.DurationSeconds == 0 {
		task.DurationSeconds = 4
	}
	if task.Resolution == "" {
		task.Resolution = "720p"
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var userStatus string
	var balance float64
	var now time.Time
	if err = tx.QueryRowContext(ctx, `SELECT status, balance, NOW() FROM users
		WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, task.CreatedBy).Scan(&userStatus, &balance, &now); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrVideoBudgetRejected
		}
		return err
	}
	if userStatus != "active" || balance+0.00000001 < maximumUSD {
		return service.ErrVideoBudgetRejected
	}
	var keyUserID int64
	var keyGroupID sql.NullInt64
	var keyStatus string
	var expiresAt sql.NullTime
	var quota, quotaUsed, limit5h, limit1d, limit7d, usage5h, usage1d, usage7d float64
	var start5h, start1d, start7d sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT user_id, group_id, status, expires_at, quota, quota_used,
		rate_limit_5h, rate_limit_1d, rate_limit_7d, usage_5h, usage_1d, usage_7d,
		window_5h_start, window_1d_start, window_7d_start
		FROM api_keys WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, task.APIKeyID).Scan(
		&keyUserID, &keyGroupID, &keyStatus, &expiresAt, &quota, &quotaUsed,
		&limit5h, &limit1d, &limit7d, &usage5h, &usage1d, &usage7d, &start5h, &start1d, &start7d)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrVideoBudgetRejected
		}
		return err
	}
	if keyUserID != task.CreatedBy || !keyGroupID.Valid || keyGroupID.Int64 != task.GroupID || keyStatus != service.StatusAPIKeyActive || (expiresAt.Valid && !expiresAt.Time.After(now)) {
		return service.ErrVideoBudgetRejected
	}
	var groupStatus, subscriptionType string
	if err = tx.QueryRowContext(ctx, `SELECT status, subscription_type FROM groups
		WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, task.GroupID).Scan(&groupStatus, &subscriptionType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrVideoBudgetRejected
		}
		return err
	}
	if groupStatus != "active" || subscriptionType != "standard" {
		return service.ErrVideoBudgetRejected
	}
	var provider, model string
	var enabled bool
	if err = tx.QueryRowContext(ctx, `SELECT provider, default_model, enabled FROM video_provider_accounts
		WHERE id=$1 AND group_id=$2 FOR UPDATE`, task.ProviderAccountID, task.GroupID).Scan(&provider, &model, &enabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrVideoProviderNotFound
		}
		return err
	}
	if !enabled || provider != "seedance" || strings.TrimSpace(model) != "" && model != service.SeedanceModel {
		return service.ErrVideoProviderNotFound
	}
	usage5h, start5h = videoRateWindow(now, usage5h, start5h, 5*time.Hour, false)
	usage1d, start1d = videoRateWindow(now, usage1d, start1d, 24*time.Hour, true)
	usage7d, start7d = videoRateWindow(now, usage7d, start7d, 7*24*time.Hour, true)
	if quota > 0 && quotaUsed+maximumUSD-quota > 0.00000001 ||
		limit5h > 0 && usage5h+maximumUSD-limit5h > 0.00000001 ||
		limit1d > 0 && usage1d+maximumUSD-limit1d > 0.00000001 ||
		limit7d > 0 && usage7d+maximumUSD-limit7d > 0.00000001 {
		return service.ErrVideoBudgetRejected
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET balance=balance-$1,
		frozen_balance=COALESCE(frozen_balance,0)+$1, updated_at=NOW() WHERE id=$2`, maximumUSD, task.CreatedBy); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE api_keys SET quota_used=quota_used+$1,
		status=CASE WHEN quota>0 AND quota_used+$1>=quota THEN $9 ELSE status END,
		usage_5h=$2+$1, usage_1d=$3+$1, usage_7d=$4+$1,
		window_5h_start=$5, window_1d_start=$6, window_7d_start=$7, updated_at=NOW() WHERE id=$8`,
		maximumUSD, usage5h, usage1d, usage7d, start5h.Time, start1d.Time, start7d.Time, task.APIKeyID, service.StatusAPIKeyQuotaExhausted); err != nil {
		return err
	}
	task.Provider = provider
	task.ReservedCostUSD = maximumUSD
	task.ReservationState = service.VideoReservationReserved
	task.ReservedAt = &now
	task.ReservationWindow5h = &start5h.Time
	task.ReservationWindow1d = &start1d.Time
	task.ReservationWindow7d = &start7d.Time
	err = tx.QueryRowContext(ctx, `INSERT INTO video_tasks
		(provider_account_id, provider, model, task_type, prompt, status, creation_key, created_by,
		 api_key_id, group_id, duration_seconds, resolution, reserved_cost_usd, reservation_state,
		 reserved_at, reservation_window_5h_start, reservation_window_1d_start, reservation_window_7d_start)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING id, version, created_at, updated_at`,
		task.ProviderAccountID, task.Provider, task.Model, task.TaskType, task.Prompt,
		task.Status, task.CreationKey, task.CreatedBy, task.APIKeyID, task.GroupID,
		task.DurationSeconds, task.Resolution, maximumUSD, service.VideoReservationReserved, now,
		start5h.Time, start1d.Time, start7d.Time,
	).Scan(&task.ID, &task.Version, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func videoRateWindow(now time.Time, usage float64, start sql.NullTime, duration time.Duration, alignDay bool) (float64, sql.NullTime) {
	if start.Valid && start.Time.Add(duration).After(now) {
		return usage, start
	}
	newStart := now
	if alignDay {
		utc := now.UTC()
		newStart = time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	}
	return 0, sql.NullTime{Time: newStart, Valid: true}
}

const videoTaskColumns = `id, COALESCE(api_key_id, 0), COALESCE(group_id, 0), provider_account_id,
	provider, model, task_type, prompt, status, upstream_task_id, result_url, last_frame_url, duration_seconds,
	resolution, usage_total_tokens, cost_amount, currency, real_dispatch_count,
	provider_error_code, provider_error_message, error_message, COALESCE(creation_key, ''),
	version, dispatch_state, created_by, created_at, updated_at, completed_at,
	reserved_cost_usd, reservation_state, reserved_at, reservation_window_5h_start,
	reservation_window_1d_start, reservation_window_7d_start, provider_actual_cost_usd`

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
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	task, err := scanVideoTask(tx.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks
		WHERE id=$1 AND created_by=$2 AND api_key_id=$3 AND group_id=$4 FOR UPDATE`, id, scope.UserID, scope.APIKeyID, scope.GroupID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	if task.Status != service.VideoStatusQueued || task.RealDispatchCount != 0 || task.DispatchState != "pending" {
		return nil, service.ErrVideoCancelConflict
	}
	if err = settleVideoReservation(ctx, tx, task, 0); err != nil {
		return nil, err
	}
	task, err = scanVideoTask(tx.QueryRowContext(ctx, `UPDATE video_tasks SET status='cancelled', reservation_state=$2,
		cost_amount=0, completed_at=NOW(), version=version+1, worker_claimed_at=NULL, worker_claimed_until=NULL, updated_at=NOW()
		WHERE id=$1 RETURNING `+videoTaskColumns, id, service.VideoReservationReleased))
	if err != nil {
		return nil, err
	}
	if err = insertVideoUsageLog(ctx, tx, task); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return task, nil
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
	task, err := scanVideoTask(tx.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE id=$1 FOR UPDATE`, input.TaskID))
	if errors.Is(err, sql.ErrNoRows) {
		return service.VideoTaskFinalizationResult{}, service.ErrVideoTaskNotFound
	}
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	result := service.VideoTaskFinalizationResult{Status: task.Status, Version: task.Version}
	if task.Status == service.VideoStatusSucceeded || task.Status == service.VideoStatusFailed || task.Status == service.VideoStatusCancelled {
		if task.Status == input.Status {
			result.Idempotent = true
			return result, nil
		}
		return service.VideoTaskFinalizationResult{}, service.ErrVideoTaskTerminalConflict
	}
	if task.Version != input.ExpectedVersion {
		return service.VideoTaskFinalizationResult{}, service.ErrVideoTaskTerminalConflict
	}
	charge, reservationState, err := validateVideoSettlement(task, input)
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if charge > 0 {
		if err = claimVideoBillingKey(ctx, tx, task, input, charge); err != nil {
			return service.VideoTaskFinalizationResult{}, err
		}
	}
	if err = settleVideoReservation(ctx, tx, task, charge); err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	var version int64
	err = tx.QueryRowContext(ctx, `UPDATE video_tasks SET status=$3, result_url=$4, last_frame_url=$5,
		error_message=$6, usage_total_tokens=$7, cost_amount=$8, currency=$9,
		provider_error_code=$10, provider_error_message=$11, completed_at=$12, version=version+1,
		worker_claimed_at=NULL, worker_claimed_until=NULL, reservation_state=$13,
		provider_actual_cost_usd=$14, balance_charged_at=CASE WHEN $8::numeric > 0 THEN NOW() ELSE NULL END,
		updated_at=NOW() WHERE id=$1 AND version=$2 RETURNING version`, input.TaskID, input.ExpectedVersion, input.Status,
		input.ResultURL, input.LastFrameURL, input.ErrorMessage, input.UsageTotalTokens, charge,
		input.Currency, input.ProviderErrorCode, input.ProviderErrorMessage, input.CompletedAt,
		reservationState, input.ProviderActualCostUSD).Scan(&version)
	if err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	task.Status, task.ResultURL, task.LastFrameURL = input.Status, input.ResultURL, input.LastFrameURL
	task.UsageTotalTokens, task.CostAmount, task.ProviderActualCostUSD = input.UsageTotalTokens, charge, input.ProviderActualCostUSD
	if err = insertVideoUsageLog(ctx, tx, task); err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return service.VideoTaskFinalizationResult{}, err
	}
	result.Applied, result.Status, result.Version = true, input.Status, version
	return result, nil
}

func validateVideoSettlement(task *service.VideoTask, input service.VideoTaskFinalization) (float64, string, error) {
	if task.ReservationState != service.VideoReservationReserved || task.ReservedCostUSD <= 0 {
		return 0, "", errors.New("video task has no active reservation")
	}
	switch input.Settlement {
	case service.VideoSettlementRelease:
		if input.Status == service.VideoStatusSucceeded {
			return 0, "", errors.New("successful video cannot release its reservation")
		}
		return 0, service.VideoReservationReleased, nil
	case service.VideoSettlementCaptureActual:
		if input.Status != service.VideoStatusSucceeded || strings.TrimSpace(input.ResultURL) == "" || input.UsageTotalTokens == nil || *input.UsageTotalTokens <= 0 || input.CostAmount <= 0 {
			return 0, "", errors.New("successful video settlement requires asset, usage, and positive cost")
		}
		if input.ProviderActualCostUSD <= 0 || math.Abs(input.ProviderActualCostUSD-input.CostAmount) > 0.00000001 {
			return 0, "", errors.New("successful video settlement requires provider actual cost to equal charged cost")
		}
		if input.CostAmount-task.ReservedCostUSD > 0.00000001 {
			return 0, "", service.ErrVideoBudgetRejected
		}
		return input.CostAmount, service.VideoReservationCaptured, nil
	case service.VideoSettlementCaptureReserved:
		if input.Status != service.VideoStatusFailed || input.ProviderErrorCode != "budget_actual_exceeded" || input.ProviderActualCostUSD-task.ReservedCostUSD <= 0.00000001 {
			return 0, "", errors.New("reserved capture requires an over-budget terminal failure")
		}
		return task.ReservedCostUSD, service.VideoReservationCaptured, nil
	default:
		return 0, "", errors.New("video settlement action is required")
	}
}

func settleVideoReservation(ctx context.Context, tx *sql.Tx, task *service.VideoTask, charge float64) error {
	refund := task.ReservedCostUSD - charge
	if refund < -0.00000001 {
		return service.ErrVideoBudgetRejected
	}
	result, err := tx.ExecContext(ctx, `UPDATE users SET balance=balance+$1,
		frozen_balance=COALESCE(frozen_balance,0)-$2, updated_at=NOW()
		WHERE id=$3 AND deleted_at IS NULL AND COALESCE(frozen_balance,0)>=$2`, refund, task.ReservedCostUSD, task.CreatedBy)
	if err != nil {
		return err
	}
	if affected, affectedErr := result.RowsAffected(); affectedErr != nil || affected != 1 {
		if affectedErr != nil {
			return affectedErr
		}
		return errors.New("video frozen balance is insufficient")
	}
	result, err = tx.ExecContext(ctx, `UPDATE api_keys SET quota_used=GREATEST(0,quota_used-$1),
		status=CASE WHEN status=$6 AND (quota=0 OR GREATEST(0,quota_used-$1)<quota) THEN $7 ELSE status END,
		usage_5h=CASE WHEN window_5h_start IS NOT DISTINCT FROM $2 THEN GREATEST(0,usage_5h-$1) ELSE usage_5h END,
		usage_1d=CASE WHEN window_1d_start IS NOT DISTINCT FROM $3 THEN GREATEST(0,usage_1d-$1) ELSE usage_1d END,
		usage_7d=CASE WHEN window_7d_start IS NOT DISTINCT FROM $4 THEN GREATEST(0,usage_7d-$1) ELSE usage_7d END,
		updated_at=NOW() WHERE id=$5 AND deleted_at IS NULL`, refund,
		timePtrValue(task.ReservationWindow5h), timePtrValue(task.ReservationWindow1d), timePtrValue(task.ReservationWindow7d), task.APIKeyID,
		service.StatusAPIKeyQuotaExhausted, service.StatusAPIKeyActive)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("video api key reservation is missing")
	}
	return nil
}

func claimVideoBillingKey(ctx context.Context, tx *sql.Tx, task *service.VideoTask, input service.VideoTaskFinalization, charge float64) error {
	requestID := fmt.Sprintf("video:%d", task.ID)
	fingerprint := service.HashUsageRequestPayload([]byte(fmt.Sprintf("%d|%d|%d|%s|%d|%.8f|%.8f|%s", task.ID, task.CreatedBy,
		task.APIKeyID, task.Model, valueOrZero(input.UsageTotalTokens), charge, input.ProviderActualCostUSD, input.Settlement)))
	var id int64
	err := tx.QueryRowContext(ctx, `INSERT INTO usage_billing_dedup (request_id,api_key_id,request_fingerprint)
		VALUES ($1,$2,$3) ON CONFLICT (request_id,api_key_id) DO NOTHING RETURNING id`, requestID, task.APIKeyID, fingerprint).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrUsageBillingRequestConflict
	}
	if err != nil {
		return err
	}
	var archived string
	err = tx.QueryRowContext(ctx, `SELECT request_fingerprint FROM usage_billing_dedup_archive
		WHERE request_id=$1 AND api_key_id=$2`, requestID, task.APIKeyID).Scan(&archived)
	if err == nil {
		return service.ErrUsageBillingRequestConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

func insertVideoUsageLog(ctx context.Context, tx *sql.Tx, task *service.VideoTask) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO video_usage_logs
		(video_task_id,provider,model,status,completion_tokens,charged_cost_usd,provider_actual_cost_usd,currency,result_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'USD',$8) ON CONFLICT (video_task_id) DO NOTHING`, task.ID, task.Provider,
		task.Model, task.Status, task.UsageTotalTokens, task.CostAmount, task.ProviderActualCostUSD, task.ResultURL)
	return err
}

func timePtrValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
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
	var completed, reservedAt, reservation5h, reservation1d, reservation7d sql.NullTime
	var usage sql.NullInt64
	if err := scanner.Scan(&task.ID, &task.APIKeyID, &task.GroupID, &task.ProviderAccountID,
		&task.Provider, &task.Model, &task.TaskType, &task.Prompt, &task.Status, &task.UpstreamTaskID, &task.ResultURL,
		&task.LastFrameURL, &task.DurationSeconds, &task.Resolution, &usage, &task.CostAmount,
		&task.Currency, &task.RealDispatchCount, &task.ProviderErrorCode, &task.ProviderErrorMessage,
		&task.ErrorMessage,
		&task.CreationKey, &task.Version, &task.DispatchState, &task.CreatedBy,
		&task.CreatedAt, &task.UpdatedAt, &completed, &task.ReservedCostUSD, &task.ReservationState,
		&reservedAt, &reservation5h, &reservation1d, &reservation7d, &task.ProviderActualCostUSD); err != nil {
		return nil, err
	}
	if usage.Valid {
		task.UsageTotalTokens = &usage.Int64
	}
	if completed.Valid {
		task.CompletedAt = &completed.Time
	}
	if reservedAt.Valid {
		task.ReservedAt = &reservedAt.Time
	}
	if reservation5h.Valid {
		task.ReservationWindow5h = &reservation5h.Time
	}
	if reservation1d.Valid {
		task.ReservationWindow1d = &reservation1d.Time
	}
	if reservation7d.Valid {
		task.ReservationWindow7d = &reservation7d.Time
	}
	return task, nil
}
