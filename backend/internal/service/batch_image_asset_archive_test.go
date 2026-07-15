//go:build unit

package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO5W7W0AAAAASUVORK5CYII="

func TestResolveBatchImageAssetAbsPathRejectsTraversal(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	tests := []string{
		"",
		"../etc/passwd",
		"/etc/passwd",
		"assets/video/1/result.mp4",
		"assets/batch_image/../../secret.png",
		"assets/batch_image/foo/../../../etc/passwd",
	}
	for _, rel := range tests {
		_, err := ResolveBatchImageAssetAbsPath(rel)
		require.Error(t, err, rel)
	}
}

func TestResolveBatchImageAssetAbsPathAcceptsConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	rel := "assets/batch_image/imgbatch_1/item_1_0.png"
	abs, err := ResolveBatchImageAssetAbsPath(rel)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(filepath.Clean(abs), filepath.Clean(root)))
}

func TestArchiveBatchImageInlineRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	outside := t.TempDir()
	linkParent := filepath.Join(root, "assets", "batch_image")
	require.NoError(t, os.MkdirAll(linkParent, 0o750))
	linkPath := filepath.Join(linkParent, "escape")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink unavailable in this environment: %v", err)
	}

	_, err := ArchiveBatchImageInline(context.Background(), ArchiveBatchImageInlineParams{
		BatchID:        "imgbatch_symlink",
		ItemID:         42,
		ImageIndex:     0,
		MimeType:       "image/png",
		Base64Data:     tinyPNGBase64,
		SourceProvider: BatchImageProviderGeminiAPI,
		SourceRef:      "files/out",
		StorageKeyHint: "assets/batch_image/escape/evil.png",
	})
	require.Error(t, err)
}

func TestArchiveBatchImageInlineStreamsAndIdempotentUpsert(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	repo := newFakeBatchImageRepository()
	job := &BatchImageJob{
		BatchID:  "imgbatch_archive",
		UserID:   7,
		APIKeyID: batchImageInt64Ptr(9),
		Provider: BatchImageProviderGeminiAPI,
		Status:   BatchImageJobStatusIndexing,
		Model:    "gemini-3.1-flash-image",
	}
	repo.jobs[job.BatchID] = job
	repo.items[job.BatchID] = []CreateBatchImageItemParams{
		{JobID: job.BatchID, CustomID: "cover_001", Status: BatchImageItemStatusSuccess, ImageCount: 1},
	}
	item := &BatchImageItem{ID: 1, JobID: job.BatchID, CustomID: "cover_001", Status: BatchImageItemStatusSuccess, ImageCount: 1}

	decoded, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	require.NoError(t, err)
	sum := sha256.Sum256(decoded)

	archiver := &BatchImageAssetArchiver{Repo: repo}
	first, err := archiver.ArchiveInline(context.Background(), job, item, 0, BatchImageInlineImage{
		MimeType:   "image/png",
		Extension:  "png",
		Base64Data: tinyPNGBase64,
	}, "files/out")
	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(sum[:]), first.SHA256)
	require.Equal(t, int64(len(decoded)), first.ByteSize)
	require.Equal(t, "image/png", first.MimeType)
	require.FileExists(t, mustAbsBatchImageAsset(t, first.StorageKey))

	second, err := archiver.ArchiveInline(context.Background(), job, item, 0, BatchImageInlineImage{
		MimeType:   "image/png",
		Extension:  "png",
		Base64Data: tinyPNGBase64,
	}, "files/out")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Len(t, repo.assets, 1)
}

