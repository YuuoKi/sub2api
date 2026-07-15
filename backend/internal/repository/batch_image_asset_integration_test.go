//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBatchImageRepository_UpsertAssetIdempotent(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newBatchImageRepositoryWithSQL(tx)
	batchID := batchImageTestID(t, "asset")

	_, err := repo.CreateBatchImageJob(ctx, service.CreateBatchImageJobParams{
		BatchID:          batchID,
		UserID:           1001,
		APIKeyID:         int64Ptr(2002),
		Provider:         service.BatchImageProviderGeminiAPI,
		Model:            "gemini-3.1-flash-image",
		ItemCount:        1,
		EstimatedCost:    0.01,
		ResponseMimeType: "image/png",
		AspectRatio:      "1:1",
		ImageSize:        "1K",
		ExecutionMode:    "mock",
	})
	require.NoError(t, err)

	require.NoError(t, repo.TransitionBatchImageJobStatus(ctx, batchID, service.BatchImageJobStatusSubmitted, service.BatchImageTransitionOptions{
		EventType: "submitted",
	}))

	require.NoError(t, repo.TransitionBatchImageJobStatus(ctx, batchID, service.BatchImageJobStatusIndexing, service.BatchImageTransitionOptions{
		EventType: "indexing_started",
	}))

	item, err := repo.CreateBatchImageItem(ctx, service.CreateBatchImageItemParams{
		JobID:      batchID,
		CustomID:   "cover_001",
		Status:     service.BatchImageItemStatusSuccess,
		ImageCount: 1,
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	first, err := repo.UpsertBatchImageAsset(ctx, service.UpsertBatchImageAssetParams{
		BatchID:        batchID,
		ItemID:         item.ID,
		ImageIndex:     0,
		StorageKey:     "assets/batch_image/" + batchID + "/item_1_0.png",
		MimeType:       "image/png",
		ByteSize:       12,
		SHA256:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ArchivedAt:     now,
		SourceProvider: service.BatchImageProviderGeminiAPI,
		SourceRef:      "files/out",
	})
	require.NoError(t, err)
	require.Positive(t, first.ID)

	second, err := repo.UpsertBatchImageAsset(ctx, service.UpsertBatchImageAssetParams{
		BatchID:        batchID,
		ItemID:         item.ID,
		ImageIndex:     0,
		StorageKey:     "assets/batch_image/" + batchID + "/item_1_0.png",
		MimeType:       "image/png",
		ByteSize:       12,
		SHA256:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ArchivedAt:     now,
		SourceProvider: service.BatchImageProviderGeminiAPI,
		SourceRef:      "files/out",
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	owned, err := repo.GetBatchImageAssetForOwner(ctx, 1001, 2002, first.ID)
	require.NoError(t, err)
	require.Equal(t, first.ID, owned.ID)

	_, err = repo.GetBatchImageAssetForOwner(ctx, 9999, 2002, first.ID)
	require.Error(t, err)

	job, err := repo.GetBatchImageJobByBatchID(ctx, batchID)
	require.NoError(t, err)
	require.Equal(t, "image/png", job.ResponseMimeType)
	require.Equal(t, "1:1", job.AspectRatio)
	require.Equal(t, "1K", job.ImageSize)
}

func int64Ptr(v int64) *int64 { return &v }
