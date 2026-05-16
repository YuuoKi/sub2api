package service

import (
	"context"
	"fmt"
	"log/slog"
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

type VideoGatewayService struct {
	repo      VideoGatewayRepository
	encryptor VideoKeyEncryptor
	adapters  map[string]VideoAdapter
	cfg       *config.Config
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
	for _, item := range items {
		s.prepareProviderForResponse(item)
	}
	return items, nil
}

func (s *VideoGatewayService) GetProviderAccount(ctx context.Context, id int64) (*VideoProviderAccount, error) {
	item, err := s.repo.GetProviderAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	s.prepareProviderForResponse(item)
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
	s.prepareProviderForResponse(account)
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
	s.prepareProviderForResponse(account)
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
	if p.ProviderAccountID <= 0 {
		return nil, ErrVideoMissingProvider
	}
	if err := validateVideoTaskType(p.TaskType); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Prompt) == "" {
		return nil, ErrVideoMissingPrompt
	}
	account, err := s.repo.GetProviderAccount(ctx, p.ProviderAccountID)
	if err != nil {
		return nil, err
	}
	if !account.Enabled {
		return nil, ErrVideoProviderDisabled
	}
	if _, err := s.adapterFor(account.Provider); err != nil {
		return nil, err
	}
	task := &VideoTask{
		ProviderAccountID: account.ID,
		Provider:          account.Provider,
		Model:             firstNonEmptyVideo(strings.TrimSpace(p.Model), account.DefaultModel, defaultVideoModel(account.Provider)),
		TaskType:          p.TaskType,
		Prompt:            strings.TrimSpace(p.Prompt),
		NegativePrompt:    strings.TrimSpace(p.NegativePrompt),
		ReferenceImageURL: strings.TrimSpace(p.ReferenceImageURL),
		ReferenceVideoURL: strings.TrimSpace(p.ReferenceVideoURL),
		AspectRatio:       firstNonEmptyVideo(strings.TrimSpace(p.AspectRatio), "16:9"),
		Duration:          defaultVideoDuration(p.Duration),
		Resolution:        firstNonEmptyVideo(strings.TrimSpace(p.Resolution), "720p"),
		Status:            VideoStatusQueued,
		CreatedBy:         p.CreatedBy,
	}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create video task: %w", err)
	}
	_ = s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
		VideoTaskID: task.ID,
		EventType:   "queued",
		Message:     "video task queued",
		Payload: map[string]any{
			"provider": task.Provider,
			"model":    task.Model,
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
	providerCounts, err := s.repo.CountProviderTasksSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("count video provider tasks: %w", err)
	}
	providers, err := s.repo.ListProviderAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list video providers: %w", err)
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
		s.prepareProviderForResponse(p)
		counts := providerCounts[p.Provider]
		providerStatus = append(providerStatus, VideoProviderStatus{
			Provider:         p.Provider,
			DisplayName:      p.DisplayName,
			Enabled:          p.Enabled,
			APIKeyConfigured: p.APIKeyConfigured,
			MaskedKey:        p.MaskedKey,
			DefaultModel:     p.DefaultModel,
			UpdatedAt:        p.UpdatedAt,
			TodayTasks:       sumStatusCounts(counts),
			RunningTasks:     counts[VideoStatusRunning],
			FailedTasks:      counts[VideoStatusFailed],
		})
	}
	return &VideoDashboard{
		TodayTasks:      total,
		SuccessRate:     successRate,
		FailedTasks:     statusCounts[VideoStatusFailed],
		QueuedTasks:     statusCounts[VideoStatusQueued],
		RunningTasks:    statusCounts[VideoStatusRunning] + statusCounts[VideoStatusSubmitted],
		ProviderStatus:  providerStatus,
		RecentFailures:  failures,
		RecentSuccesses: successes,
		UsageOverview:   usage,
	}, nil
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
	account.APIKeyConfigured = strings.TrimSpace(account.EncryptedAPIKey) != "" || strings.TrimSpace(account.MaskedKey) != ""
	account.PlainAPIKey = ""
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
