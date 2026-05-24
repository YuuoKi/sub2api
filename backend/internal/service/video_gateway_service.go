package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	videoDefaultPageSize  = 20
	videoMaxPageSize      = 100
	videoDashboardLimit   = 8
	videoDefaultBatchSize = 20
)

const (
	videoKeyStatusNormal         = "normal"
	videoKeyStatusMissing        = "missing"
	videoKeyStatusDisabled       = "disabled"
	videoKeyStatusAuthFailed     = "auth_failed"
	videoKeyStatusRateLimited    = "rate_limited"
	videoKeyStatusQuotaExhausted = "quota_exhausted"

	videoHealthStatusHealthy          = "healthy"
	videoHealthStatusNeedsKey         = "needs_key"
	videoHealthStatusDisabled         = "disabled"
	videoHealthStatusAuthFailed       = "auth_failed"
	videoHealthStatusRateLimited      = "rate_limited"
	videoHealthStatusQuotaExhausted   = "quota_exhausted"
	videoHealthStatusTimeout          = "timeout"
	videoHealthStatusModerationFailed = "moderation_failed"
	videoHealthStatusModelUnavailable = "model_unavailable"
	videoHealthStatusUnknown          = "unknown"
)

type VideoGatewayService struct {
	repo      VideoGatewayRepository
	encryptor VideoKeyEncryptor
	adapters  map[string]VideoAdapter
	cfg       *config.Config
}

type videoRouteDecision struct {
	Account  *VideoProviderAccount
	Strategy string
	Reason   string
	Skipped  []videoRouteSkip
}

type videoRouteSkip struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Provider    string `json:"provider"`
	Reason      string `json:"reason"`
}

func NewVideoGatewayService(repo VideoGatewayRepository, encryptor VideoKeyEncryptor, cfg *config.Config) *VideoGatewayService {
	return &VideoGatewayService{
		repo:      repo,
		encryptor: encryptor,
		adapters:  NewVideoAdapterRegistry(),
		cfg:       cfg,
	}
}

func (s *VideoGatewayService) ListProviderAccounts(ctx context.Context) ([]*VideoProviderAccount, error) {
	items, err := s.repo.ListProviderAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list video providers: %w", err)
	}
	stats, err := s.providerRuntimeStats(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		s.decorateProviderForResponse(item, stats[item.ID])
	}
	return items, nil
}

func (s *VideoGatewayService) GetProviderAccount(ctx context.Context, id int64) (*VideoProviderAccount, error) {
	item, err := s.repo.GetProviderAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	stats, err := s.providerRuntimeStats(ctx)
	if err != nil {
		return nil, err
	}
	s.decorateProviderForResponse(item, stats[item.ID])
	return item, nil
}

func (s *VideoGatewayService) CreateProviderAccount(ctx context.Context, p VideoProviderCreateParams) (*VideoProviderAccount, error) {
	if err := validateVideoProvider(p.Provider); err != nil {
		return nil, err
	}
	displayName := strings.TrimSpace(p.DisplayName)
	if displayName == "" {
		displayName = defaultVideoProviderDisplayName(p.Provider)
	}
	account := &VideoProviderAccount{
		Provider:           strings.TrimSpace(p.Provider),
		DisplayName:        displayName,
		Enabled:            p.Enabled,
		BaseURL:            strings.TrimSpace(p.BaseURL),
		DefaultModel:       firstNonEmptyVideo(strings.TrimSpace(p.DefaultModel), defaultVideoModel(p.Provider)),
		RateLimitPerMinute: defaultVideoRateLimit(p.RateLimitPerMinute),
		Metadata:           p.Metadata,
	}
	if strings.TrimSpace(p.APIKey) != "" {
		if err := s.applyProviderAPIKey(account, p.APIKey); err != nil {
			return nil, err
		}
	}
	if err := s.repo.CreateProviderAccount(ctx, account); err != nil {
		return nil, fmt.Errorf("create video provider: %w", err)
	}
	s.decorateProviderForResponse(account, VideoProviderRuntimeStats{})
	return account, nil
}

