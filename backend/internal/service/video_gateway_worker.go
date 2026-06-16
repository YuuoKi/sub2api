package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	// videoProviderCallTimeout mirrors the adapter's http.Client timeout: a single
	// provider create/poll round-trip can take this long.
	videoProviderCallTimeout = 30 * time.Second
	// videoTaskProcessBudget is the per-task processing budget (one provider call
	// plus the follow-up DB writes). It is derived from a cancel-detached root so a
	// poll that outlives the worker's tick cadence still has time to PERSIST its
	// result; without it a slow poll's UpdateTask runs on an already-expired tick
	// context and the result (even a 'succeeded') is silently dropped. (VA2.)
	videoTaskProcessBudget = videoProviderCallTimeout + 10*time.Second
	// videoDefaultMaxPollAttempts is the per-task poll cap when config is absent.
	// 72 × 5s poll interval = 360s window ≥ 2× the ~170s Seedance generation time.
	videoDefaultMaxPollAttempts = 72
	// videoDefaultPollInterval is the worker tick / poll cadence when config is absent.
	videoDefaultPollInterval = 5 * time.Second
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
	interval := videoDefaultPollInterval
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
	// A long-lived cancelable context tied to stopCh. Ticks intentionally do NOT
	// impose the poll interval as a deadline: a single provider poll can take up to
	// videoProviderCallTimeout (~30s), far longer than the 5s tick cadence, and its
	// result must still be persisted. Per-task budgets are enforced in processTask;
	// shutdown cancels any in-flight batch via this context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-w.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := w.ProcessOnce(ctx); err != nil {
				slog.Warn("video_gateway: worker tick failed", "error", err)
			}
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

// maxPollAttempts is the per-task VA2 poll cap, sourced from config with a safe
// default when config is absent (e.g. unit tests passing a nil cfg).
func (s *VideoGatewayService) maxPollAttempts() int {
	if s.cfg != nil && s.cfg.VideoGateway.MaxPollAttempts > 0 {
		return s.cfg.VideoGateway.MaxPollAttempts
	}
	return videoDefaultMaxPollAttempts
}

func (s *VideoGatewayService) processTask(ctx context.Context, task *VideoTask, timeout time.Duration) error {
	if task == nil || IsTerminalVideoStatus(task.Status) {
		return nil
	}
	// Honor shutdown / cancellation between tasks before starting provider work.
	if err := ctx.Err(); err != nil {
		return err
	}
	// Give each task a bounded budget for one provider round-trip plus the follow-up
	// DB writes. The loop no longer imposes the short poll interval as a deadline, so
	// this budget — deliberately larger than the provider call timeout — is what lets
	// a poll slower than the tick cadence still persist its result (VA2). Cancellation
	// still propagates from the parent, so shutdown promptly aborts in-flight work.
	taskCtx, cancel := context.WithTimeout(ctx, videoTaskProcessBudget)
	defer cancel()

	if timeout > 0 && time.Since(task.CreatedAt) > timeout {
		return s.terminateVideoTask(taskCtx, task, "video task timed out", map[string]any{
			"timeout_minutes": int(timeout.Minutes()),
		})
	}
	account, err := s.repo.GetProviderAccount(taskCtx, task.ProviderAccountID)
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
		if err := s.submitTask(taskCtx, adapter, account, task); err != nil {
			return s.failTask(taskCtx, task, "video provider submit failed: "+err.Error(), map[string]any{"stage": "submit"})
		}
	case VideoStatusSubmitted, VideoStatusRunning:
		// VA2 poll cap: once a task has used its poll budget without reaching a
		// terminal status, fail it deterministically instead of polling forever.
		maxPolls := s.maxPollAttempts()
		if task.PollCount >= maxPolls {
			return s.terminateVideoTask(taskCtx, task, fmt.Sprintf("video task exceeded max poll attempts (%d)", maxPolls), map[string]any{
				"max_poll_attempts": maxPolls,
				"poll_count":        task.PollCount,
			})
		}
		if err := s.pollTask(taskCtx, adapter, account, task); err != nil {
			return s.failTask(taskCtx, task, "video provider poll failed: "+err.Error(), map[string]any{"stage": "poll"})
		}
	default:
		return nil
	}
	return nil
}

// terminateVideoTask closes out a non-terminal task as failed with a reason
// (wall-clock timeout or VA2 poll-cap exhaustion), recording the terminal event
// and a usage log so it is always reconciled exactly once.
func (s *VideoGatewayService) terminateVideoTask(ctx context.Context, task *VideoTask, message string, payload map[string]any) error {
	task.Status = VideoStatusFailed
	task.ErrorMessage = message
	now := time.Now().UTC()
	task.CompletedAt = &now
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return err
	}
	_ = s.repo.AddTaskEvent(ctx, &VideoTaskEvent{
		VideoTaskID: task.ID,
		EventType:   "failed",
		Message:     message,
		Payload:     payload,
	})
	_ = s.repo.InsertUsageLog(ctx, task)
	return nil
}

func (s *VideoGatewayService) submitTask(ctx context.Context, adapter VideoAdapter, account *VideoProviderAccount, task *VideoTask) error {
	result, err := adapter.CreateTask(ctx, account, task)
	if err != nil {
		return err
	}
	status := firstNonEmptyVideo(result.Status, VideoStatusSubmitted)
	// A SUCCESSFUL create means the task now lives upstream and must NEVER be re-created:
	// the only next action is to POLL it. Some providers (e.g. Ark — exact create-response
	// status token UNVERIFIED) return a create status that normalizes back to "queued"; if
	// we persisted that, processTask's `case VideoStatusQueued` would re-enter submitTask on
	// the next tick and issue a SECOND billed create. Advance any post-create "queued" to
	// "submitted" so the next tick polls instead. (No double-submit regardless of the token.)
	if status == VideoStatusQueued {
		status = VideoStatusSubmitted
	}
	task.Status = status
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
	// VA2: count every completed poll so processTask can enforce the per-task cap.
	task.PollCount++
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
		// VA1 billing: deduct only on a delivered (succeeded) generation.
		if status == VideoStatusSucceeded {
			s.chargeForVideo(ctx, task)
		}
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
