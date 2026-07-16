package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestVideoGatewayRepositoryCreateAndGetTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()
	task := &service.VideoTask{ProviderAccountID: 7, Provider: "seedance", Model: "video-v1", TaskType: "text_to_video", Prompt: "test prompt", Status: service.VideoStatusQueued, CreatedBy: 9, CreationKey: "creation-1"}

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO video_tasks")).WillReturnRows(sqlmock.NewRows([]string{"id", "version", "created_at", "updated_at"}).AddRow(int64(31), int64(1), now, now))
	require.NoError(t, repo.CreateTask(context.Background(), task))
	require.EqualValues(t, 31, task.ID)

	mock.ExpectQuery(regexp.QuoteMeta("FROM video_tasks")).WithArgs(int64(31)).WillReturnRows(videoTaskRows(now).AddRow(int64(31), int64(7), "seedance", "video-v1", "text_to_video", "test prompt", "queued", "", "", "creation-1", int64(1), "pending", int64(9), now, now, nil))
	stored, err := repo.GetTask(context.Background(), 31)
	require.NoError(t, err)
	require.Equal(t, task.Prompt, stored.Prompt)
	require.Equal(t, task.CreationKey, stored.CreationKey)
	require.Equal(t, task.CreatedBy, stored.CreatedBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryClaimsRunnableTasks(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()
	mock.ExpectQuery("FOR UPDATE SKIP LOCKED").WithArgs(2, 90).WillReturnRows(videoTaskRows(now).AddRow(int64(4), int64(2), "mock", "mock-v1", "text_to_video", "prompt", "queued", "", "", "claim-4", int64(1), "pending", int64(13), now, now, nil))
	tasks, err := repo.ClaimRunnableTasks(context.Background(), 2, 90*time.Second)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.EqualValues(t, 4, tasks[0].ID)
	require.Equal(t, "claim-4", tasks[0].CreationKey)
	require.EqualValues(t, 13, tasks[0].CreatedBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoGatewayRepositoryNilDatabaseFailsExplicitly(t *testing.T) {
	repo := NewVideoGatewayRepository(nil)
	ctx := context.Background()

	require.ErrorContains(t, repo.CreateTask(ctx, &service.VideoTask{}), "database is required")
	_, err := repo.GetTask(ctx, 1)
	require.ErrorContains(t, err, "database is required")
	_, err = repo.ClaimRunnableTasks(ctx, 1, time.Second)
	require.ErrorContains(t, err, "database is required")
	_, err = repo.FinalizeTask(ctx, service.VideoTaskFinalization{Status: service.VideoStatusSucceeded})
	require.ErrorContains(t, err, "database is required")
}

func TestVideoGatewayRepositoryNilReceiverFailsExplicitly(t *testing.T) {
	var repo *videoGatewayRepository
	ctx := context.Background()

	require.ErrorContains(t, repo.CreateTask(ctx, &service.VideoTask{}), "database is required")
	_, err := repo.GetTask(ctx, 1)
	require.ErrorContains(t, err, "database is required")
	_, err = repo.ClaimRunnableTasks(ctx, 1, time.Second)
	require.ErrorContains(t, err, "database is required")
	_, err = repo.FinalizeTask(ctx, service.VideoTaskFinalization{Status: service.VideoStatusSucceeded})
	require.ErrorContains(t, err, "database is required")
}

func TestVideoGatewayRepositoryTerminalFinalizationIsIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE video_tasks").WithArgs(int64(8), int64(1), service.VideoStatusSucceeded, "https://assets.invalid/result.mp4", "", now).WillReturnRows(sqlmock.NewRows([]string{"status", "version"}).AddRow(service.VideoStatusSucceeded, int64(2)))
	mock.ExpectExec("INSERT INTO video_usage_logs").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	result, err := repo.FinalizeTask(context.Background(), service.VideoTaskFinalization{TaskID: 8, ExpectedVersion: 1, Status: service.VideoStatusSucceeded, ResultURL: "https://assets.invalid/result.mp4", CompletedAt: now})
	require.NoError(t, err)
	require.True(t, result.Applied)

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE video_tasks").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT status, version").WithArgs(int64(8)).WillReturnRows(sqlmock.NewRows([]string{"status", "version"}).AddRow(service.VideoStatusSucceeded, int64(2)))
	mock.ExpectRollback()
	replay, err := repo.FinalizeTask(context.Background(), service.VideoTaskFinalization{TaskID: 8, ExpectedVersion: 1, Status: service.VideoStatusSucceeded, ResultURL: "https://assets.invalid/result.mp4", CompletedAt: now})
	require.NoError(t, err)
	require.True(t, replay.Idempotent)
	require.NoError(t, mock.ExpectationsWereMet())
}

func videoTaskRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "provider_account_id", "provider", "model", "task_type", "prompt", "status", "result_url", "error_message", "creation_key", "version", "dispatch_state", "created_by", "created_at", "updated_at", "completed_at"})
}