func (s *VideoGatewayService) UpdateProviderAccount(ctx context.Context, id int64, p VideoProviderUpdateParams) (*VideoProviderAccount, error) {
	account, err := s.repo.GetProviderAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.DisplayName != nil {
		account.DisplayName = strings.TrimSpace(*p.DisplayName)
		if account.DisplayName == "" {
			account.DisplayName = defaultVideoProviderDisplayName(account.Provider)
		}
	}
	if p.Enabled != nil {
		account.Enabled = *p.Enabled
	}
	if p.BaseURL != nil {
		account.BaseURL = strings.TrimSpace(*p.BaseURL)
	}
	if p.DefaultModel != nil {
		account.DefaultModel = firstNonEmptyVideo(strings.TrimSpace(*p.DefaultModel), defaultVideoModel(account.Provider))
	}
	if p.RateLimitPerMinute != nil {
		account.RateLimitPerMinute = defaultVideoRateLimit(*p.RateLimitPerMinute)
	}
	if p.Metadata != nil {
		account.Metadata = *p.Metadata
	}
	if p.APIKey != nil && strings.TrimSpace(*p.APIKey) != "" {
		if err := s.applyProviderAPIKey(account, *p.APIKey); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateProviderAccount(ctx, account); err != nil {
		return nil, fmt.Errorf("update video provider: %w", err)
	}
	stats, err := s.providerRuntimeStats(ctx)
	if err != nil {
		return nil, err
	}
	s.decorateProviderForResponse(account, stats[account.ID])
	return account, nil
}

func (s *VideoGatewayService) TestProviderAccount(ctx context.Context, id int64) (*VideoProviderTestResult, error) {
	account, err := s.repo.GetProviderAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	s.decryptProviderKey(account)
	adapter, err := s.adapterFor(account.Provider)
	if err != nil {
		return nil, err
	}
	preview := adapter.BuildCreatePayload(account, &VideoTask{
		Model:       firstNonEmptyVideo(account.DefaultModel, defaultVideoModel(account.Provider)),
		TaskType:    VideoTaskTypeTextToVideo,
		Prompt:      "mock connection test",
		AspectRatio: "16:9",
		Duration:    5,
		Resolution:  "720p",
	})
	if account.Provider == VideoProviderMock {
		return &VideoProviderTestResult{
			Provider:         account.Provider,
			Configured:       true,
			Reachable:        account.Enabled,
			Message:          "mock provider is local and ready",
			NormalizedStatus: adapter.NormalizeStatus(VideoStatusSubmitted),
			PayloadPreview:   preview,
		}, nil
	}
	if !account.APIKeyConfigured || strings.TrimSpace(account.PlainAPIKey) == "" {
		return &VideoProviderTestResult{
			Provider:         account.Provider,
			Configured:       false,
			Reachable:        false,
			Message:          "api key is not configured; real upstream call skipped",
			NormalizedStatus: adapter.NormalizeStatus("processing"),
			PayloadPreview:   preview,
		}, nil
	}
	return &VideoProviderTestResult{
		Provider:         account.Provider,
		Configured:       true,
		Reachable:        false,
		Message:          "adapter skeleton is mapped; real network test is disabled in P0",
		NormalizedStatus: adapter.NormalizeStatus("processing"),
		PayloadPreview:   preview,
	}, nil
}

func (s *VideoGatewayService) CreateTask(ctx context.Context, p VideoTaskCreateParams) (*VideoTask, error) {
	if err := validateVideoTaskType(p.TaskType); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Prompt) == "" {
		return nil, ErrVideoMissingPrompt
	}
	route, err := s.resolveVideoRoute(ctx, p.ProviderAccountID)
	if err != nil {
		return nil, err
	}
	account := route.Account
	if _, err := s.adapterFor(account.Provider); err != nil {
		return nil, err
	}
	task := &VideoTask{
		ProviderAccountID:   account.ID,
		ProviderAccountName: account.DisplayName,
		Provider:            account.Provider,
		Model:               firstNonEmptyVideo(strings.TrimSpace(p.Model), account.DefaultModel, defaultVideoModel(account.Provider)),
		TaskType:            p.TaskType,
		Prompt:              strings.TrimSpace(p.Prompt),
		NegativePrompt:      strings.TrimSpace(p.NegativePrompt),
		ReferenceImageURL:   strings.TrimSpace(p.ReferenceImageURL),
		ReferenceVideoURL:   strings.TrimSpace(p.ReferenceVideoURL),
		AspectRatio:         firstNonEmptyVideo(strings.TrimSpace(p.AspectRatio), "16:9"),
		Duration:            defaultVideoDuration(p.Duration),
		Resolution:          firstNonEmptyVideo(strings.TrimSpace(p.Resolution), "720p"),
		Status:              VideoStatusQueued,
		CreatedBy:           p.CreatedBy,
	}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create video task: %w", err)
	}
	_ = s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
		VideoTaskID: task.ID,
		EventType:   "routed",
		Message:     "video task routed",
		Payload: map[string]any{
			"strategy":              route.Strategy,
			"reason":                route.Reason,
			"selected_account_id":   account.ID,
			"selected_account_name": account.DisplayName,
			"provider":              account.Provider,
			"skipped_accounts":      route.Skipped,
		},
	})
	_ = s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
		VideoTaskID: task.ID,
		EventType:   "queued",
		Message:     "video task queued",
		Payload: map[string]any{
			"provider":            task.Provider,
			"model":               task.Model,
			"provider_account_id": task.ProviderAccountID,
			"route_strategy":      route.Strategy,
		},
	})
	return task, nil
}

