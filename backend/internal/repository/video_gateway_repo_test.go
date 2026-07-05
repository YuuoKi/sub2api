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
		"reference_image_url", "reference_video_url", "content_json", "has_video_input",
		"aspect_ratio", "duration", "resolution", "generate_audio", "watermark", "camera_fixed", "return_last_frame",
		"usage_total_tokens", "actual_resolution", "actual_duration", "last_frame_url",
		"status", "upstream_task_id", "result_url", "error_message", "cost_estimate", "poll_count",
		"created_by", "created_at", "updated_at", "completed_at", "display_name", "email", "username",
	}).AddRow(
		int64(42), int64(7), service.VideoProviderMock, "mock-video-v1", service.VideoTaskTypeTextToVideo,
		"claim me", "", "", "", []byte(`[]`), false, "16:9", 5, "720p", nil, nil, nil, nil,
		nil, nil, nil, nil, service.VideoStatusQueued, "", "", "", 0.0, 0, int64(9), now, now, nil,
		"Mock Provider", "user@example.test", "operator",
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
		ID:             42,
		Provider:       service.VideoProviderSeedance,
		Model:          "seedance-2-0-pro",
		Status:         service.VideoStatusSucceeded,
		CostEstimate:   0.12,
		Duration:       3,
		Currency:       service.BillingCurrencyCNY,
		PricingSource:  service.PricingSourceProviderUsage,
		PricingVersion: service.VideoPricingVersionSeedance202603,
	}

	mock.ExpectExec(regexp.QuoteMeta(insertVideoUsageLogSQL)).
		WithArgs(task.ID, task.Provider, task.Model, task.Status, task.CostEstimate, task.Duration, task.Currency, task.PricingSource, task.PricingVersion).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.InsertUsageLog(context.Background(), task); err != nil {
		t.Fatalf("insert usage log: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(insertVideoUsageLogSQL)).
		WithArgs(task.ID, task.Provider, task.Model, task.Status, task.CostEstimate, task.Duration, task.Currency, task.PricingSource, task.PricingVersion).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := repo.InsertUsageLog(context.Background(), task); err != nil {
		t.Fatalf("second insert usage log should be idempotent: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestVideoGatewayRepositoryClaimVideoBalanceCharge(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewVideoGatewayRepository(db)

	claimedAt := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)UPDATE video_tasks.*balance_charged_at = NOW\(\).*WHERE id = \$1 AND balance_charged_at IS NULL.*RETURNING balance_charged_at`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"balance_charged_at"}).AddRow(claimedAt))
	gotClaimedAt, claimed, err := repo.ClaimVideoBalanceCharge(context.Background(), 42)
	if err != nil {
		t.Fatalf("claim balance charge: %v", err)
	}
	if !claimed {
		t.Fatalf("expected first claim to succeed")
	}
	if !gotClaimedAt.Equal(claimedAt) {
		t.Fatalf("claimed_at = %s, want %s", gotClaimedAt, claimedAt)
	}

	mock.ExpectQuery(`(?s)UPDATE video_tasks.*balance_charged_at = NOW\(\).*WHERE id = \$1 AND balance_charged_at IS NULL.*RETURNING balance_charged_at`).
		WithArgs(int64(42)).
		WillReturnError(sql.ErrNoRows)
	gotClaimedAt, claimed, err = repo.ClaimVideoBalanceCharge(context.Background(), 42)
	if err != nil {
		t.Fatalf("second claim balance charge: %v", err)
	}
	if claimed {
		t.Fatalf("expected second claim to be idempotent")
	}
	if !gotClaimedAt.IsZero() {
		t.Fatalf("unclaimed task should return zero claim time, got %s", gotClaimedAt)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestVideoGatewayRepositoryClearVideoBalanceChargeIfClaimedAt(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewVideoGatewayRepository(db)
	claimedAt := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)

	mock.ExpectExec(`(?s)UPDATE video_tasks.*balance_charged_at = NULL.*WHERE id = \$1 AND balance_charged_at = \$2`).
		WithArgs(int64(42), claimedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	cleared, err := repo.ClearVideoBalanceChargeIfClaimedAt(context.Background(), 42, claimedAt)
	if err != nil {
		t.Fatalf("clear balance charge: %v", err)
	}
	if !cleared {
		t.Fatalf("expected compare-clear to affect the matching claim")
	}

	mock.ExpectExec(`(?s)UPDATE video_tasks.*balance_charged_at = NULL.*WHERE id = \$1 AND balance_charged_at = \$2`).
		WithArgs(int64(42), claimedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	cleared, err = repo.ClearVideoBalanceChargeIfClaimedAt(context.Background(), 42, claimedAt)
	if err != nil {
		t.Fatalf("second clear balance charge: %v", err)
	}
	if cleared {
		t.Fatalf("expected stale claim timestamp not to clear")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestVideoGatewayRepositoryListUnchargedSucceededVideoTasks(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{
		"id", "provider_account_id", "provider", "model", "task_type", "prompt", "negative_prompt",
		"reference_image_url", "reference_video_url", "content_json", "has_video_input",
		"aspect_ratio", "duration", "resolution", "generate_audio", "watermark", "camera_fixed", "return_last_frame",
		"usage_total_tokens", "actual_resolution", "actual_duration", "last_frame_url",
		"status", "upstream_task_id", "result_url", "error_message", "cost_estimate", "poll_count",
		"created_by", "created_at", "updated_at", "completed_at", "display_name", "email", "username",
	}).AddRow(
		int64(42), int64(7), service.VideoProviderSeedance, "doubao-seedance-2-0-260128", service.VideoTaskTypeTextToVideo,
		"charge me", "", "", "", []byte(`[]`), false, "16:9", 5, "720p", nil, nil, nil, nil,
		int64(102960), "720p", nil, nil, service.VideoStatusSucceeded, "upstream-42", "https://result.example/video.mp4", "", 4.73616, 2, int64(9), now, now, now,
		"Seedance", "user@example.test", "operator",
	)

	mock.ExpectQuery(`(?s)WHERE vt\.status = 'succeeded'.*vt\.balance_charged_at IS NULL.*ORDER BY vt\.completed_at ASC NULLS LAST, vt\.updated_at ASC, vt\.id ASC`).
		WithArgs(3).
		WillReturnRows(rows)

	repo := NewVideoGatewayRepository(db)
	tasks, err := repo.ListUnchargedSucceededVideoTasks(context.Background(), 3)
	if err != nil {
		t.Fatalf("list uncharged succeeded video tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != 42 || tasks[0].Status != service.VideoStatusSucceeded {
		t.Fatalf("unexpected uncharged tasks: %#v", tasks)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestVideoGatewayRepositoryUpdateTaskPersistsPollResponseDetails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewVideoGatewayRepository(db)
	now := time.Now().UTC()
	completedAt := now.Add(time.Second)
	tokens := int64(654321)
	actualDuration := 12
	task := &service.VideoTask{
		ID:               42,
		Status:           service.VideoStatusSucceeded,
		UpstreamTaskID:   "seedance-task-42",
		ResultURL:        "https://ark-content.cn-beijing.volces.com/v/ok.mp4",
		ErrorMessage:     "",
		CostEstimate:     0.12,
		CompletedAt:      &completedAt,
		PollCount:        3,
		UsageTotalTokens: &tokens,
		ActualResolution: "1080p",
		ActualDuration:   &actualDuration,
		LastFrameURL:     "https://ark-content.cn-beijing.volces.com/i/last.png",
	}

	mock.ExpectQuery(`(?s)UPDATE video_tasks.*usage_total_tokens = \$9.*actual_resolution = \$10.*actual_duration = \$11.*last_frame_url = \$12`).
		WithArgs(
			task.ID,
			task.Status,
			task.UpstreamTaskID,
			task.ResultURL,
			task.ErrorMessage,
			task.CostEstimate,
			completedAt,
			task.PollCount,
			tokens,
			task.ActualResolution,
			actualDuration,
			task.LastFrameURL,
		).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

	if err := repo.UpdateTask(context.Background(), task); err != nil {
		t.Fatalf("update task: %v", err)
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
	generateAudio := false
	task := &service.VideoTask{
		ProviderAccountID: 3,
		Provider:          service.VideoProviderSeedance,
		Model:             "seedance-lite-test",
		TaskType:          service.VideoTaskTypeTextToVideo,
		Prompt:            "tiny real trial",
		Content: []service.VideoTaskContentItem{
			{Type: service.VideoContentTypeVideoURL, Role: service.VideoContentRoleReferenceVideo, URL: "https://assets.example.com/ref.mp4"},
		},
		HasVideoInput: true,
		AspectRatio:   "16:9",
		Duration:      3,
		Resolution:    "720p",
		GenerateAudio: &generateAudio,
		Status:        service.VideoStatusQueued,
		CreatedBy:     7,
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
			`[{"type":"video_url","role":"reference_video","url":"https://assets.example.com/ref.mp4"}]`,
			true,
			task.AspectRatio,
			task.Duration,
			task.Resolution,
			false,
			nil,
			nil,
			nil,
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
