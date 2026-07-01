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

func TestGenerationContentRepositoryCreateVideoTaskContentUsesTaskIDConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	userID := int64(7)
	taskID := int64(99)
	content := &service.GenerationContent{
		UserID:            &userID,
		TaskID:            &taskID,
		Model:             "mock-video-v1",
		PromptRedacted:    `{"prompt":"safe"}`,
		ResponseRedacted:  `{"metadata_summary":true}`,
		PromptBytes:       17,
		ResponseBytes:     25,
		ResponseTruncated: false,
		RedactionVersion:  2,
	}
	now := time.Now().UTC()

	mock.ExpectQuery(`(?s)INSERT INTO ai_generation_content .*task_id.*ON CONFLICT \(task_id\) WHERE task_id IS NOT NULL DO NOTHING.*RETURNING id, created_at`).
		WithArgs(
			content.RequestID,
			nil,
			userID,
			nil,
			nil,
			taskID,
			content.Model,
			content.RequestPayloadHash,
			content.PromptRedacted,
			content.ResponseRedacted,
			content.PromptBytes,
			content.ResponseBytes,
			content.ResponseTruncated,
			content.RedactionVersion,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(123), now))

	repo := NewGenerationContentRepository(db)
	if err := repo.CreateVideoTaskContent(context.Background(), content); err != nil {
		t.Fatalf("create video task content: %v", err)
	}
	if content.ID != 123 || !content.CreatedAt.Equal(now) {
		t.Fatalf("expected returned id/created_at, got id=%d created_at=%s", content.ID, content.CreatedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGenerationContentRepositoryCreateVideoTaskContentDuplicateIsNoOp(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	taskID := int64(99)
	content := &service.GenerationContent{TaskID: &taskID, Model: "mock-video-v1"}

	mock.ExpectQuery(regexp.QuoteMeta(`
INSERT INTO ai_generation_content (
    request_id, api_key_id, user_id, group_id, account_id, task_id, model, request_payload_hash,
    prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14
)
ON CONFLICT (task_id) WHERE task_id IS NOT NULL DO NOTHING
RETURNING id, created_at`)).
		WillReturnError(sql.ErrNoRows)

	repo := NewGenerationContentRepository(db)
	if err := repo.CreateVideoTaskContent(context.Background(), content); err != nil {
		t.Fatalf("duplicate video task content should be no-op: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