func (s *VideoGatewayService) ListTasks(ctx context.Context, p VideoTaskListParams) ([]*VideoTask, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = videoDefaultPageSize
	}
	if p.PageSize > videoMaxPageSize {
		p.PageSize = videoMaxPageSize
	}
	if p.Status != "" {
		if err := validateVideoStatus(p.Status); err != nil {
			return nil, 0, err
		}
	}
	if p.Provider != "" {
		if err := validateVideoProvider(p.Provider); err != nil {
			return nil, 0, err
		}
	}
	return s.repo.ListTasks(ctx, p)
}

func (s *VideoGatewayService) GetTask(ctx context.Context, id, userID int64, isAdmin bool) (*VideoTask, []*VideoTaskEvent, error) {
	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if !isAdmin && task.CreatedBy != userID {
		return nil, nil, ErrVideoTaskNotFound
	}
	events, err := s.repo.ListTaskEvents(ctx, id, 200)
	if err != nil {
		return nil, nil, fmt.Errorf("list video task events: %w", err)
	}
	return task, events, nil
}

func (s *VideoGatewayService) CancelTask(ctx context.Context, id, userID int64, isAdmin bool) (*VideoTask, error) {
	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isAdmin && task.CreatedBy != userID {
		return nil, ErrVideoTaskNotFound
	}
	if IsTerminalVideoStatus(task.Status) {
		return task, nil
	}
	account, err := s.repo.GetProviderAccount(ctx, task.ProviderAccountID)
	if err != nil {
		return nil, err
	}
	s.decryptProviderKey(account)
	adapter, err := s.adapterFor(task.Provider)
	if err != nil {
		return nil, err
	}
	result, err := adapter.CancelTask(ctx, account, task)
	if err != nil {
		return nil, err
	}
	task.Status = firstNonEmptyVideo(result.Status, VideoStatusCancelled)
	now := time.Now().UTC()
	task.CompletedAt = &now
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("cancel video task: %w", err)
	}
	_ = s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
		VideoTaskID: task.ID,
		EventType:   "cancelled",
		Message:     "video task cancelled",
		Payload:     result.Payload,
	})
	_ = s.repo.InsertUsageLog(ctx, task)
	return task, nil
}

