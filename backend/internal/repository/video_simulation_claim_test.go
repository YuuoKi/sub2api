package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestClaimRunnableTasksSQLExcludesMockProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()

	// Real Seedance claim must exclude provider='mock' (or equivalently require seedance).
	mock.ExpectQuery(`(?is)provider\s*(<>|!=)\s*'mock'|provider\s*=\s*'seedance'`).
		WithArgs(2, 90).
		WillReturnRows(videoTaskRows(now).AddRow(
			int64(4), int64(21), int64(22), int64(2), "seedance", "doubao-seedance-2-0-260128", "text_to_video", "prompt", "queued",
			"", "", "", 4, "720p", nil, 0, "USD", nil, nil, nil, nil, nil, 0, "", "", "", "claim-4", int64(1), "pending", int64(13),
			now, now, nil, 0.2, "reserved", now, now, now, now, 0, nil, nil, nil, nil, nil, nil, 10.0, nil, nil, nil, nil, nil, nil,
		))

	tasks, err := repo.ClaimRunnableTasks(context.Background(), 2, 90*time.Second)
	require.NoError(t, err, "ClaimRunnableTasks SQL must exclude mock provider")
	require.Len(t, tasks, 1)
	require.NotEqual(t, "mock", tasks[0].Provider)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimMockRunnableTasksSQLRequiresMockProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoGatewayRepository{db: db}
	now := time.Now().UTC()

	mock.ExpectQuery(`(?is)provider\s*=\s*'mock'`).
		WithArgs(2, 90, mockSucceededContentReclaimSeconds).
		WillReturnRows(videoTaskRows(now).AddRow(
			int64(5), int64(21), int64(22), int64(2), "mock", "mock-video-v1", "text_to_video", "prompt", "queued",
			"", "", "", 4, "720p", nil, 0, "USD", "internal_simulation", "simulation-v1", nil, nil, nil, 0, "", "", "", "claim-mock-5", int64(1), "pending", int64(13),
			now, now, nil, 0, "none", nil, nil, nil, nil, 0, nil, nil, nil, nil, nil, nil, 0, nil, nil, nil, nil, nil, nil,
		))

	tasks, err := repo.ClaimMockRunnableTasks(context.Background(), 2, 90*time.Second)
	require.NoError(t, err, "ClaimMockRunnableTasks SQL must require provider=mock")
	require.Len(t, tasks, 1)
	require.Equal(t, "mock", tasks[0].Provider)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimMockRunnableTasksSQLIncludesSucceededWithoutContent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &videoGatewayRepository{db: db}
	now := time.Now().UTC()
	completed := now.Add(-time.Minute)

	// Production restart reclaim: claim SQL must surface succeeded mock tasks that still
	// lack ai_generation_content, within the reclaim window. Stub-forced claims are not enough.
	mock.ExpectQuery(`(?is)status\s*=\s*'succeeded'[\s\S]*ai_generation_content|ai_generation_content[\s\S]*status\s*=\s*'succeeded'`).
		WithArgs(1, 30, mockSucceededContentReclaimSeconds).
		WillReturnRows(videoTaskRows(now).AddRow(
			int64(6), int64(21), int64(22), int64(2), "mock", "mock-video-v1", "text_to_video", "prompt", "succeeded",
			"", "", "", 4, "720p", nil, 0, "USD", "internal_simulation", "simulation-v1", nil, nil, nil, 0, "", "", "", "claim-mock-6", int64(3), "pending", int64(13),
			now, now, completed, 0, "none", nil, nil, nil, nil, 0, nil, nil, nil, nil, nil, nil, 0, nil, nil, nil, nil, nil, nil,
		))

	tasks, err := repo.ClaimMockRunnableTasks(context.Background(), 1, 30*time.Second)
	require.NoError(t, err, "ClaimMockRunnableTasks must reclaim succeeded-without-content")
	require.Len(t, tasks, 1)
	require.Equal(t, "mock", tasks[0].Provider)
	require.Equal(t, "succeeded", tasks[0].Status)
	require.NoError(t, mock.ExpectationsWereMet())
}
