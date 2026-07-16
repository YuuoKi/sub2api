package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
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
	row := r.db.QueryRowContext(ctx, `WITH inserted AS (INSERT INTO video_provider_accounts
		(group_id,provider,display_name,enabled,encrypted_api_key,masked_key,base_url,default_model)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING *)
		SELECT `+strings.ReplaceAll(videoAdminProviderColumns, "p.", "inserted.")+` FROM inserted JOIN groups g ON g.id=inserted.group_id`,
		item.GroupID, item.Provider, item.DisplayName, item.Enabled, item.EncryptedAPIKey, item.MaskedKey, item.BaseURL, item.DefaultModel)
	return scanVideoAdminProvider(row)
}
func (r *videoAdminRepository) UpdateVideoProvider(ctx context.Context, id int64, in service.VideoProviderAdminUpdate) (*service.VideoProviderAccount, error) {
	row := r.db.QueryRowContext(ctx, `WITH updated AS (UPDATE video_provider_accounts SET
		group_id=CASE WHEN $2 THEN $3 ELSE group_id END, display_name=CASE WHEN $4 THEN $5 ELSE display_name END,
		enabled=CASE WHEN $6 THEN $7 ELSE enabled END, encrypted_api_key=CASE WHEN $8 THEN $9 ELSE encrypted_api_key END,
		masked_key=CASE WHEN $8 THEN $10 ELSE masked_key END, base_url=CASE WHEN $11 THEN $12 ELSE base_url END,
		default_model=CASE WHEN $13 THEN $14 ELSE default_model END, updated_at=NOW() WHERE id=$1 RETURNING *)
		SELECT `+strings.ReplaceAll(videoAdminProviderColumns, "p.", "updated.")+` FROM updated JOIN groups g ON g.id=updated.group_id`,
		id, in.GroupID != nil, valueInt64(in.GroupID), in.DisplayName != nil, valueString(in.DisplayName), in.Enabled != nil, valueBool(in.Enabled),
		in.EncryptedAPIKey != nil, valueString(in.EncryptedAPIKey), valueString(in.MaskedKey), in.BaseURL != nil, valueString(in.BaseURL), in.DefaultModel != nil, valueString(in.DefaultModel))
	item, err := scanVideoAdminProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrVideoProviderNotFound
	}
	return item, err
}
func (r *videoAdminRepository) AuthorizeTinyReal(ctx context.Context, id, actor int64) (*service.VideoProviderAccount, error) {
	row := r.db.QueryRowContext(ctx, `WITH updated AS (UPDATE video_provider_accounts SET tiny_real_authorized_at=NOW(), tiny_real_authorized_by=$2, updated_at=NOW()
		WHERE id=$1 AND provider='seedance' AND enabled=TRUE AND encrypted_api_key<>'' AND tiny_real_authorized_at IS NULL AND tiny_real_consumed_at IS NULL RETURNING *)
		SELECT `+strings.ReplaceAll(videoAdminProviderColumns, "p.", "updated.")+` FROM updated JOIN groups g ON g.id=updated.group_id`, id, actor)
	item, err := scanVideoAdminProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("tiny_real authorization requires an enabled, configured Seedance provider and can only be granted once")
	}
	return item, err
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