func (s *VideoGatewayService) Dashboard(ctx context.Context) (*VideoDashboard, error) {
	since := time.Now().In(time.Local).Truncate(24 * time.Hour)
	statusCounts, err := s.repo.CountTasksSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("count video tasks: %w", err)
	}
	providers, err := s.repo.ListProviderAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list video providers: %w", err)
	}
	providerStats, err := s.repo.ProviderAccountTaskStatsSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("count video provider account tasks: %w", err)
	}
	failures, err := s.repo.ListRecentTasksByStatus(ctx, VideoStatusFailed, videoDashboardLimit)
	if err != nil {
		return nil, fmt.Errorf("list recent failed video tasks: %w", err)
	}
	successes, err := s.repo.ListRecentTasksByStatus(ctx, VideoStatusSucceeded, videoDashboardLimit)
	if err != nil {
		return nil, fmt.Errorf("list recent successful video tasks: %w", err)
	}
	usage, err := s.repo.UsageSummarySince(ctx, since.AddDate(0, 0, -7))
	if err != nil {
		return nil, fmt.Errorf("summarize video usage: %w", err)
	}
	total := sumStatusCounts(statusCounts)
	success := statusCounts[VideoStatusSucceeded]
	successRate := 0.0
	if total > 0 {
		successRate = float64(success) * 100 / float64(total)
	}
	providerStatus := make([]VideoProviderStatus, 0, len(providers))
	for _, p := range providers {
		s.decorateProviderForResponse(p, providerStats[p.ID])
		providerStatus = append(providerStatus, VideoProviderStatus{
			Provider:         p.Provider,
			DisplayName:      p.DisplayName,
			Enabled:          p.Enabled,
			APIKeyConfigured: p.APIKeyConfigured,
			MaskedKey:        p.MaskedKey,
			DefaultModel:     p.DefaultModel,
			KeyStatus:        p.KeyStatus,
			HealthStatus:     p.HealthStatus,
			DiagnosticType:   p.DiagnosticType,
			SuggestedAction:  p.SuggestedAction,
			RouteAvailable:   p.RouteAvailable,
			RouteSkipReason:  p.RouteSkipReason,
			Priority:         p.Priority,
			CurrentInflight:  p.CurrentInflight,
			LastError:        p.LastError,
			LastTestAt:       p.LastTestAt,
			UpdatedAt:        p.UpdatedAt,
			TodayTasks:       p.TodayTasks,
			RunningTasks:     p.CurrentInflight,
			FailedTasks:      p.TodayFailures,
		})
	}
	return &VideoDashboard{
		TodayTasks:        total,
		SuccessRate:       successRate,
		FailedTasks:       statusCounts[VideoStatusFailed],
		QueuedTasks:       statusCounts[VideoStatusQueued],
		RunningTasks:      statusCounts[VideoStatusRunning] + statusCounts[VideoStatusSubmitted],
		ProviderStatus:    providerStatus,
		HealthDiagnostics: buildVideoHealthDiagnostics(providers),
		RecentFailures:    failures,
		RecentSuccesses:   successes,
		UsageOverview:     usage,
	}, nil
}

func (s *VideoGatewayService) providerRuntimeStats(ctx context.Context) (map[int64]VideoProviderRuntimeStats, error) {
	since := time.Now().In(time.Local).Truncate(24 * time.Hour)
	stats, err := s.repo.ProviderAccountTaskStatsSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("count video provider account tasks: %w", err)
	}
	return stats, nil
}

func (s *VideoGatewayService) resolveVideoRoute(ctx context.Context, providerAccountID int64) (*videoRouteDecision, error) {
	stats, err := s.providerRuntimeStats(ctx)
	if err != nil {
		return nil, err
	}
	if providerAccountID > 0 {
		account, err := s.repo.GetProviderAccount(ctx, providerAccountID)
		if err != nil {
			return nil, err
		}
		s.decorateProviderForResponse(account, stats[account.ID])
		if !account.RouteAvailable {
			return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
				"provider_account_id": fmt.Sprintf("%d", account.ID),
				"reason":              account.RouteSkipReason,
			})
		}
		return &videoRouteDecision{
			Account:  account,
			Strategy: "explicit",
			Reason:   "指定账号可用",
		}, nil
	}

	accounts, err := s.repo.ListProviderAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list video providers: %w", err)
	}
	for _, account := range accounts {
		s.decorateProviderForResponse(account, stats[account.ID])
	}
	route := selectLeastInflightVideoRoute(accounts)
	if route == nil {
		return nil, ErrVideoProviderDisabled.WithMetadata(map[string]string{
			"reason": "没有可用的视频 API 账号",
		})
	}
	return route, nil
}

