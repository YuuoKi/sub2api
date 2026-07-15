package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const batchImageAssetColumns = `
id, batch_id, item_id, image_index, storage_key, mime_type, byte_size, sha256,
archived_at, source_provider, source_ref, created_at`

func (r *batchImageRepository) UpsertBatchImageAsset(ctx context.Context, params service.UpsertBatchImageAssetParams) (*service.BatchImageAsset, error) {
	archivedAt := params.ArchivedAt
	if archivedAt.IsZero() {
		archivedAt = time.Now().UTC()
	}
	var sourceRef any
	if params.SourceRef != "" {
		sourceRef = params.SourceRef
	}
	row := r.sql.QueryRowContext(ctx, `
INSERT INTO batch_image_assets (
    batch_id, item_id, image_index, storage_key, mime_type, byte_size, sha256,
    archived_at, source_provider, source_ref
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (batch_id, item_id, image_index) DO UPDATE SET
    storage_key = EXCLUDED.storage_key,
    mime_type = EXCLUDED.mime_type,
    byte_size = EXCLUDED.byte_size,
    sha256 = EXCLUDED.sha256,
    archived_at = EXCLUDED.archived_at,
    source_provider = EXCLUDED.source_provider,
    source_ref = EXCLUDED.source_ref
RETURNING `+batchImageAssetColumns,
		params.BatchID, params.ItemID, params.ImageIndex, params.StorageKey, params.MimeType, params.ByteSize, params.SHA256,
		archivedAt, params.SourceProvider, sourceRef,
	)
	asset, err := scanBatchImageAsset(row)
	if err != nil {
		return nil, translatePersistenceError(err, nil, nil)
	}
	return asset, nil
}

func (r *batchImageRepository) GetBatchImageAssetByItemIndex(ctx context.Context, batchID string, itemID int64, imageIndex int) (*service.BatchImageAsset, error) {
	asset, err := scanBatchImageAsset(r.sql.QueryRowContext(ctx, `
SELECT `+batchImageAssetColumns+` FROM batch_image_assets
 WHERE batch_id = $1 AND item_id = $2 AND image_index = $3`, batchID, itemID, imageIndex))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrBatchImageAssetNotFound, nil)
	}
	return asset, nil
}

func (r *batchImageRepository) GetBatchImageAssetByCustomID(ctx context.Context, batchID, customID string, imageIndex int) (*service.BatchImageAsset, error) {
	asset, err := scanBatchImageAsset(r.sql.QueryRowContext(ctx, `
SELECT a.id, a.batch_id, a.item_id, a.image_index, a.storage_key, a.mime_type, a.byte_size, a.sha256,
       a.archived_at, a.source_provider, a.source_ref, a.created_at
  FROM batch_image_assets a
  JOIN batch_image_items i ON i.id = a.item_id
 WHERE a.batch_id = $1 AND i.custom_id = $2 AND a.image_index = $3`, batchID, customID, imageIndex))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrBatchImageAssetNotFound, nil)
	}
	return asset, nil
}

func (r *batchImageRepository) GetBatchImageAssetForOwner(ctx context.Context, userID, apiKeyID, assetID int64) (*service.BatchImageAsset, error) {
	asset, err := scanBatchImageAsset(r.sql.QueryRowContext(ctx, `
SELECT a.id, a.batch_id, a.item_id, a.image_index, a.storage_key, a.mime_type, a.byte_size, a.sha256,
       a.archived_at, a.source_provider, a.source_ref, a.created_at
  FROM batch_image_assets a
  JOIN batch_image_jobs j ON j.batch_id = a.batch_id
 WHERE a.id = $1 AND j.user_id = $2 AND j.api_key_id = $3 AND j.user_deleted_at IS NULL`, assetID, userID, apiKeyID))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrBatchImageAssetNotFound, nil)
	}
	return asset, nil
}

func (r *batchImageRepository) ListBatchImageAssetsForBatch(ctx context.Context, batchID string) ([]*service.BatchImageAsset, error) {
	rows, err := r.sql.QueryContext(ctx, `
SELECT `+batchImageAssetColumns+` FROM batch_image_assets
 WHERE batch_id = $1
 ORDER BY item_id ASC, image_index ASC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*service.BatchImageAsset
	for rows.Next() {
		asset, err := scanBatchImageAsset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, asset)
	}
	return out, rows.Err()
}

func scanBatchImageAsset(row rowScanner) (*service.BatchImageAsset, error) {
	var asset service.BatchImageAsset
	var sourceRef sql.NullString
	err := row.Scan(
		&asset.ID, &asset.BatchID, &asset.ItemID, &asset.ImageIndex, &asset.StorageKey, &asset.MimeType, &asset.ByteSize, &asset.SHA256,
		&asset.ArchivedAt, &asset.SourceProvider, &sourceRef, &asset.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	asset.SourceRef = batchImageNullStringPtr(sourceRef)
	return &asset, nil
}
