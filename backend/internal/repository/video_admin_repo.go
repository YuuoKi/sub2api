package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type videoAdminRepository struct{ db *sql.DB }

func NewVideoAdminRepository(db *sql.DB) service.VideoAdminRepository {
	return &videoAdminRepository{db: db}
}

const videoAdminProviderColumns = `p.id, p.group_id, COALESCE(g.name,''), p.provider, p.display_name, p.enabled,
	'', p.masked_key, p.base_url, p.default_model, p.tiny_real_authorized_at,
	COALESCE(p.tiny_real_authorized_by,0), p.tiny_real_consumed_at`

type videoProviderScanner interface{ Scan(...any) error }

func scanVideoAdminProvider(row videoProviderScanner) (*service.VideoProviderAccount, error) {
	var item service.VideoProviderAccount
	err := row.Scan(&item.ID, &item.GroupID, &item.GroupName, &item.Provider, &item.DisplayName, &item.Enabled,
		&item.EncryptedAPIKey, &item.MaskedKey, &item.BaseURL, &item.DefaultModel, &item.TinyRealAuthorizedAt,
		&item.TinyRealAuthorizedBy, &item.TinyRealConsumedAt)
	if err != nil {
		return nil, err
	}
	item.APIKeyConfigured = item.MaskedKey != ""
	return &item, nil
}