func selectLeastInflightVideoRoute(accounts []*VideoProviderAccount) *videoRouteDecision {
	candidates := make([]*VideoProviderAccount, 0, len(accounts))
	skipped := make([]videoRouteSkip, 0)
	for _, account := range accounts {
		if account == nil {
			continue
		}
		if !account.RouteAvailable {
			reason := strings.TrimSpace(account.RouteSkipReason)
			if reason == "" {
				reason = "账号不可调度"
			}
			skipped = append(skipped, videoRouteSkip{
				ID:          account.ID,
				DisplayName: account.DisplayName,
				Provider:    account.Provider,
				Reason:      reason,
			})
			continue
		}
		candidates = append(candidates, account)
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.CurrentInflight != right.CurrentInflight {
			return left.CurrentInflight < right.CurrentInflight
		}
		if left.TodayFailures != right.TodayFailures {
			return left.TodayFailures < right.TodayFailures
		}
		if left.Priority != right.Priority {
			return left.Priority < right.Priority
		}
		return left.ID < right.ID
	})
	selected := candidates[0]
	return &videoRouteDecision{
		Account:  selected,
		Strategy: VideoRouteStrategyLeastInflight,
		Reason: fmt.Sprintf(
			"选择当前处理中 %d、今日失败 %d、优先级 %d 的可用账号",
			selected.CurrentInflight,
			selected.TodayFailures,
			selected.Priority,
		),
		Skipped: skipped,
	}
}

func (s *VideoGatewayService) decorateProviderForResponse(account *VideoProviderAccount, stats VideoProviderRuntimeStats) {
	s.prepareProviderForResponse(account)
	if account == nil {
		return
	}
	account.TodayTasks = stats.TodayTasks
	account.CurrentInflight = stats.CurrentInflight
	account.TodayFailures = stats.TodayFailures
	account.LastError = firstNonEmptyVideo(videoMetadataString(account.Metadata, "last_error"), stats.LastError)
	if stats.LastErrorAt != nil {
		lastErrorAt := *stats.LastErrorAt
		account.LastTestAt = &lastErrorAt
	}
	if lastTestAt := videoMetadataTime(account.Metadata, "last_test_at"); lastTestAt != nil {
		account.LastTestAt = lastTestAt
	}
	account.Priority = videoMetadataInt(account.Metadata, "priority", 50)
	account.KeyStatus = deriveVideoKeyStatus(account)
	account.HealthStatus = deriveVideoHealthStatus(account)
	account.DiagnosticType = firstNonEmptyVideo(
		videoMetadataString(account.Metadata, "diagnostic_type"),
		videoProviderDiagnosticType(account.KeyStatus, account.HealthStatus),
	)
	account.SuggestedAction = firstNonEmptyVideo(
		videoMetadataString(account.Metadata, "suggested_action"),
		videoProviderSuggestedAction(account.DiagnosticType, account.KeyStatus, account.HealthStatus),
	)
	account.RouteAvailable, account.RouteSkipReason = videoProviderRouteAvailability(account)
	if account.Provider == VideoProviderMock && account.KeyStatus == videoKeyStatusNormal {
		account.APIKeyConfigured = true
	}
}

func deriveVideoKeyStatus(account *VideoProviderAccount) string {
	if account == nil {
		return videoKeyStatusMissing
	}
	if !account.Enabled {
		return firstNonEmptyVideo(videoMetadataString(account.Metadata, "key_status"), videoKeyStatusDisabled)
	}
	status := firstNonEmptyVideo(videoMetadataString(account.Metadata, "key_status"))
	if status != "" {
		return status
	}
	if account.Provider == VideoProviderMock || account.APIKeyConfigured {
		return videoKeyStatusNormal
	}
	return videoKeyStatusMissing
}

func deriveVideoHealthStatus(account *VideoProviderAccount) string {
	if account == nil {
		return videoHealthStatusUnknown
	}
	if !account.Enabled {
		return firstNonEmptyVideo(videoMetadataString(account.Metadata, "health_status"), videoHealthStatusDisabled)
	}
	status := firstNonEmptyVideo(videoMetadataString(account.Metadata, "health_status"))
	if status != "" {
		return status
	}
	switch account.KeyStatus {
	case videoKeyStatusNormal:
		return videoHealthStatusHealthy
	case videoKeyStatusMissing:
		return videoHealthStatusNeedsKey
	case videoKeyStatusAuthFailed:
		return videoHealthStatusAuthFailed
	case videoKeyStatusRateLimited:
		return videoHealthStatusRateLimited
	case videoKeyStatusQuotaExhausted:
		return videoHealthStatusQuotaExhausted
	default:
		return videoHealthStatusUnknown
	}
}

