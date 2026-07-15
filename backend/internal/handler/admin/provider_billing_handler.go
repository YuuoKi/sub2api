package admin

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ProviderBillingHandler struct {
	svc *service.ProviderBillingService
}

func NewProviderBillingHandler(svc *service.ProviderBillingService) *ProviderBillingHandler {
	return &ProviderBillingHandler{svc: svc}
}

type providerBillingHeaderForm struct {
	Provider           string `form:"provider" binding:"required"`
	ProviderAccountID  string `form:"provider_account_id" binding:"required"`
	BillingPeriodStart string `form:"billing_period_start" binding:"required"`
	BillingPeriodEnd   string `form:"billing_period_end" binding:"required"`
	Timezone           string `form:"timezone" binding:"required"`
	OriginalCurrency   string `form:"original_currency" binding:"required"`
	SourceType         string `form:"source_type" binding:"required"`
	InvoiceNumber      string `form:"invoice_number"`
}

func (h *ProviderBillingHandler) PreviewImport(c *gin.Context) {
	header, filename, raw, ok := h.readUpload(c)
	if !ok {
		return
	}
	parsed, err := h.svc.PreviewRawFile(c.Request.Context(), header, filename, raw)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"file_sha256": parsed.FileSHA256,
		"line_count":  len(parsed.Lines),
		"lines":       parsed.Lines,
		"duplicate":   false,
	})
}

func (h *ProviderBillingHandler) Import(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	header, filename, raw, ok := h.readUpload(c)
	if !ok {
		return
	}
	rec, err := h.svc.ImportRawFile(c.Request.Context(), header, filename, raw, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	matches, err := h.svc.ReconcileImport(c.Request.Context(), rec.ID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"import":  importToResponse(rec),
		"matches": matchesToResponse(matches),
	})
}

func (h *ProviderBillingHandler) ListImports(c *gin.Context) {
	provider := strings.TrimSpace(c.Query("provider"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.svc.ListImports(c.Request.Context(), provider, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, importToResponse(&items[i]))
	}
	response.Success(c, gin.H{"items": out})
}

func (h *ProviderBillingHandler) GetImport(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid import id")
		return
	}
	rec, err := h.svc.GetImport(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if rec == nil {
		response.ErrorFrom(c, service.ErrProviderBillingNotFound)
		return
	}
	response.Success(c, gin.H{
		"import": importToResponse(rec),
		"lines":  rec.Lines,
	})
}

func (h *ProviderBillingHandler) ListMatches(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid import id")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	matches, err := h.svc.ListMatches(c.Request.Context(), id, status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": matchesToResponse(matches)})
}

func (h *ProviderBillingHandler) Reconcile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid import id")
		return
	}
	matches, err := h.svc.ReconcileImport(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": matchesToResponse(matches)})
}

func (h *ProviderBillingHandler) PeriodSummary(c *gin.Context) {
	start, end, err := parsePeriodRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	items, err := h.svc.GetPeriodSummary(c.Request.Context(), start, end)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *ProviderBillingHandler) BossConclusions(c *gin.Context) {
	items, err := h.svc.GetBossConclusions(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// Boss homepage: only reconciled / has_diff / not_uploaded conclusions.
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, gin.H{
			"provider":             item.Provider,
			"provider_account_id":  item.ProviderAccountID,
			"billing_period_start": item.BillingPeriodStart,
			"billing_period_end":   item.BillingPeriodEnd,
			"conclusion":           item.Conclusion,
		})
	}
	response.Success(c, gin.H{"items": out})
}

func (h *ProviderBillingHandler) ExportMatchesCSV(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid import id")
		return
	}
	raw, err := h.svc.ExportMatchesCSV(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=provider_billing_matches_"+strconv.FormatInt(id, 10)+".csv")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", raw)
}

