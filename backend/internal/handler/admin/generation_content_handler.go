package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	generationContentSampleLimit     = 20
	generationContentPromptPreview   = 120
	generationContentResponsePreview = 80
)

type GenerationContentHandler struct {
	repo           service.GenerationContentRepository
	cfg            *config.Config
	settingService *service.SettingService
}

func NewGenerationContentHandler(repo service.GenerationContentRepository, cfg *config.Config, settingService *service.SettingService) *GenerationContentHandler {
	return &GenerationContentHandler{repo: repo, cfg: cfg, settingService: settingService}
}

func (h *GenerationContentHandler) GetStats(c *gin.Context) {
	stats, err := h.repo.GetCaptureStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get generation content stats")
		return
	}
	if stats == nil {
		stats = &service.GenerationContentStats{}
	}
	response.Success(c, gin.H{
		"captured_today": stats.CapturedToday, "captured_week": stats.CapturedWeek,
		"distinct_employees": stats.DistinctEmployees, "distinct_teams": stats.DistinctTeams,
		"distinct_models": stats.DistinctModels, "total_bytes": stats.TotalBytes,
		"daily_rate": float64(stats.CapturedWeek) / 7, "daily_series": buildGenerationContentDailySeries(stats.DailySeries),
		"is_live": stats.Total > 0,
	})
}

func (h *GenerationContentHandler) GetSamples(c *gin.Context) {
	rows, err := h.repo.GetRecent(c.Request.Context(), generationContentSampleLimit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get generation content samples")
		return
	}
	samples := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		prompt, promptTruncated := truncateGenerationRunes(row.PromptRedacted, generationContentPromptPreview)
		result, resultTruncated := truncateGenerationRunes(row.ResponseRedacted, generationContentResponsePreview)
		samples = append(samples, gin.H{
			"employee_name": generationEmployeeName(row.Username, row.Email), "team_name": generationTeamName(row.GroupName),
			"task_id": row.TaskID, "model": row.Model, "video_status": row.VideoStatus,
			"cost_estimate": row.CostEstimate, "currency": row.Currency, "pricing_source": row.PricingSource,
			"created_at": row.CreatedAt, "prompt_preview": prompt, "response_preview": result,
			"total_bytes": row.PromptBytes + row.ResponseBytes, "adoption_status": row.AdoptionStatus,
			"quality_score": row.QualityScore, "adoption_notes": row.AdoptionNotes,
			"truncated": promptTruncated || resultTruncated || row.ResponseTruncated,
		})
	}
	response.Success(c, gin.H{"samples": samples, "is_live": len(rows) > 0, "usd_cny_rate": resolveGenerationUSDCNYRate(c, h.settingService)})
}

type updateGenerationContentAdoptionRequest struct {
	AdoptionStatus string   `json:"adoption_status"`
	QualityScore   *float64 `json:"quality_score"`
	Notes          string   `json:"notes"`
}

func (h *GenerationContentHandler) UpdateAdoption(c *gin.Context) {
	taskID, err := strconv.ParseInt(strings.TrimSpace(c.Param("task_id")), 10, 64)
	if err != nil || taskID <= 0 {
		response.BadRequest(c, "Invalid task_id")
		return
	}
	var request updateGenerationContentAdoptionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid adoption payload")
		return
	}
	status := strings.ToLower(strings.TrimSpace(request.AdoptionStatus))
	if !validGenerationAdoptionStatus(status) {
		response.BadRequest(c, "Invalid adoption_status")
		return
	}
	if request.QualityScore != nil && (*request.QualityScore < 0 || *request.QualityScore > 1) {
		response.BadRequest(c, "quality_score must be between 0 and 1")
		return
	}
	input := service.GenerationContentAdoptionInput{
		TaskID: taskID, AdoptionStatus: status, QualityScore: request.QualityScore,
		Notes: truncateGenerationNote(request.Notes, 2048),
	}
	if h == nil || h.cfg == nil || !h.cfg.Gateway.ContentCapture.Enabled {
		response.Success(c, gin.H{"enabled": false, "saved": false, "reason": "content_capture_disabled", "task_id": taskID,
			"adoption_status": status, "quality_score": request.QualityScore, "notes": input.Notes})
		return
	}
	result, err := h.repo.UpdateTaskAdoption(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save adoption feedback")
		return
	}
	saved := result != nil && result.Saved
	payload := gin.H{"enabled": true, "saved": saved, "task_id": taskID, "adoption_status": status,
		"quality_score": request.QualityScore, "notes": input.Notes}
	if !saved {
		payload["reason"] = "task_not_found"
	}
	response.Success(c, payload)
}