func (r *videoAdminRepository) ListVideoProviders(ctx context.Context) ([]service.VideoProviderAccount, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+videoAdminProviderColumns+` FROM video_provider_accounts p JOIN groups g ON g.id=p.group_id ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.VideoProviderAccount, 0)
	for rows.Next() {
		item, scanErr := scanVideoAdminProvider(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}
func (r *videoAdminRepository) CreateVideoProvider(ctx context.Context, item service.VideoProviderAccount) (*service.VideoProviderAccount, error) {
	valid, conflict, err := r.validateVideoProviderTarget(ctx, item.GroupID, item.Provider, item.DefaultModel, 0)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, service.ErrVideoAdminInvalidGroup
	}
	if conflict {
		return nil, service.ErrVideoAdminConflict
	}
	row := r.db.QueryRowContext(ctx, `WITH inserted AS (INSERT INTO video_provider_accounts
		(group_id,provider,display_name,enabled,encrypted_api_key,masked_key,base_url,default_model)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8 FROM groups
		WHERE id=$1 AND status='active' AND subscription_type='standard' AND deleted_at IS NULL RETURNING *)
		SELECT `+strings.ReplaceAll(videoAdminProviderColumns, "p.", "inserted.")+` FROM inserted JOIN groups g ON g.id=inserted.group_id`,
		item.GroupID, item.Provider, item.DisplayName, item.Enabled, item.EncryptedAPIKey, item.MaskedKey, item.BaseURL, item.DefaultModel)
	created, err := scanVideoAdminProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoAdminInvalidGroup
	}
	return created, err
}
func (r *videoAdminRepository) UpdateVideoProvider(ctx context.Context, id int64, in service.VideoProviderAdminUpdate) (*service.VideoProviderAccount, error) {
	var currentGroupID int64
	var currentProvider, currentModel string
	if err := r.db.QueryRowContext(ctx, `SELECT group_id, provider, default_model FROM video_provider_accounts WHERE id=$1`, id).Scan(&currentGroupID, &currentProvider, &currentModel); errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoProviderNotFound
	} else if err != nil {
		return nil, err
	}
	targetGroupID := currentGroupID
	if in.GroupID != nil {
		targetGroupID = *in.GroupID
	}
	targetModel := currentModel
	if in.DefaultModel != nil {
		targetModel = *in.DefaultModel
	}
	valid, conflict, err := r.validateVideoProviderTarget(ctx, targetGroupID, currentProvider, targetModel, id)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, service.ErrVideoAdminInvalidGroup
	}
	if conflict {
		return nil, service.ErrVideoAdminConflict
	}
	row := r.db.QueryRowContext(ctx, `WITH updated AS (UPDATE video_provider_accounts SET
		group_id=CASE WHEN $2 THEN $3 ELSE group_id END, display_name=CASE WHEN $4 THEN $5 ELSE display_name END,
		enabled=CASE WHEN $6 THEN $7 ELSE enabled END, encrypted_api_key=CASE WHEN $8 THEN $9 ELSE encrypted_api_key END,
		masked_key=CASE WHEN $8 THEN $10 ELSE masked_key END, base_url=CASE WHEN $11 THEN $12 ELSE base_url END,
		default_model=CASE WHEN $13 THEN $14 ELSE default_model END, updated_at=NOW() WHERE id=$1
		AND EXISTS (SELECT 1 FROM groups candidate WHERE candidate.id=CASE WHEN $2 THEN $3 ELSE video_provider_accounts.group_id END
		AND candidate.status='active' AND candidate.subscription_type='standard' AND candidate.deleted_at IS NULL) RETURNING *)
		SELECT `+strings.ReplaceAll(videoAdminProviderColumns, "p.", "updated.")+` FROM updated JOIN groups g ON g.id=updated.group_id`,
		id, in.GroupID != nil, valueInt64(in.GroupID), in.DisplayName != nil, valueString(in.DisplayName), in.Enabled != nil, valueBool(in.Enabled),
		in.EncryptedAPIKey != nil, valueString(in.EncryptedAPIKey), valueString(in.MaskedKey), in.BaseURL != nil, valueString(in.BaseURL), in.DefaultModel != nil, valueString(in.DefaultModel))
	item, err := scanVideoAdminProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if existsErr := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM video_provider_accounts WHERE id=$1)`, id).Scan(&exists); existsErr != nil {
			return nil, existsErr
		}
		if exists {
			return nil, service.ErrVideoAdminInvalidGroup
		}
		return nil, service.ErrVideoProviderNotFound
	}
	return item, err
}
func (r *videoAdminRepository) DeleteVideoProvider(ctx context.Context, id int64) error {
	var exists, deleted bool
	err := r.db.QueryRowContext(ctx, `WITH deleted AS (DELETE FROM video_provider_accounts WHERE id=$1
		AND NOT EXISTS (SELECT 1 FROM video_tasks WHERE provider_account_id=$1) RETURNING id)
		SELECT EXISTS(SELECT 1 FROM video_provider_accounts WHERE id=$1), EXISTS(SELECT 1 FROM deleted)`, id).Scan(&exists, &deleted)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return fmt.Errorf("%w: 该通道仍被视频任务引用，不能删除", service.ErrVideoAdminConflict)
		}
		return err
	}
	if deleted {
		return nil
	}
	if !exists {
		return service.ErrVideoProviderNotFound
	}
	return fmt.Errorf("%w: 该通道仍被视频任务引用，不能删除", service.ErrVideoAdminConflict)
}
func (r *videoAdminRepository) AuthorizeTinyReal(ctx context.Context, id, actor int64) (*service.VideoProviderAccount, error) {
	row := r.db.QueryRowContext(ctx, `WITH updated AS (UPDATE video_provider_accounts SET tiny_real_authorized_at=NOW(), tiny_real_authorized_by=$2, updated_at=NOW()
		FROM groups candidate WHERE video_provider_accounts.id=$1 AND candidate.id=video_provider_accounts.group_id
		AND candidate.status='active' AND candidate.subscription_type='standard' AND candidate.deleted_at IS NULL
		AND video_provider_accounts.provider='seedance' AND video_provider_accounts.default_model=$3 AND video_provider_accounts.base_url=$4
		AND video_provider_accounts.enabled=TRUE AND video_provider_accounts.encrypted_api_key<>''
		AND video_provider_accounts.tiny_real_authorized_at IS NULL AND video_provider_accounts.tiny_real_consumed_at IS NULL RETURNING video_provider_accounts.*)
		SELECT `+strings.ReplaceAll(videoAdminProviderColumns, "p.", "updated.")+` FROM updated JOIN groups g ON g.id=updated.group_id`, id, actor, service.SeedanceModel, service.SeedanceBaseURL)
	item, err := scanVideoAdminProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if existsErr := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM video_provider_accounts WHERE id=$1)`, id).Scan(&exists); existsErr != nil {
			return nil, existsErr
		}
		if !exists {
			return nil, service.ErrVideoProviderNotFound
		}
		return nil, service.ErrVideoAdminAuthorizationConflict
	}
	return item, err
}

func (r *videoAdminRepository) validateVideoProviderTarget(ctx context.Context, groupID int64, provider, model string, excludeID int64) (bool, bool, error) {
	var valid, conflict bool
	err := r.db.QueryRowContext(ctx, `SELECT
		EXISTS(SELECT 1 FROM groups WHERE id=$1 AND status='active' AND subscription_type='standard' AND deleted_at IS NULL),
		EXISTS(SELECT 1 FROM video_provider_accounts WHERE group_id=$1 AND provider=$2 AND default_model=$3 AND id<>$4)`,
		groupID, provider, model, excludeID).Scan(&valid, &conflict)
	return valid, conflict, err
}
func (r *videoAdminRepository) ListVideoTasks(ctx context.Context, filter service.VideoAdminTaskFilter) ([]service.VideoTask, int64, error) {
	where := ""
	args := []any{}
	if strings.TrimSpace(filter.Status) != "" {
		where = " WHERE status=$1"
		args = append(args, strings.TrimSpace(filter.Status))
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM video_tasks"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	limitPos := len(args) - 1
	offsetPos := len(args)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM video_tasks%s ORDER BY created_at DESC,id DESC LIMIT $%d OFFSET $%d", videoTaskColumns, where, limitPos, offsetPos), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]service.VideoTask, 0)
	for rows.Next() {
		task, scanErr := scanVideoTask(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, *task)
	}
	return items, total, rows.Err()
}
func (r *videoAdminRepository) GetVideoTaskAdmin(ctx context.Context, id int64) (*service.VideoTask, error) {
	task, err := scanVideoTask(r.db.QueryRowContext(ctx, `SELECT `+videoTaskColumns+` FROM video_tasks WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoTaskNotFound
	}
	return task, err
}
func (r *videoAdminRepository) VideoSystemCheck(ctx context.Context) (service.VideoSystemCheck, error) {
	var out service.VideoSystemCheck
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(*) FILTER(WHERE enabled),COUNT(*) FILTER(WHERE tiny_real_authorized_at IS NOT NULL AND tiny_real_consumed_at IS NULL) FROM video_provider_accounts`).Scan(&out.ProviderCount, &out.EnabledProviderCount, &out.AuthorizedProviderCount)
	if err != nil {
		return out, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(real_dispatch_count),0) FROM video_tasks`).Scan(&out.TaskCount, &out.RealDispatchCount)
	if err != nil {
		return out, err
	}
	var count int64
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM video_single_smoke_consumptions WHERE gate_key='global'`).Scan(&count)
	out.GlobalTinyRealConsumed = count > 0
	return out, err
}
func valueInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
func valueString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func valueBool(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}