func videoProviderRouteAvailability(account *VideoProviderAccount) (bool, string) {
	if account == nil {
		return false, "账号不存在"
	}
	if !account.Enabled {
		return false, "账号已停用"
	}
	if account.APIKeyDecryptFailed {
		return false, "Key 解密失败"
	}
	switch account.KeyStatus {
	case videoKeyStatusMissing:
		return false, "Key 未配置"
	case videoKeyStatusDisabled:
		return false, "账号已停用"
	case videoKeyStatusAuthFailed:
		return false, "鉴权失败"
	case videoKeyStatusRateLimited:
		return false, "触发限流"
	case videoKeyStatusQuotaExhausted:
		return false, "额度不足"
	}
	switch account.HealthStatus {
	case videoHealthStatusHealthy:
		return true, ""
	case videoHealthStatusNeedsKey:
		return false, "Key 未配置"
	case videoHealthStatusDisabled:
		return false, "账号已停用"
	case videoHealthStatusAuthFailed:
		return false, "鉴权失败"
	case videoHealthStatusRateLimited:
		return false, "触发限流"
	case videoHealthStatusQuotaExhausted:
		return false, "额度不足"
	case videoHealthStatusTimeout:
		return false, "上游超时"
	case videoHealthStatusModerationFailed:
		return false, "内容审核失败"
	case videoHealthStatusModelUnavailable:
		return false, "模型不可用"
	case videoHealthStatusUnknown:
		return false, "未知异常"
	default:
		return true, ""
	}
}

func videoProviderDiagnosticType(keyStatus, healthStatus string) string {
	switch keyStatus {
	case videoKeyStatusMissing:
		return "Key 未配置"
	case videoKeyStatusAuthFailed:
		return "鉴权失败"
	case videoKeyStatusQuotaExhausted:
		return "额度不足"
	case videoKeyStatusRateLimited:
		return "触发限流"
	}
	switch healthStatus {
	case videoHealthStatusNeedsKey:
		return "Key 未配置"
	case videoHealthStatusAuthFailed:
		return "鉴权失败"
	case videoHealthStatusQuotaExhausted:
		return "额度不足"
	case videoHealthStatusRateLimited:
		return "触发限流"
	case videoHealthStatusTimeout:
		return "上游超时"
	case videoHealthStatusModerationFailed:
		return "内容审核失败"
	case videoHealthStatusModelUnavailable:
		return "模型不可用"
	case videoHealthStatusUnknown:
		return "未知异常"
	}
	return ""
}

func videoProviderSuggestedAction(diagnosticType, keyStatus, healthStatus string) string {
	switch firstNonEmptyVideo(diagnosticType, videoProviderDiagnosticType(keyStatus, healthStatus)) {
	case "Key 未配置":
		return "请配置 API Key 后启用真实调用"
	case "鉴权失败":
		return "请检查 Key 是否过期或填错"
	case "额度不足":
		return "请确认上游账号余额或额度"
	case "触发限流":
		return "请降低并发或增加账号"
	case "上游超时":
		return "稍后重试或切换通道"
	case "内容审核失败":
		return "调整提示词或素材后重试"
	case "模型不可用":
		return "切换模型或确认上游模型可用"
	case "未知异常":
		return "查看任务详情和事件时间线"
	default:
		return ""
	}
}

func buildVideoHealthDiagnostics(providers []*VideoProviderAccount) []VideoHealthDiagnostic {
	out := make([]VideoHealthDiagnostic, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		exceptionType := provider.DiagnosticType
		if strings.TrimSpace(exceptionType) == "" {
			exceptionType = "正常"
		}
		status := "正常"
		if !provider.RouteAvailable {
			status = "需处理"
		}
		out = append(out, VideoHealthDiagnostic{
			Provider:        provider.Provider,
			DisplayName:     defaultVideoProviderDisplayName(provider.Provider),
			RouteAccount:    provider.DisplayName,
			KeyStatus:       provider.KeyStatus,
			LastTestAt:      provider.LastTestAt,
			ExceptionType:   exceptionType,
			ImpactTasks:     provider.TodayFailures,
			RecentError:     provider.LastError,
			SuggestedAction: provider.SuggestedAction,
			Status:          status,
		})
	}
	return out
}

