package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type generationContentRepository struct {
	db *sql.DB
}

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
	const query = `
INSERT INTO ai_generation_content (
    request_id, api_key_id, user_id, group_id, account_id, model, request_payload_hash,
    prompt_redacted, response_redacted, prompt_bytes, response_bytes, response_truncated, redaction_version
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (api_key_id, request_id) WHERE request_id <> '' DO NOTHING
RETURNING id, created_at`
	err := scanSingleRow(ctx, r.db, query, []any{
		content.RequestID, apiKeyID, userID, groupID, accountID, content.Model, content.RequestPayloadHash,
		content.PromptRedacted, content.ResponseRedacted, content.PromptBytes, content.ResponseBytes,
		content.ResponseTruncated, content.RedactionVersion,
	}, &content.ID, &content.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("insert ai generation content: %w", err)
	}
	return nil
}

func (r *generationContentRepository) UpdateTaskAdoption(ctx context.Context, input service.GenerationContentAdoptionInput) (*service.GenerationContentAdoption, error) {
	result := &service.GenerationContentAdoption{
		TaskID: input.TaskID, AdoptionStatus: input.AdoptionStatus, QualityScore: input.QualityScore, Notes: input.Notes,
	}
	if r == nil || r.db == nil || input.TaskID <= 0 {
		return result, nil
	}
	const query = `
UPDATE ai_generation_content
SET adoption_status = $2, quality_score = $3, adoption_notes = $4
WHERE task_id = $1
RETURNING task_id, adoption_status, quality_score, adoption_notes`
	var score sql.NullFloat64
	var notes string
	err := scanSingleRow(ctx, r.db, query, []any{input.TaskID, input.AdoptionStatus, input.QualityScore, input.Notes},
		&result.TaskID, &result.AdoptionStatus, &score, &notes)
	if errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update generation content adoption: %w", err)
	}
	if score.Valid {
		result.QualityScore = &score.Float64
	} else {
		result.QualityScore = nil
	}
	result.Notes = notes
	result.Saved = true
	return result, nil
}