func TestOpenLocalBatchImageAssetValidatesMIMESizeAndSHA(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	decoded, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	require.NoError(t, err)
	sum := sha256.Sum256(decoded)
	rel := "assets/batch_image/imgbatch_v/item_1_0.png"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o750))
	require.NoError(t, os.WriteFile(abs, decoded, 0o640))

	asset := &BatchImageAsset{
		ID:          1,
		BatchID:     "imgbatch_v",
		ItemID:      1,
		ImageIndex:  0,
		StorageKey:  rel,
		MimeType:    "image/png",
		ByteSize:    int64(len(decoded)),
		SHA256:      hex.EncodeToString(sum[:]),
		SourceProvider: BatchImageProviderGeminiAPI,
	}

	rc, ctype, size, err := OpenValidatedBatchImageAsset(asset)
	require.NoError(t, err)
	defer rc.Close()
	body, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, decoded, body)
	require.Equal(t, "image/png", ctype)
	require.Equal(t, int64(len(decoded)), size)

	asset.SHA256 = strings.Repeat("0", 64)
	_, _, _, err = OpenValidatedBatchImageAsset(asset)
	require.Error(t, err)

	asset.SHA256 = hex.EncodeToString(sum[:])
	asset.MimeType = "application/octet-stream"
	_, _, _, err = OpenValidatedBatchImageAsset(asset)
	require.Error(t, err)

	asset.MimeType = "image/png"
	asset.ByteSize = batchImageAssetMaxBytes + 1
	_, _, _, err = OpenValidatedBatchImageAsset(asset)
	require.Error(t, err)
}

func TestBatchImageDownloadPrefersLocalAssetAfterProviderExpiry(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	svc, repo, _ := newTestBatchImageDownloadService()
	job := repo.jobs["imgbatch_download"]
	job.Status = BatchImageJobStatusCompleted
	customID := "cover/../001"
	decoded, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	require.NoError(t, err)
	sum := sha256.Sum256(decoded)
	rel := "assets/batch_image/imgbatch_download/item_1_0.png"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o750))
	require.NoError(t, os.WriteFile(abs, decoded, 0o640))
	asset := &BatchImageAsset{
		ID:             55,
		BatchID:        "imgbatch_download",
		ItemID:         1,
		ImageIndex:     0,
		StorageKey:     rel,
		MimeType:       "image/png",
		ByteSize:       int64(len(decoded)),
		SHA256:         hex.EncodeToString(sum[:]),
		SourceProvider: BatchImageProviderGeminiAPI,
	}
	repo.assets[55] = asset
	repo.assetsByKey["imgbatch_download:1:0"] = asset
	repo.assetsByKey["imgbatch_download:"+customID+":0"] = asset

	// Provider result gone — local asset must still serve.
	svc.ProviderRegistry = NewBatchImageProviderRegistry(&expiredResultProvider{})

	stream, err := svc.OpenItemContent(context.Background(), testBatchImageOwner(), "imgbatch_download", customID, 0)
	require.NoError(t, err)
	defer stream.Reader.Close()
	body, err := io.ReadAll(stream.Reader)
	require.NoError(t, err)
	require.Equal(t, decoded, body)
	require.Equal(t, "image/png", stream.ContentType)
}

func TestBatchImageDownloadBackfillsArchiveWhenLocalMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	svc, repo, _ := newTestBatchImageDownloadService()
	repo.items["imgbatch_download"] = []CreateBatchImageItemParams{
		{JobID: "imgbatch_download", CustomID: "cover_backfill", Status: BatchImageItemStatusSuccess, ImageCount: 1},
	}

	pngLine := `{"custom_id":"cover_backfill","response":{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"` + tinyPNGBase64 + `"}}]}}]}}`
	svc.ProviderRegistry = NewBatchImageProviderRegistry(&staticJSONLProvider{jsonl: pngLine})

	stream, err := svc.OpenItemContent(context.Background(), testBatchImageOwner(), "imgbatch_download", "cover_backfill", 0)
	require.NoError(t, err)
	defer stream.Reader.Close()
	body, err := io.ReadAll(stream.Reader)
	require.NoError(t, err)
	decoded, _ := base64.StdEncoding.DecodeString(tinyPNGBase64)
	require.Equal(t, decoded, body)
	require.NotEmpty(t, repo.assets)
}