func (s *VideoGatewayService) applyProviderAPIKey(account *VideoProviderAccount, plain string) error {
	plain = strings.TrimSpace(plain)
	encrypted, err := s.encryptor.Encrypt(plain)
	if err != nil {
		return fmt.Errorf("encrypt video provider key: %w", err)
	}
	account.EncryptedAPIKey = encrypted
	account.MaskedKey = MaskVideoAPIKey(plain)
	return nil
}

func (s *VideoGatewayService) prepareProviderForResponse(account *VideoProviderAccount) {
	if account == nil {
		return
	}
	account.APIKeyConfigured = strings.TrimSpace(account.EncryptedAPIKey) != ""
	account.PlainAPIKey = ""
}

func videoMetadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func videoMetadataInt(metadata map[string]any, key string, fallback int) int {
	if metadata == nil {
		return fallback
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return int(parsed)
		}
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func videoMetadataTime(metadata map[string]any, key string) *time.Time {
	raw := videoMetadataString(metadata, key)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return &parsed
		}
	}
	return nil
}

func (s *VideoGatewayService) decryptProviderKey(account *VideoProviderAccount) {
	if account == nil || strings.TrimSpace(account.EncryptedAPIKey) == "" {
		if account != nil {
			account.APIKeyConfigured = false
		}
		return
	}
	account.APIKeyConfigured = true
	plain, err := s.encryptor.Decrypt(account.EncryptedAPIKey)
	if err != nil {
		slog.Warn("video_gateway: decrypt provider key failed", "provider_id", account.ID, "provider", account.Provider, "error", err)
		account.APIKeyDecryptFailed = true
		account.PlainAPIKey = ""
		return
	}
	account.PlainAPIKey = plain
}

func (s *VideoGatewayService) adapterFor(provider string) (VideoAdapter, error) {
	adapter, ok := s.adapters[provider]
	if !ok {
		return nil, ErrVideoInvalidProvider
	}
	return adapter, nil
}

func MaskVideoAPIKey(plain string) string {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return ""
	}
	if len(plain) <= 8 {
		return "***"
	}
	return plain[:4] + "***" + plain[len(plain)-4:]
}

func validateVideoProvider(provider string) error {
	switch strings.TrimSpace(provider) {
	case VideoProviderMock, VideoProviderSeedance, VideoProviderKling:
		return nil
	default:
		return ErrVideoInvalidProvider
	}
}

func validateVideoTaskType(taskType string) error {
	switch strings.TrimSpace(taskType) {
	case VideoTaskTypeTextToVideo, VideoTaskTypeImageToVideo, VideoTaskTypeReferenceToVideo:
		return nil
	default:
		return ErrVideoInvalidTaskType
	}
}

func validateVideoStatus(status string) error {
	switch strings.TrimSpace(status) {
	case VideoStatusQueued, VideoStatusSubmitted, VideoStatusRunning, VideoStatusSucceeded, VideoStatusFailed, VideoStatusCancelled:
		return nil
	default:
		return ErrVideoInvalidStatus
	}
}

func defaultVideoProviderDisplayName(provider string) string {
	switch provider {
	case VideoProviderMock:
		return "Mock Provider"
	case VideoProviderSeedance:
		return "Seedance 2.0"
	case VideoProviderKling:
		return "Kling"
	default:
		return provider
	}
}

func defaultVideoModel(provider string) string {
	switch provider {
	case VideoProviderMock:
		return "mock-video-v1"
	case VideoProviderSeedance:
		return "seedance-2-0-pro"
	case VideoProviderKling:
		return "kling-v1"
	default:
		return "video-model"
	}
}

func defaultVideoRateLimit(v int) int {
	if v <= 0 {
		return 60
	}
	return v
}

func defaultVideoDuration(v int) int {
	if v <= 0 {
		return 5
	}
	return v
}

func firstNonEmptyVideo(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func sumStatusCounts(counts map[string]int64) int64 {
	var total int64
	for _, v := range counts {
		total += v
	}
	return total
}
