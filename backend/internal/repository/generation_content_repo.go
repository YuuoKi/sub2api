package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type generationContentRepository struct {
	db *sql.DB
}

// NewGenerationContentRepository builds the raw-SQL repository for the M1 采集口
// side table ai_generation_content (mirrors the content_moderation_logs precedent:
// append-only, no ent schema/codegen).
func NewGenerationContentRepository(db *sql.DB) service.GenerationContentRepository {
	return &generationContentRepository{db: db}
}

func (r *generationContentRepository) Create(ctx context.Context, content *service.GenerationContent) error {
	if r == nil || r.db == nil || content == nil {
		return nil
	}
	var apiKeyID, userID, groupID, accountID any
	if content.APIKeyID != nil {
		apiKeyID = *content.APIKeyID
	}
	if content.UserID != nil {
		userID = *content.UserID
	}
	if content.GroupID != nil {
		groupID = *content.GroupID
	}
	if content.AccountID != nil {
		accountID = *content.AccountID
	}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO ai_generation_content (
    request_id, api_key_id, user_id, group_id, account_id, model, request_payload_hash,
    prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13
)
ON CONFLICT (api_key_id, request_id) WHERE request_id <> '' DO NOTHING
RETURNING id, created_at`,
		content.RequestID, apiKeyID, userID, groupID, accountID, content.Model, content.RequestPayloadHash,
		content.PromptRedacted, content.ResponseRedacted, content.PromptBytes, content.ResponseBytes, content.ResponseTruncated, content.RedactionVersion,
	).Scan(&content.ID, &content.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// 命中 (api_key_id, request_id) 唯一索引 → 该请求已采集，幂等 no-op。
		return nil
	}
	if err != nil {
		return fmt.Errorf("insert ai generation content: %w", err)
	}
	return nil
}
