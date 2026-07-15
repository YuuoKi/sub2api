package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	batchImageAssetMaxBytes = 20 << 20 // 20 MiB per image
	batchImageAssetPrefix   = "assets/batch_image/"
)

var (
	ErrBatchImageAssetNotFound      = infraerrors.New(http.StatusNotFound, "BATCH_IMAGE_ASSET_NOT_FOUND", "batch image asset not found")
	ErrBatchImageAssetInvalid       = infraerrors.New(http.StatusBadRequest, "BATCH_IMAGE_ASSET_INVALID", "batch image asset is invalid")
	ErrBatchImageAssetPathUnsafe    = infraerrors.New(http.StatusBadRequest, "BATCH_IMAGE_ASSET_PATH_UNSAFE", "batch image asset path is unsafe")
	ErrBatchImageAssetValidation    = infraerrors.New(http.StatusConflict, "BATCH_IMAGE_ASSET_VALIDATION_FAILED", "batch image asset failed validation")
	ErrBatchImageAssetForbidden     = infraerrors.New(http.StatusForbidden, "BATCH_IMAGE_ASSET_FORBIDDEN", "batch image asset is not owned by current user")
)

type BatchImageAsset struct {
	ID             int64
	BatchID        string
	ItemID         int64
	ImageIndex     int
	StorageKey     string
	MimeType       string
	ByteSize       int64
	SHA256         string
	ArchivedAt     time.Time
	SourceProvider string
	SourceRef      *string
	CreatedAt      time.Time
}

type UpsertBatchImageAssetParams struct {
	BatchID        string
	ItemID         int64
	ImageIndex     int
	StorageKey     string
	MimeType       string
	ByteSize       int64
	SHA256         string
	ArchivedAt     time.Time
	SourceProvider string
	SourceRef      string
}

type ArchiveBatchImageInlineParams struct {
	BatchID        string
	ItemID         int64
	ImageIndex     int
	MimeType       string
	Base64Data     string
	SourceProvider string
	SourceRef      string
	StorageKeyHint string
}

type BatchImageAssetArchiver struct {
	Repo BatchImageRepository
}

func batchImageAssetDataDir() string {
	return videoAssetDataDir()
}

func ResolveBatchImageAssetAbsPath(rel string) (string, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		return "", ErrBatchImageAssetPathUnsafe
	}
	if !strings.HasPrefix(rel, batchImageAssetPrefix) {
		return "", ErrBatchImageAssetPathUnsafe
	}
	root := filepath.Clean(batchImageAssetDataDir())
	abs := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	if !isPathWithinBase(abs, root) {
		return "", ErrBatchImageAssetPathUnsafe
	}
	if err := rejectBatchImageAssetSymlinkEscape(root, abs); err != nil {
		return "", err
	}
	return abs, nil
}

func rejectBatchImageAssetSymlinkEscape(root, target string) error {
	current := root
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return ErrBatchImageAssetPathUnsafe
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ErrBatchImageAssetPathUnsafe
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return ErrBatchImageAssetPathUnsafe
		}
	}
	return nil
}

func isPathWithinBase(target, base string) bool {
	target = filepath.Clean(target)
	base = filepath.Clean(base)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func batchImageAssetStorageKey(batchID string, itemID int64, imageIndex int, ext string) string {
	ext = sanitizeBatchImageFilenameExtension(ext)
	if ext == "" {
		ext = "bin"
	}
	return filepath.ToSlash(filepath.Join(
		"assets",
		"batch_image",
		sanitizeBatchImageFilenameBase(batchID),
		fmt.Sprintf("item_%d_%d.%s", itemID, imageIndex, ext),
	))
}

func (a *BatchImageAssetArchiver) ArchiveInline(ctx context.Context, job *BatchImageJob, item *BatchImageItem, imageIndex int, image BatchImageInlineImage, sourceRef string) (*BatchImageAsset, error) {
	if a == nil || a.Repo == nil || job == nil || item == nil {
		return nil, ErrBatchImageAssetInvalid
	}
	if existing, err := a.Repo.GetBatchImageAssetByItemIndex(ctx, job.BatchID, item.ID, imageIndex); err == nil && existing != nil {
		if abs, resolveErr := ResolveBatchImageAssetAbsPath(existing.StorageKey); resolveErr == nil {
			if _, statErr := os.Stat(abs); statErr == nil {
				return existing, nil
			}
		}
	} else if err != nil && !errors.Is(err, ErrBatchImageAssetNotFound) {
		return nil, err
	}

	mimeType := strings.TrimSpace(image.MimeType)
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return nil, ErrBatchImageAssetInvalid
	}
	ext := strings.TrimSpace(image.Extension)
	if ext == "" {
		ext = batchImageFileExtension(mimeType)
	}
	storageKey := batchImageAssetStorageKey(job.BatchID, item.ID, imageIndex, ext)
	written, err := ArchiveBatchImageInline(ctx, ArchiveBatchImageInlineParams{
		BatchID:        job.BatchID,
		ItemID:         item.ID,
		ImageIndex:     imageIndex,
		MimeType:       mimeType,
		Base64Data:     image.Base64Data,
		SourceProvider: job.Provider,
		SourceRef:      sourceRef,
		StorageKeyHint: storageKey,
	})
	if err != nil {
		return nil, err
	}
	return a.Repo.UpsertBatchImageAsset(ctx, *written)
}