func (h *ProviderBillingHandler) readUpload(c *gin.Context) (service.ProviderBillingImportHeader, string, []byte, bool) {
	var form providerBillingHeaderForm
	if err := c.ShouldBind(&form); err != nil {
		response.BadRequest(c, "invalid import header: "+err.Error())
		return service.ProviderBillingImportHeader{}, "", nil, false
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(form.BillingPeriodStart))
	if err != nil {
		response.BadRequest(c, "billing_period_start must be RFC3339")
		return service.ProviderBillingImportHeader{}, "", nil, false
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(form.BillingPeriodEnd))
	if err != nil {
		response.BadRequest(c, "billing_period_end must be RFC3339")
		return service.ProviderBillingImportHeader{}, "", nil, false
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return service.ProviderBillingImportHeader{}, "", nil, false
	}
	f, err := fileHeader.Open()
	if err != nil {
		response.BadRequest(c, "unable to open upload")
		return service.ProviderBillingImportHeader{}, "", nil, false
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, int64(service.ProviderBillingMaxUploadBytes())+1))
	if err != nil {
		response.BadRequest(c, "unable to read upload")
		return service.ProviderBillingImportHeader{}, "", nil, false
	}
	header := service.ProviderBillingImportHeader{
		Provider:           strings.TrimSpace(form.Provider),
		ProviderAccountID:  strings.TrimSpace(form.ProviderAccountID),
		BillingPeriodStart: start.UTC(),
		BillingPeriodEnd:   end.UTC(),
		Timezone:           strings.TrimSpace(form.Timezone),
		OriginalCurrency:   strings.ToUpper(strings.TrimSpace(form.OriginalCurrency)),
		SourceType:         strings.ToLower(strings.TrimSpace(form.SourceType)),
		InvoiceNumber:      strings.TrimSpace(form.InvoiceNumber),
	}
	return header, fileHeader.Filename, raw, true
}

func parsePeriodRange(c *gin.Context) (time.Time, time.Time, error) {
	startRaw := strings.TrimSpace(c.Query("start"))
	endRaw := strings.TrimSpace(c.Query("end"))
	if startRaw == "" || endRaw == "" {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return start, end, nil
	}
	start, err := time.Parse(time.RFC3339, startRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := time.Parse(time.RFC3339, endRaw)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start.UTC(), end.UTC(), nil
}

func importToResponse(rec *service.ProviderBillingImportRecord) gin.H {
	if rec == nil {
		return gin.H{}
	}
	return gin.H{
		"id":                   rec.ID,
		"provider":             rec.Provider,
		"provider_account_id":  rec.ProviderAccountID,
		"billing_period_start": rec.BillingPeriodStart.UTC().Format(time.RFC3339),
		"billing_period_end":   rec.BillingPeriodEnd.UTC().Format(time.RFC3339),
		"timezone":             rec.Timezone,
		"original_currency":    rec.OriginalCurrency,
		"source_type":          rec.SourceType,
		"invoice_number":       rec.InvoiceNumber,
		"file_sha256":          rec.FileSHA256,
		"storage_key":          rec.StorageKey,
		"original_filename":    rec.OriginalFilename,
		"byte_size":            rec.ByteSize,
		"status":               rec.Status,
		"line_count":           rec.LineCount,
		"created_at":           rec.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func matchesToResponse(matches []service.ProviderBillingMatchResult) []gin.H {
	out := make([]gin.H, 0, len(matches))
	for _, m := range matches {
		item := gin.H{
			"id":                m.ID,
			"import_id":         m.ImportID,
			"external_line_id":  m.ExternalLineID,
			"match_status":      m.MatchStatus,
			"match_mode":        m.MatchMode,
			"internal_ref_type": m.InternalRefType,
			"internal_ref_id":   m.InternalRefID,
			"provider_amount":   m.ProviderAmount.String(),
			"internal_amount":   m.InternalAmount.String(),
			"provider_usage":    m.ProviderUsage.String(),
			"internal_usage":    m.InternalUsage.String(),
			"currency":          m.Currency,
			"model":             m.Model,
			"sku":               m.SKU,
			"diff":              m.DiffJSON,
		}
		if m.AccountDay != nil {
			item["account_day"] = m.AccountDay.UTC().Format("2006-01-02")
		}
		out = append(out, item)
	}
	return out
}