func (h *GenerationContentHandler) GetWeeklyReport(c *gin.Context) {
	start, end, err := generationWeeklyWindow(c.Query("start"), c.Query("end"), time.Now().UTC())
	if err != nil {
		response.BadRequest(c, "Invalid weekly report window")
		return
	}
	report, err := h.repo.GetWeeklyReport(c.Request.Context(), start, end)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get generation content weekly report")
		return
	}
	if report == nil {
		report = &service.GenerationContentWeeklyReport{PeriodStart: start, PeriodEnd: end}
	}
	response.Success(c, gin.H{
		"period_start": report.PeriodStart, "period_end": report.PeriodEnd, "entries": report.Entries,
		"video_tasks": report.VideoTasks, "total_cost_estimate": report.TotalCostEstimate,
		"adopted_count": report.AdoptedCount, "rejected_count": report.RejectedCount,
		"pending_count": report.PendingCount, "unreviewed_count": report.UnreviewedCount,
		"adoption_rate": report.AdoptionRate, "usd_cny_rate": resolveGenerationUSDCNYRate(c, h.settingService),
		"anomalies": gin.H{"failed_tasks": report.Anomalies.FailedTasks, "missing_task_joins": report.Anomalies.MissingTaskJoins, "truncated_rows": report.Anomalies.TruncatedRows},
		"markdown":  buildGenerationWeeklyMarkdown(report),
	})
}

func resolveGenerationUSDCNYRate(c *gin.Context, settings *service.SettingService) float64 {
	if settings == nil {
		return service.DefaultUSDCNYRate
	}
	return settings.GetUSDCNYRate(c.Request.Context())
}

func validGenerationAdoptionStatus(status string) bool {
	return status == "adopted" || status == "rejected" || status == "pending"
}

func truncateGenerationNote(note string, max int) string {
	note = strings.TrimSpace(note)
	if max <= 0 || utf8.RuneCountInString(note) <= max {
		return note
	}
	return string([]rune(note)[:max])
}

func generationWeeklyWindow(startRaw, endRaw string, now time.Time) (time.Time, time.Time, error) {
	end := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -7)
	var err error
	if strings.TrimSpace(startRaw) != "" {
		start, err = parseGenerationReportDate(startRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if strings.TrimSpace(endRaw) != "" {
		end, err = parseGenerationReportDate(endRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("start must be before end")
	}
	return start, end, nil
}

func parseGenerationReportDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func buildGenerationWeeklyMarkdown(report *service.GenerationContentWeeklyReport) string {
	return fmt.Sprintf(`# Weekly Production Ledger

- Period: %s to %s
- Entries: %d
- Video tasks: %d
- Cost estimate: %.6f USD
- Adoption: adopted=%d rejected=%d pending=%d unreviewed=%d rate=%.2f
- Anomalies: failed_tasks=%d missing_task_joins=%d truncated_rows=%d
`, report.PeriodStart.Format("2006-01-02"), report.PeriodEnd.Format("2006-01-02"), report.Entries, report.VideoTasks,
		report.TotalCostEstimate, report.AdoptedCount, report.RejectedCount, report.PendingCount, report.UnreviewedCount,
		report.AdoptionRate, report.Anomalies.FailedTasks, report.Anomalies.MissingTaskJoins, report.Anomalies.TruncatedRows)
}

func buildGenerationContentDailySeries(points []service.GenerationContentDailyPoint) []gin.H {
	counts := make(map[string]int64, len(points))
	for _, point := range points {
		counts[point.Date] = point.Count
	}
	now := time.Now().UTC()
	result := make([]gin.H, 0, 7)
	for day := 6; day >= 0; day-- {
		date := now.AddDate(0, 0, -day).Format("2006-01-02")
		result = append(result, gin.H{"date": date, "count": counts[date]})
	}
	return result
}

func truncateGenerationRunes(value string, max int) (string, bool) {
	if utf8.RuneCountInString(value) <= max {
		return value, false
	}
	return string([]rune(value)[:max]), true
}

func generationEmployeeName(username, email string) string {
	if username != "" {
		return username
	}
	if email != "" {
		return email
	}
	return "—"
}

func generationTeamName(name string) string {
	if name != "" {
		return name
	}
	return "—"
}
