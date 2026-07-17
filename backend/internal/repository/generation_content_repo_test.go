package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestGenerationContentRepositoryCreateUsesIdempotentRequestKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewGenerationContentRepository(db)
	createdAt := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("ON CONFLICT (api_key_id, request_id) WHERE request_id <> '' DO NOTHING")).
		WithArgs("req-1", int64(2), int64(3), int64(4), int64(5), "claude", "hash", "prompt", "response", 6, 8, false, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(10), createdAt))
	apiKeyID, userID, groupID, accountID := int64(2), int64(3), int64(4), int64(5)
	content := &service.GenerationContent{
		RequestID: "req-1", APIKeyID: &apiKeyID, UserID: &userID, GroupID: &groupID, AccountID: &accountID,
		Model: "claude", RequestPayloadHash: "hash", PromptRedacted: "prompt", ResponseRedacted: "response",
		PromptBytes: 6, ResponseBytes: 8, RedactionVersion: 1,
	}
	if err := repo.Create(context.Background(), content); err != nil {
		t.Fatal(err)
	}
	if content.ID != 10 || !content.CreatedAt.Equal(createdAt) {
		t.Fatalf("insert evidence not scanned: %+v", content)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationContentWeeklyReportUsesCurrentChargedCostContracts(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewGenerationContentRepository(db)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)
	query := `(?s).*CASE\s+WHEN c\.task_id IS NOT NULL THEN COALESCE\(vul\.charged_cost_usd, 0\)\s+ELSE COALESCE\(ul\.actual_cost, 0\)\s+END.*LEFT JOIN video_usage_logs vul ON vul\.video_task_id = c\.task_id.*LEFT JOIN usage_logs ul ON ul\.api_key_id = c\.api_key_id AND ul\.request_id = NULLIF\(c\.request_id, ''\).*`
	mock.ExpectQuery(query).WithArgs(start, end).WillReturnRows(sqlmock.NewRows([]string{
		"entries", "video_tasks", "total_cost_estimate", "adopted_count", "rejected_count", "pending_count", "unreviewed_count", "failed_task_count", "missing_task_join_count", "truncated_count",
	}).AddRow(int64(4), int64(1), 1.25, int64(1), int64(1), int64(1), int64(1), int64(0), int64(0), int64(1)))

	report, err := repo.GetWeeklyReport(context.Background(), start, end)
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalCostEstimate != 1.25 || report.AdoptionRate != 0.25 || report.Anomalies.TruncatedRows != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationContentRecentSamplesExposeCostSource(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewGenerationContentRepository(db)
	query := `(?s).*vul\.charged_cost_usd.*ul\.actual_cost.*pricing_source.*`
	mock.ExpectQuery(query).WithArgs(20).WillReturnRows(sqlmock.NewRows([]string{
		"task_id", "model", "created_at", "prompt_redacted", "response_redacted", "prompt_bytes", "response_bytes", "response_truncated", "username", "email", "group_name", "adoption_status", "quality_score", "adoption_notes", "video_status", "cost_estimate", "currency", "pricing_source",
	}).AddRow(nil, "claude", time.Now().UTC(), "p", "r", int64(1), int64(1), false, "user", "u@example.test", "team", "", nil, "", "", 0.5, "USD", "usage_logs.actual_cost"))

	rows, err := repo.GetRecent(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].PricingSource != "usage_logs.actual_cost" || rows[0].CostEstimate != 0.5 {
		t.Fatalf("unexpected samples: %+v", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationContentRepositorySQLDoesNotUseLegacyCostHelpers(t *testing.T) {
	for _, query := range []string{generationContentRecentQuery, generationContentWeeklyQuery} {
		for _, forbidden := range []string{"dashboardUSDCNYRateCTE", "dashboardVideoCostUSDExpr", "cost_estimate"} {
			if strings.Contains(query, forbidden) {
				t.Fatalf("query references legacy contract %q", forbidden)
			}
		}
	}
}
