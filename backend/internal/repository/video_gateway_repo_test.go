package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestVideoGatewayRepositoryListRunnableTasksClaimsRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{
		"id", "provider_account_id", "provider", "model", "task_type", "prompt", "negative_prompt",
		"reference_image_url", "reference_video_url", "aspect_ratio", "duration", "resolution",
		"status", "upstream_task_id", "result_url", "error_message", "cost_estimate", "poll_count",
		"created_by", "created_at", "updated_at", "completed_at", "display_name", "email", "username",
	}).AddRow(
		int64(42), int64(7), service.VideoProviderMock, "mock-video-v1", service.VideoTaskTypeTextToVideo,
		"claim me", "", "", "", "16:9", 5, "720p", service.VideoStatusQueued,
		"", "", "", 0.0, 0, int64(9), now, now, nil, "Mock Provider", "user@example.test", "operator",
	)

	mock.ExpectQuery(`(?s)WITH candidate_ids AS .*FOR UPDATE SKIP LOCKED.*UPDATE video_tasks vt.*worker_claimed_until.*RETURNING vt\.\*.*FROM claimed vt`).
		WithArgs(2, videoTaskClaimLeaseSeconds).
		WillReturnRows(rows)

	repo := NewVideoGatewayRepository(db)
	tasks, err := repo.ListRunnableTasks(context.Background(), 2)
	if err != nil {
		t.Fatalf("list runnable tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != 42 {
		t.Fatalf("unexpected claimed tasks: %#v", tasks)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestVideoGatewayRepositoryInsertUsageLogIsIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewVideoGatewayRepository(db)
	task := &service.VideoTask{
		ID:           42,
		Provider:     service.VideoProviderSeedance,
		Model:        "seedance-2-0-pro",
		Status:       service.VideoStatusSucceeded,
		CostEstimate: 0.12,
		Duration:     3,
	}

	mock.ExpectExec(regexp.QuoteMeta(insertVideoUsageLogSQL)).
		WithArgs(task.ID, task.Provider, task.Model, task.Status, task.CostEstimate, task.Duration).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.InsertUsageLog(context.Background(), task); err != nil {
		t.Fatalf("insert usage log: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestVideoGatewayRepositoryCreateDailyTrialTaskRejectsExistingReservation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	trialDate := time.Date(2026, 6, 28, 0, 0, 0, 0, time.Local)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)INSERT INTO video_daily_trial_reservations .*ON CONFLICT \(provider, created_by, trial_date\) DO NOTHING.*RETURNING id`).
		WithArgs(service.VideoProviderSeedance, int64(7), trialDate).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	repo := NewVideoGatewayRepository(db)
	reserved, err := repo.CreateDailyTrialTask(context.Background(), &service.VideoTask{}, service.VideoProviderSeedance, 7, trialDate)
	if err != nil {
		t.Fatalf("create daily trial task: %v", err)
	}
	if reserved {
		t.Fatalf("expected existing reservation to reject task creation")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestVideoGatewayRepositoryCreateDailyTrialTaskCreatesTaskInReservationTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	trialDate := time.Date(2026, 6, 28, 0, 0, 0, 0, time.Local)
	task := &service.VideoTask{
		ProviderAccountID: 3,
		Provider:          service.VideoProviderSeedance,
		Model:             "seedance-lite-test",
		TaskType:          service.VideoTaskTypeTextToVideo,
		Prompt:            "tiny real trial",
		AspectRatio:       "16:9",
		Duration:          3,
		Resolution:        "720p",
		Status:            service.VideoStatusQueued,
		CreatedBy:         7,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)INSERT INTO video_daily_trial_reservations .*ON CONFLICT \(provider, created_by, trial_date\) DO NOTHING.*RETURNING id`).
		WithArgs(service.VideoProviderSeedance, int64(7), trialDate).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(88)))
	mock.ExpectQuery(regexp.QuoteMeta(createVideoTaskSQL)).
		WithArgs(
			task.ProviderAccountID,
			task.Provider,
			task.Model,
			task.TaskType,
			task.Prompt,
			task.NegativePrompt,
			task.ReferenceImageURL,
			task.ReferenceVideoURL,
			task.AspectRatio,
			task.Duration,
			task.Resolution,
			task.Status,
			task.CreatedBy,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(42), now, now))
	mock.ExpectExec(`(?s)UPDATE video_daily_trial_reservations.*SET video_task_id = \$2.*WHERE id = \$1`).
		WithArgs(int64(88), int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewVideoGatewayRepository(db)
	reserved, err := repo.CreateDailyTrialTask(context.Background(), task, service.VideoProviderSeedance, 7, trialDate)
	if err != nil {
		t.Fatalf("create daily trial task: %v", err)
	}
	if !reserved {
		t.Fatalf("expected reservation to create task")
	}
	if task.ID != 42 {
		t.Fatalf("expected task id 42, got %d", task.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
