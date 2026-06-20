package admin

import (
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 采集内容看板（只读）的展示常量。
const (
	genContentSampleLimit   = 20  // 取最近 N 条供样本墙挑选展示
	genContentPromptPreview = 120 // prompt 预览最大 rune 数
	genContentRespPreview   = 80  // response 预览最大 rune 数
)

// GenerationContentHandler 提供 ai_generation_content 表的只读统计 + 样本端点（护城河看板）。
// 纯聚合无业务逻辑，故直接依赖 repo 接口，不引入额外 service 层。
type GenerationContentHandler struct {
	repo service.GenerationContentRepository
}

func NewGenerationContentHandler(repo service.GenerationContentRepository) *GenerationContentHandler {
	return &GenerationContentHandler{repo: repo}
}

// GetStats handles GET /api/v1/admin/generation-content/stats
func (h *GenerationContentHandler) GetStats(c *gin.Context) {
	stats, err := h.repo.GetCaptureStats(c.Request.Context())
	if err != nil {
		response.Error(c, 500, "Failed to get generation content stats")
		return
	}

	response.Success(c, gin.H{
		"captured_today":     stats.CapturedToday,
		"captured_week":      stats.CapturedWeek,
		"distinct_employees": stats.DistinctEmployees,
		"distinct_teams":     stats.DistinctTeams,
		"distinct_models":    stats.DistinctModels,
		"total_bytes":        stats.TotalBytes,
		"daily_rate":         float64(stats.CapturedWeek) / 7.0,
		"daily_series":       buildDailySeries(stats.DailySeries),
		// 有任何数据即为实时态；空表 → 看板显"示例(未开启采集)"，绝不伪造聚合。
		"is_live": stats.Total > 0,
	})
}

// GetSamples handles GET /api/v1/admin/generation-content/samples
func (h *GenerationContentHandler) GetSamples(c *gin.Context) {
	rows, err := h.repo.GetRecent(c.Request.Context(), genContentSampleLimit)
	if err != nil {
		response.Error(c, 500, "Failed to get generation content samples")
		return
	}

	samples := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		promptPreview, pTrunc := truncateRunes(r.PromptRedacted, genContentPromptPreview)
		respPreview, rTrunc := truncateRunes(r.ResponseRedacted, genContentRespPreview)
		samples = append(samples, gin.H{
			"employee_name":    employeeName(r.Username, r.Email),
			"team_name":        teamName(r.GroupName),
			"model":            r.Model,
			"created_at":       r.CreatedAt,
			"prompt_preview":   promptPreview,
			"response_preview": respPreview,
			"total_bytes":      r.PromptBytes + r.ResponseBytes,
			// 截断标:预览被裁 或 上游响应被限容截断,任一即标记。
			"truncated": pTrunc || rTrunc || r.ResponseTruncated,
		})
	}

	response.Success(c, gin.H{
		"samples": samples,
		"is_live": len(rows) > 0,
	})
}

// buildDailySeries 把仓库返回的稀疏日序列零填充为最近 7 个 UTC 日点(老→新)，前端永远拿到 7 个点。
func buildDailySeries(points []service.GenerationContentDailyPoint) []gin.H {
	counts := make(map[string]int64, len(points))
	for _, p := range points {
		counts[p.Date] = p.Count
	}
	now := time.Now().UTC()
	out := make([]gin.H, 0, 7)
	for i := 6; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, gin.H{"date": d, "count": counts[d]})
	}
	return out
}

// truncateRunes 返回最多 max 个 rune(UTF-8 安全,中文不截半)及是否发生截断。
func truncateRunes(s string, max int) (string, bool) {
	if utf8.RuneCountInString(s) <= max {
		return s, false
	}
	return string([]rune(s)[:max]), true
}

// employeeName 优先用户名,退化为邮箱,再退化为占位符。
func employeeName(username, email string) string {
	if username != "" {
		return username
	}
	if email != "" {
		return email
	}
	return "—"
}

// teamName 团队名退化为占位符。
func teamName(name string) string {
	if name != "" {
		return name
	}
	return "—"
}
