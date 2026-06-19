package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 采集内容容量兜底（当配置为 0 时使用）。response 容量由 M1-B.2 的限容 tee 在采集前强制；
// 这里的 prompt 容量在采集时强制（防止超长 prompt 撑大内容表）。
const (
	defaultGenerationResponseMaxBytes = 64 * 1024
	defaultGenerationPromptMaxBytes   = 256 * 1024
)

// GenerationContent 是一次 AI 生产被采集的（脱敏后）内容，落 ai_generation_content 表，
// 与 usage_logs 以 (APIKeyID, RequestID) 1:1 关联。
type GenerationContent struct {
	ID                 int64
	RequestID          string
	APIKeyID           *int64
	UserID             *int64
	GroupID            *int64
	AccountID          *int64
	Model              string
	RequestPayloadHash string
	PromptRedacted     string
	ResponseRedacted   string
	PromptBytes        int
	ResponseBytes      int
	ResponseTruncated  bool
	RedactionVersion   int
	CreatedAt          time.Time
}

// GenerationContentRepository 写入采集内容（append-only 旁路表）。
type GenerationContentRepository interface {
	Create(ctx context.Context, content *GenerationContent) error
}

// GenerationContentCaptureArgs 是 gateway handler 在一次中继完成后交给采集器的原料。
type GenerationContentCaptureArgs struct {
	RequestID          string
	UserID             int64
	APIKeyID           int64
	GroupID            *int64
	AccountID          int64
	Model              string
	RequestPayloadHash string
	PromptBody         []byte
	Result             *ForwardResult
}

// GenerationContentCollector 负责把 prompt+response 脱敏后写入内容表。
// 全程 fail-open：采集任何失败都不影响主调用/计费。
type GenerationContentCollector struct {
	repo GenerationContentRepository
	cfg  *config.Config
}

func NewGenerationContentCollector(repo GenerationContentRepository, cfg *config.Config) *GenerationContentCollector {
	return &GenerationContentCollector{repo: repo, cfg: cfg}
}

func (c *GenerationContentCollector) promptMaxBytes() int {
	if c != nil && c.cfg != nil && c.cfg.Gateway.ContentCapture.PromptMaxBytes > 0 {
		return c.cfg.Gateway.ContentCapture.PromptMaxBytes
	}
	return defaultGenerationPromptMaxBytes
}

// Collect 脱敏并写入一条采集内容。fail-open：recover panic、吞掉仓库错误（仅告警），绝不向上抛。
func (c *GenerationContentCollector) Collect(ctx context.Context, args GenerationContentCaptureArgs) {
	if c == nil || c.repo == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logger.LegacyPrintf("service.gateway", "generation content collect panic: %v (request_id=%s)", r, args.RequestID)
		}
	}()

	prompt := args.PromptBody
	if max := c.promptMaxBytes(); max > 0 && len(prompt) > max {
		prompt = prompt[:max]
	}

	var respSample []byte
	var respBytes int
	var respTruncated bool
	if args.Result != nil {
		respSample = args.Result.ResponseSample
		respBytes = args.Result.ResponseBytes
		respTruncated = args.Result.ResponseTruncated
	}

	row := &GenerationContent{
		RequestID:          args.RequestID,
		Model:              args.Model,
		RequestPayloadHash: args.RequestPayloadHash,
		PromptRedacted:     redactGenerationPrompt(prompt),
		ResponseRedacted:   redactGenerationResponse(respSample),
		PromptBytes:        len(args.PromptBody),
		ResponseBytes:      respBytes,
		ResponseTruncated:  respTruncated,
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

// SetGenerationContentCollector 注入采集器（可选依赖）。传 nil = 关闭采集（默认即关闭）。
// 仿 SetBudgetGuard 的可选依赖模式：未注入时所有采集调用为 no-op。
func (s *GatewayService) SetGenerationContentCollector(collector *GenerationContentCollector) {
	s.generationCollector = collector
}

// contentCaptureEnabled 仅当配置开关打开时为 true（默认 false → 热路径零开销、内容不采集）。
func (s *GatewayService) contentCaptureEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.ContentCapture.Enabled
}

// CollectGenerationContent 是 handler 在 RecordUsage 之后并列调用的采集入口（与计费隔离）。
// 仅在采集器已注入且开关打开时执行；采集器内部 fail-open。
func (s *GatewayService) CollectGenerationContent(ctx context.Context, args GenerationContentCaptureArgs) {
	if s == nil || s.generationCollector == nil || !s.contentCaptureEnabled() {
		return
	}
	s.generationCollector.Collect(ctx, args)
}
