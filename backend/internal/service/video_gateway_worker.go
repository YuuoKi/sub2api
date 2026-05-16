package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type VideoGatewayWorker struct {
	service  *VideoGatewayService
	interval time.Duration
	timeout  time.Duration
	batch    int
	stopCh   chan struct{}
	doneCh   chan struct{}
	once     sync.Once
}

func NewVideoGatewayWorker(service *VideoGatewayService, cfg *config.Config) *VideoGatewayWorker {
	interval := 2 * time.Second
	timeout := 15 * time.Minute
	batch := videoDefaultBatchSize
	if cfg != nil {
		if cfg.VideoGateway.PollIntervalSeconds > 0 {
			interval = time.Duration(cfg.VideoGateway.PollIntervalSeconds) * time.Second
		}
		if cfg.VideoGateway.TaskTimeoutMinutes > 0 {
			timeout = time.Duration(cfg.VideoGateway.TaskTimeoutMinutes) * time.Minute
		}
		if cfg.VideoGateway.WorkerBatchSize > 0 {
			batch = cfg.VideoGateway.WorkerBatchSize
		}
	}
	return &VideoGatewayWorker{
		service:  service,
		interval: interval,
		timeout:  timeout,
		batch:    batch,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func ProvideVideoGatewayWorker(service *VideoGatewayService, cfg *config.Config) *VideoGatewayWorker {
	worker := NewVideoGatewayWorker(service, cfg)
	if cfg == nil || cfg.VideoGateway.WorkerEnabled {
		worker.Start()
	}
	return worker
}

func (w *VideoGatewayWorker) Start() {
	if w == nil || w.service == nil {
		return
	}
	go w.loop()
}

func (w *VideoGatewayWorker) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		close(w.stopCh)
		<-w.doneCh
	})
}

func (w *VideoGatewayWorker) loop() {
	defer close(w.doneCh)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), w.interval)
			if err := w.ProcessOnce(ctx); err != nil {
				slog.Warn("video_gateway: worker tick failed", "error", err)
			}
			cancel()
		case <-w.stopCh:
			return
		}
	}
}

func (w *VideoGatewayWorker) ProcessOnce(ctx context.Context) error {
	if w == nil || w.service == nil {
		return nil
	}
	return w.service.ProcessRunnableTasks(ctx, w.batch, w.timeout)
}

func (s *VideoGatewayService) ProcessRunnableTasks(ctx context.Context, batch int, timeout time.Duration) error {
	if batch <= 0 {
		batch = videoDefaultBatchSize
	}
	tasks, err := s.repo.ListRunnableTasks(ctx, batch)
	if err != nil {
		return fmt.Errorf("list runnable video tasks: %w", err)
	}
	for _, task := range tasks {
		if err := s.processTask(ctx, task, timeout); err != nil {
			slog.Warn("video_gateway: process task failed", "task_id", task.ID, "provider", task.Provider, "error", err)
		}
	}
	return nil
}

func (s *VideoGatewayService) processTask(ctx context.Context, task *VideoTask, timeout time.Duration) error {
	if task == nil || IsTerminalVideoStatus(task.Status) {
		return nil
	}
	if timeout > 0 && time.Since(task.CreatedAt) > timeout {
		task.Status = VideoStatusFailed
		task.ErrorMessage = "video task timed out"
		now := time.Now().UTC()
		task.CompletedAt = &now
		if err := s.repo.UpdateTask(ctx, task); err != nil {
			return err
		}
		_ = s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
			VideoTaskID: task.ID,
			EventType:   "failed",
			Message:     "video task timed out",
			Payload: map[string]any{
				"timeout_minutes": int(timeout.Minutes()),
			},
		})
		_ = s.repo.InsertUsageLog(ctx, task)
		return nil
	}
	account, err := s.repo.GetProviderAccount(ctx, task.ProviderAccountID)
	if err != nil {
		return err
	}
	s.decryptProviderKey(account)
	if account.APIKeyDecryptFailed {
		return ErrVideoKeyDecryptFailed
	}
	adapter, err := s.adapterFor(task.Provider)
	if err != nil {
		return err
	}
	switch task.Status {
	case VideoStatusQueued:
		if err := s.submitTask(ctx, adapter, account, task); err != nil {
			return s.failTask(ctx, task, "video provider submit failed: "+err.Error(), map[string]any{"stage": "submit"})
		}
	case VideoStatusSubmitted, VideoStatusRunning:
		if err := s.pollTask(ctx, adapter, account, task); err != nil {
			return s.failTask(ctx, task, "video provider poll failed: "+err.Error(), map[string]any{"stage": "poll"})
		}
	default:
		return nil
	}
	return nil
}

func (s *VideoGatewayService) submitTask(ctx context.Context, adapter VideoAdapter, account *VideoProviderAccount, task *VideoTask) error {
	result, err := adapter.CreateTask(ctx, account, task)
	if err != nil {
		return err
	}
	task.Status = firstNonEmptyVideo(result.Status, VideoStatusSubmitted)
	task.UpstreamTaskID = firstNonEmptyVideo(result.UpstreamTaskID, task.UpstreamTaskID)
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return err
	}
	return s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
		VideoTaskID: task.ID,
		EventType:   VideoStatusSubmitted,
		Message:     "video task submitted to provider",
		Payload:     result.Payload,
	})
}

func (s *VideoGatewayService) pollTask(ctx context.Context, adapter VideoAdapter, account *VideoProviderAccount, task *VideoTask) error {
	result, err := adapter.PollTask(ctx, account, task)
	if err != nil {
		return err
	}
	status := adapter.NormalizeStatus(result.Status)
	task.Status = status
	if result.ResultURL != "" {
		task.ResultURL = result.ResultURL
	}
	if result.ErrorMessage != "" {
		task.ErrorMessage = result.ErrorMessage
	}
	if result.CostEstimate > 0 {
		task.CostEstimate = result.CostEstimate
	}
	if IsTerminalVideoStatus(status) {
		now := time.Now().UTC()
		task.CompletedAt = &now
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return err
	}
	eventType := status
	message := "video task status updated"
	if status == VideoStatusSucceeded {
		message = "video task succeeded"
	} else if status == VideoStatusFailed {
		message = "video task failed"
	}
	if err := s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
		VideoTaskID: task.ID,
		EventType:   eventType,
		Message:     message,
		Payload:     result.Payload,
	}); err != nil {
		return err
	}
	if IsTerminalVideoStatus(status) {
		_ = s.repo.InsertUsageLog(ctx, task)
	}
	return nil
}

func (s *VideoGatewayService) failTask(ctx context.Context, task *VideoTask, message string, payload map[string]any) error {
	task.Status = VideoStatusFailed
	task.ErrorMessage = message
	now := time.Now().UTC()
	task.CompletedAt = &now
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return err
	}
	if err := s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
		VideoTaskID: task.ID,
		EventType:   VideoStatusFailed,
		Message:     message,
		Payload:     payload,
	}); err != nil {
		return err
	}
	_ = s.repo.InsertUsageLog(ctx, task)
	return nil
}