func TestResolveOwnedBatchImageAssetForReuse(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	repo := newFakeBatchImageRepository()
	job := &BatchImageJob{BatchID: "imgbatch_reuse", UserID: 42, APIKeyID: batchImageInt64Ptr(7), Status: BatchImageJobStatusCompleted, Provider: BatchImageProviderGeminiAPI}
	repo.jobs[job.BatchID] = job
	decoded, _ := base64.StdEncoding.DecodeString(tinyPNGBase64)
	sum := sha256.Sum256(decoded)
	rel := "assets/batch_image/imgbatch_reuse/item_9_0.png"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o750))
	require.NoError(t, os.WriteFile(abs, decoded, 0o640))
	asset := &BatchImageAsset{
		ID: 99, BatchID: job.BatchID, ItemID: 9, ImageIndex: 0,
		StorageKey: rel, MimeType: "image/png", ByteSize: int64(len(decoded)),
		SHA256: hex.EncodeToString(sum[:]), SourceProvider: BatchImageProviderGeminiAPI,
	}
	repo.assets[99] = asset

	got, err := ResolveOwnedBatchImageAssetBytes(context.Background(), repo, BatchImageOwner{UserID: 42, APIKeyID: 7}, 99)
	require.NoError(t, err)
	require.Equal(t, "image/png", got.MimeType)
	require.Equal(t, decoded, got.Data)

	_, err = ResolveOwnedBatchImageAssetBytes(context.Background(), repo, BatchImageOwner{UserID: 99, APIKeyID: 7}, 99)
	require.Error(t, err)

	_, err = ResolveOwnedBatchImageAssetBytes(context.Background(), repo, BatchImageOwner{UserID: 42, APIKeyID: 7}, 0)
	require.Error(t, err)
}

func TestBuildGeminiBatchJSONL_SendsImageGenerationSpecs(t *testing.T) {
	input := validGeminiBatchInput()
	input.ResponseMimeType = "image/jpeg"
	input.AspectRatio = "16:9"
	input.ImageSize = "2K"

	jsonl, err := BuildGeminiBatchJSONL(input)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(jsonl), &got))
	cfg := got["request"].(map[string]any)["generationConfig"].(map[string]any)
	require.Equal(t, "image/jpeg", cfg["responseMimeType"])
	imageCfg := cfg["imageConfig"].(map[string]any)
	require.Equal(t, "16:9", imageCfg["aspectRatio"])
	require.Equal(t, "2K", imageCfg["imageSize"])
}

func mustAbsBatchImageAsset(t *testing.T, rel string) string {
	t.Helper()
	abs, err := ResolveBatchImageAssetAbsPath(rel)
	require.NoError(t, err)
	return abs
}

func batchImageInt64Ptr(v int64) *int64 { return &v }

type expiredResultProvider struct{}

func (p *expiredResultProvider) Name() string { return BatchImageProviderGeminiAPI }
func (p *expiredResultProvider) SupportsAccount(*Account) bool {
	return true
}
func (p *expiredResultProvider) Submit(context.Context, *BatchImageJob, *Account, BatchImageInput) (*BatchProviderJob, error) {
	return nil, ErrBatchImageResultMissing
}
func (p *expiredResultProvider) Get(context.Context, *BatchImageJob, *Account) (*BatchProviderStatus, error) {
	return nil, ErrBatchImageResultMissing
}
func (p *expiredResultProvider) Cancel(context.Context, *BatchImageJob, *Account) error {
	return nil
}
func (p *expiredResultProvider) OpenResult(context.Context, *BatchImageJob, *Account) (io.ReadCloser, string, error) {
	return nil, "", ErrBatchImageResultMissing
}
func (p *expiredResultProvider) Cleanup(context.Context, *BatchImageJob, *Account, CleanupTarget) error {
	return nil
}

type staticJSONLProvider struct{ jsonl string }

func (p *staticJSONLProvider) Name() string { return BatchImageProviderGeminiAPI }
func (p *staticJSONLProvider) SupportsAccount(*Account) bool {
	return true
}
func (p *staticJSONLProvider) Submit(context.Context, *BatchImageJob, *Account, BatchImageInput) (*BatchProviderJob, error) {
	return nil, nil
}
func (p *staticJSONLProvider) Get(context.Context, *BatchImageJob, *Account) (*BatchProviderStatus, error) {
	return nil, nil
}
func (p *staticJSONLProvider) Cancel(context.Context, *BatchImageJob, *Account) error { return nil }
func (p *staticJSONLProvider) OpenResult(context.Context, *BatchImageJob, *Account) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader(p.jsonl)), "application/jsonl", nil
}
func (p *staticJSONLProvider) Cleanup(context.Context, *BatchImageJob, *Account, CleanupTarget) error {
	return nil
}
