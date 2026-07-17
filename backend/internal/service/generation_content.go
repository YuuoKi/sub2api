package service

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	defaultGenerationPromptMaxBytes   = 256 * 1024
	defaultGenerationResponseMaxBytes = 64 * 1024
	maxGenerationPromptMaxBytes       = 1024 * 1024
	maxGenerationResponseMaxBytes     = 256 * 1024
)

type GenerationContent struct {
	ID                 int64
	RequestID          string
	APIKeyID           *int64
	UserID             *int64
	GroupID            *int64
	AccountID          *int64
	TaskID             *int64
	Model              string
	RequestPayloadHash string
	PromptRedacted     string
	ResponseRedacted   string
	PromptBytes        int
	ResponseBytes      int
	ResponseTruncated  bool
	RedactionVersion   int
	AdoptionStatus     string
	QualityScore       *float64
	AdoptionNotes      string
	CreatedAt          time.Time
}

type GenerationContentRepository interface {
	Create(context.Context, *GenerationContent) error
	GetCaptureStats(context.Context) (*GenerationContentStats, error)
	GetRecent(context.Context, int) ([]GenerationContentSample, error)
	UpdateTaskAdoption(context.Context, GenerationContentAdoptionInput) (*GenerationContentAdoption, error)
	GetWeeklyReport(context.Context, time.Time, time.Time) (*GenerationContentWeeklyReport, error)
}

type GenerationContentDailyPoint struct {
	Date  string
	Count int64
}

type GenerationContentStats struct {
	Total             int64
	CapturedToday     int64
	CapturedWeek      int64
	DistinctEmployees int64
	DistinctTeams     int64
	DistinctModels    int64
	TotalBytes        int64
	DailySeries       []GenerationContentDailyPoint
}

type GenerationContentSample struct {
	TaskID            *int64
	Model             string
	CreatedAt         time.Time
	PromptRedacted    string
	ResponseRedacted  string
	PromptBytes       int64
	ResponseBytes     int64
	ResponseTruncated bool
	Username          string
	Email             string
	GroupName         string
	AdoptionStatus    string
	QualityScore      *float64
	AdoptionNotes     string
	VideoStatus       string
	CostEstimate      float64
	Currency          string
	PricingSource     string
}

type GenerationContentAdoptionInput struct {
	TaskID         int64
	AdoptionStatus string
	QualityScore   *float64
	Notes          string
}

type GenerationContentAdoption struct {
	TaskID         int64
	AdoptionStatus string
	QualityScore   *float64
	Notes          string
	Saved          bool
}

type GenerationContentWeeklyAnomalies struct {
	FailedTasks      int64
	MissingTaskJoins int64
	TruncatedRows    int64
}

type GenerationContentWeeklyReport struct {
	PeriodStart       time.Time
	PeriodEnd         time.Time
	Entries           int64
	VideoTasks        int64
	TotalCostEstimate float64
	AdoptedCount      int64
	RejectedCount     int64
	PendingCount      int64
	UnreviewedCount   int64
	AdoptionRate      float64
	Anomalies         GenerationContentWeeklyAnomalies
}

type GenerationContentCaptureArgs struct {
	RequestID          string
	UserID             int64
	APIKeyID           int64
	GroupID            *int64
	AccountID          int64
	Model              string
	RequestPayloadHash string
	PromptBody         []byte
	PromptBytes        int
	Result             *ForwardResult
}

type GenerationPromptSnapshot struct {
	Body          []byte
	OriginalBytes int
	Truncated     bool
}

type GenerationContentCollector struct {
	repo GenerationContentRepository
	cfg  *config.Config
}

func NewGenerationContentCollector(repo GenerationContentRepository, cfg *config.Config) *GenerationContentCollector {
	return &GenerationContentCollector{repo: repo, cfg: cfg}
}

func (c *GenerationContentCollector) promptMaxBytes() int {
	if c != nil && c.cfg != nil {
		return boundedGenerationBytes(c.cfg.Gateway.ContentCapture.PromptMaxBytes, defaultGenerationPromptMaxBytes, maxGenerationPromptMaxBytes)
	}
	return defaultGenerationPromptMaxBytes
}