func ArchiveBatchImageInline(_ context.Context, params ArchiveBatchImageInlineParams) (*UpsertBatchImageAssetParams, error) {
	storageKey := strings.TrimSpace(params.StorageKeyHint)
	if storageKey == "" {
		storageKey = batchImageAssetStorageKey(params.BatchID, params.ItemID, params.ImageIndex, batchImageFileExtension(params.MimeType))
	}
	abs, err := ResolveBatchImageAssetAbsPath(storageKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return nil, err
	}

	tmp := abs + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(params.Base64Data))
	limited := io.LimitReader(decoder, batchImageAssetMaxBytes+1)
	n, copyErr := io.Copy(writer, limited)
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		return nil, firstErr(copyErr, closeErr)
	}
	if n > batchImageAssetMaxBytes {
		_ = os.Remove(tmp)
		return nil, ErrBatchImageAssetValidation
	}
	if n == 0 {
		_ = os.Remove(tmp)
		return nil, ErrBatchImageAssetInvalid
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}

	now := time.Now().UTC()
	return &UpsertBatchImageAssetParams{
		BatchID:        params.BatchID,
		ItemID:         params.ItemID,
		ImageIndex:     params.ImageIndex,
		StorageKey:     storageKey,
		MimeType:       params.MimeType,
		ByteSize:       n,
		SHA256:         hex.EncodeToString(hasher.Sum(nil)),
		ArchivedAt:     now,
		SourceProvider: params.SourceProvider,
		SourceRef:      params.SourceRef,
	}, nil
}

func OpenValidatedBatchImageAsset(asset *BatchImageAsset) (io.ReadCloser, string, int64, error) {
	if asset == nil {
		return nil, "", 0, ErrBatchImageAssetNotFound
	}
	mimeType := strings.TrimSpace(asset.MimeType)
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return nil, "", 0, ErrBatchImageAssetValidation
	}
	if asset.ByteSize < 0 || asset.ByteSize > batchImageAssetMaxBytes {
		return nil, "", 0, ErrBatchImageAssetValidation
	}
	abs, err := ResolveBatchImageAssetAbsPath(asset.StorageKey)
	if err != nil {
		return nil, "", 0, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, "", 0, ErrBatchImageAssetNotFound
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, "", 0, ErrBatchImageAssetPathUnsafe
	}
	if info.Size() > batchImageAssetMaxBytes || (asset.ByteSize > 0 && info.Size() != asset.ByteSize) {
		return nil, "", 0, ErrBatchImageAssetValidation
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, "", 0, err
	}
	hasher := sha256.New()
	limited := io.LimitReader(f, batchImageAssetMaxBytes+1)
	n, copyErr := io.Copy(hasher, limited)
	if copyErr != nil {
		_ = f.Close()
		return nil, "", 0, copyErr
	}
	if n > batchImageAssetMaxBytes {
		_ = f.Close()
		return nil, "", 0, ErrBatchImageAssetValidation
	}
	gotSum := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(strings.TrimSpace(asset.SHA256), gotSum) {
		_ = f.Close()
		return nil, "", 0, ErrBatchImageAssetValidation
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		_ = f.Close()
		return nil, "", 0, err
	}
	return f, mimeType, n, nil
}

func ResolveOwnedBatchImageAssetBytes(ctx context.Context, repo BatchImageRepository, owner BatchImageOwner, assetID int64) (*BatchImageReference, error) {
	if repo == nil || assetID <= 0 {
		return nil, ErrBatchImageAssetInvalid
	}
	asset, err := repo.GetBatchImageAssetForOwner(ctx, owner.UserID, owner.APIKeyID, assetID)
	if err != nil {
		return nil, err
	}
	rc, mimeType, _, err := OpenValidatedBatchImageAsset(asset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(io.LimitReader(rc, batchImageAssetMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > batchImageAssetMaxBytes {
		return nil, ErrBatchImageAssetValidation
	}
	return &BatchImageReference{
		MimeType: mimeType,
		Data:     data,
		AssetID:  asset.ID,
	}, nil
}

func BatchImageUpstreamEndpoint(provider string) string {
	switch strings.TrimSpace(provider) {
	case BatchImageProviderGeminiAPI:
		return "gemini:batchGenerateContent"
	case BatchImageProviderVertex:
		return "vertex:batchPredictionJobs"
	case BatchImageProviderMock:
		return "mock:batchImage"
	default:
		if provider == "" {
			return "unknown:batchImage"
		}
		return provider + ":batchImage"
	}
}