func (r *generationContentRepository) GetCaptureStats(ctx context.Context) (*service.GenerationContentStats, error) {
	stats := &service.GenerationContentStats{}
	if r == nil || r.db == nil {
		return stats, nil
	}
	const aggregateQuery = `
SELECT COUNT(*),
       COUNT(*) FILTER (WHERE created_at >= date_trunc('day', NOW() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC'),
       COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days'),
       COUNT(DISTINCT user_id), COUNT(DISTINCT group_id), COUNT(DISTINCT NULLIF(model, '')),
       COALESCE(SUM(prompt_bytes + response_bytes), 0)
FROM ai_generation_content`
	if err := scanSingleRow(ctx, r.db, aggregateQuery, nil,
		&stats.Total, &stats.CapturedToday, &stats.CapturedWeek, &stats.DistinctEmployees,
		&stats.DistinctTeams, &stats.DistinctModels, &stats.TotalBytes); err != nil {
		return nil, fmt.Errorf("generation content stats: %w", err)
	}
	const seriesQuery = `
SELECT TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD'), COUNT(*)
FROM ai_generation_content
WHERE created_at >= (date_trunc('day', NOW() AT TIME ZONE 'UTC') - INTERVAL '6 days') AT TIME ZONE 'UTC'
GROUP BY 1 ORDER BY 1`
	rows, err := r.db.QueryContext(ctx, seriesQuery)
	if err != nil {
		return nil, fmt.Errorf("generation content daily series: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var point service.GenerationContentDailyPoint
		if err := rows.Scan(&point.Date, &point.Count); err != nil {
			return nil, fmt.Errorf("scan generation content daily series: %w", err)
		}
		stats.DailySeries = append(stats.DailySeries, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate generation content daily series: %w", err)
	}
	return stats, nil
}

const generationContentRecentQuery = `
SELECT c.task_id, c.model, c.created_at, c.prompt_redacted, c.response_redacted,
       c.prompt_bytes, c.response_bytes, c.response_truncated,
       COALESCE(u.username, ''), COALESCE(u.email, ''), COALESCE(g.name, ''),
       c.adoption_status, c.quality_score, c.adoption_notes, COALESCE(vt.status, ''),
       CASE
           WHEN c.task_id IS NOT NULL THEN COALESCE(vul.charged_cost_usd, 0)
           ELSE COALESCE(ul.actual_cost, 0)
       END AS cost_amount,
       CASE
           WHEN c.task_id IS NOT NULL THEN COALESCE(NULLIF(vul.currency, ''), 'USD')
           ELSE 'USD'
       END AS currency,
       CASE
           WHEN c.task_id IS NOT NULL AND vul.id IS NOT NULL THEN 'video_usage_logs.charged_cost_usd'
           WHEN c.task_id IS NULL AND ul.id IS NOT NULL THEN 'usage_logs.actual_cost'
           ELSE 'unknown'
       END AS pricing_source
FROM ai_generation_content c
LEFT JOIN users u ON u.id = c.user_id
LEFT JOIN groups g ON g.id = c.group_id
LEFT JOIN video_tasks vt ON vt.id = c.task_id
LEFT JOIN video_usage_logs vul ON vul.video_task_id = c.task_id
LEFT JOIN usage_logs ul ON ul.api_key_id = c.api_key_id AND ul.request_id = NULLIF(c.request_id, '')
ORDER BY c.created_at DESC
LIMIT $1`

func (r *generationContentRepository) GetRecent(ctx context.Context, limit int) ([]service.GenerationContentSample, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, generationContentRecentQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent generation content: %w", err)
	}
	defer rows.Close()
	result := make([]service.GenerationContentSample, 0, limit)
	for rows.Next() {
		var sample service.GenerationContentSample
		var taskID sql.NullInt64
		var score sql.NullFloat64
		if err := rows.Scan(
			&taskID, &sample.Model, &sample.CreatedAt, &sample.PromptRedacted, &sample.ResponseRedacted,
			&sample.PromptBytes, &sample.ResponseBytes, &sample.ResponseTruncated,
			&sample.Username, &sample.Email, &sample.GroupName, &sample.AdoptionStatus, &score,
			&sample.AdoptionNotes, &sample.VideoStatus, &sample.CostEstimate, &sample.Currency, &sample.PricingSource,
		); err != nil {
			return nil, fmt.Errorf("scan recent generation content: %w", err)
		}
		if taskID.Valid {
			sample.TaskID = &taskID.Int64
		}
		if score.Valid {
			sample.QualityScore = &score.Float64
		}
		result = append(result, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent generation content: %w", err)
	}
	return result, nil
}

const generationContentWeeklyQuery = `
SELECT COUNT(*) AS entries,
       COUNT(c.task_id) AS video_tasks,
       COALESCE(SUM(CASE
           WHEN c.task_id IS NOT NULL THEN COALESCE(vul.charged_cost_usd, 0)
           ELSE COALESCE(ul.actual_cost, 0)
       END), 0)::float8 AS total_cost,
       COUNT(*) FILTER (WHERE c.adoption_status = 'adopted'),
       COUNT(*) FILTER (WHERE c.adoption_status = 'rejected'),
       COUNT(*) FILTER (WHERE c.adoption_status = 'pending'),
       COUNT(*) FILTER (WHERE c.adoption_status = ''),
       COUNT(*) FILTER (WHERE vt.status = 'failed'),
       COUNT(*) FILTER (WHERE c.task_id IS NOT NULL AND vt.id IS NULL),
       COUNT(*) FILTER (WHERE c.response_truncated = TRUE)
FROM ai_generation_content c
LEFT JOIN video_tasks vt ON vt.id = c.task_id
LEFT JOIN video_usage_logs vul ON vul.video_task_id = c.task_id
LEFT JOIN usage_logs ul ON ul.api_key_id = c.api_key_id AND ul.request_id = NULLIF(c.request_id, '')
WHERE c.created_at >= $1 AND c.created_at < $2`

func (r *generationContentRepository) GetWeeklyReport(ctx context.Context, start, end time.Time) (*service.GenerationContentWeeklyReport, error) {
	report := &service.GenerationContentWeeklyReport{PeriodStart: start, PeriodEnd: end}
	if r == nil || r.db == nil {
		return report, nil
	}
	if err := scanSingleRow(ctx, r.db, generationContentWeeklyQuery, []any{start, end},
		&report.Entries, &report.VideoTasks, &report.TotalCostEstimate, &report.AdoptedCount,
		&report.RejectedCount, &report.PendingCount, &report.UnreviewedCount,
		&report.Anomalies.FailedTasks, &report.Anomalies.MissingTaskJoins,
		&report.Anomalies.TruncatedRows); err != nil {
		return nil, fmt.Errorf("get generation content weekly report: %w", err)
	}
	if report.Entries > 0 {
		report.AdoptionRate = float64(report.AdoptedCount) / float64(report.Entries)
	}
	return report, nil
}