func (c *GenerationContentCollector) responseMaxBytes() int {
	if c != nil && c.cfg != nil {
		return boundedGenerationBytes(c.cfg.Gateway.ContentCapture.ResponseMaxBytes, defaultGenerationResponseMaxBytes, maxGenerationResponseMaxBytes)
	}
	return defaultGenerationResponseMaxBytes
}

func boundedGenerationBytes(value, fallback, maximum int) int {
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func (c *GenerationContentCollector) Collect(ctx context.Context, args GenerationContentCaptureArgs) {
	if c == nil || c.repo == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.LegacyPrintf("service.gateway", "generation content collect panic: %v (request_id=%s)", recovered, args.RequestID)
		}
	}()

	promptInput := truncateValidUTF8(args.PromptBody, c.promptMaxBytes())
	promptRedacted := truncateStringUTF8(redactGenerationPrompt(promptInput), c.promptMaxBytes())
	promptBytes := args.PromptBytes
	if promptBytes < len(args.PromptBody) {
		promptBytes = len(args.PromptBody)
	}
	responseSample := []byte(nil)
	responseBytes := 0
	responseTruncated := false
	if args.Result != nil {
		responseSample = truncateValidUTF8(args.Result.ResponseSample, c.responseMaxBytes())
		responseBytes = args.Result.ResponseBytes
		if responseBytes == 0 {
			responseBytes = len(args.Result.ResponseSample)
		}
		responseTruncated = args.Result.ResponseTruncated || len(args.Result.ResponseSample) > len(responseSample)
	}
	responseRedacted := truncateStringUTF8(redactGenerationResponse(responseSample), c.responseMaxBytes())

	row := &GenerationContent{
		RequestID:          args.RequestID,
		Model:              args.Model,
		RequestPayloadHash: args.RequestPayloadHash,
		PromptRedacted:     promptRedacted,
		ResponseRedacted:   responseRedacted,
		PromptBytes:        promptBytes,
		ResponseBytes:      responseBytes,
		ResponseTruncated:  responseTruncated,
		RedactionVersion:   generationRedactionVersion,
	}
	if args.APIKeyID > 0 {
		row.APIKeyID = &args.APIKeyID
	}
	if args.UserID > 0 {
		row.UserID = &args.UserID
	}
	if args.GroupID != nil {
		row.GroupID = args.GroupID
	}
	if args.AccountID > 0 {
		row.AccountID = &args.AccountID
	}
	if err := c.repo.Create(ctx, row); err != nil {
		logger.LegacyPrintf("service.gateway", "generation content create failed: %v (request_id=%s)", err, args.RequestID)
	}
}

func truncateValidUTF8(value []byte, max int) []byte {
	if max <= 0 || len(value) <= max {
		return value
	}
	value = value[:max]
	for len(value) > 0 && !utf8.Valid(value) {
		value = value[:len(value)-1]
	}
	return value
}

func truncateStringUTF8(value string, max int) string {
	return string(truncateValidUTF8([]byte(value), max))
}

func (s *GatewayService) SetGenerationContentCollector(collector *GenerationContentCollector) {
	if s != nil {
		s.generationCollector = collector
	}
}

func (s *GatewayService) contentCaptureEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.ContentCapture.Enabled
}

func (s *GatewayService) SnapshotGenerationPrompt(body []byte) GenerationPromptSnapshot {
	if !s.contentCaptureEnabled() {
		return GenerationPromptSnapshot{}
	}
	limit := boundedGenerationBytes(s.cfg.Gateway.ContentCapture.PromptMaxBytes, defaultGenerationPromptMaxBytes, maxGenerationPromptMaxBytes)
	bounded := truncateValidUTF8(body, limit)
	return GenerationPromptSnapshot{
		Body:          append([]byte(nil), bounded...),
		OriginalBytes: len(body),
		Truncated:     len(bounded) < len(body),
	}
}

func (s *GatewayService) CollectGenerationContent(ctx context.Context, args GenerationContentCaptureArgs) {
	if s == nil || s.generationCollector == nil || !s.contentCaptureEnabled() {
		return
	}
	s.generationCollector.Collect(ctx, args)
}
