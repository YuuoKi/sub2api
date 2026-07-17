package service

import (
	"context"
	"errors"
	"time"
)

type videoSimulationWorkerStore interface {
	ClaimMockRunnableTasks(ctx context.Context, limit int, lease time.Duration) ([]*VideoTask, error)
	TransitionSimulationTask(ctx context.Context, taskID, expectedVersion int64, fromStatus, toStatus string) (*VideoTask, error)
	FinalizeSimulationTask(ctx context.Context, taskID, expectedVersion int64, status, errorMessage string) (VideoTaskFinalizationResult, error)
}

// VideoSimulationFailureStrategy is a test-only hook; production workers leave it nil.
type VideoSimulationFailureStrategy interface {
	ShouldFail(task *VideoTask) (bool, string)
}

// VideoSimulationWorker advances mock tasks queued -> running -> succeeded/failed.
// queued->running and running->terminal happen on separate RunOnce ticks so clients
// can observe running via GET between ticks.
type VideoSimulationWorker struct {
	repo     videoSimulationWorkerStore
	content  videoSimulationContentCapturer
	fail     VideoSimulationFailureStrategy
	claimLim int
	lease    time.Duration
}

func NewVideoSimulationWorker(repo videoSimulationWorkerStore, content videoSimulationContentCapturer) *VideoSimulationWorker {
	return &VideoSimulationWorker{
		repo:     repo,
		content:  content,
		claimLim: 20,
		lease:    30 * time.Second,
	}
}

func (w *VideoSimulationWorker) WithFailureStrategy(strategy VideoSimulationFailureStrategy) *VideoSimulationWorker {
	if w == nil {
		return nil
	}
	w.fail = strategy
	return w
}

func (w *VideoSimulationWorker) ClaimRunnable(ctx context.Context, limit int, lease time.Duration) ([]*VideoTask, error) {
	if w == nil || w.repo == nil {
		return nil, nil
	}
	return w.repo.ClaimMockRunnableTasks(ctx, limit, lease)
}

func (w *VideoSimulationWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.repo == nil {
		return nil
	}
	tasks, err := w.repo.ClaimMockRunnableTasks(ctx, w.claimLim, w.lease)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := w.process(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func (w *VideoSimulationWorker) process(ctx context.Context, task *VideoTask) error {
	if task == nil {
		return nil
	}
	current := task
	switch current.Status {
	case VideoStatusSucceeded:
		w.captureSucceeded(ctx, current)
		return nil
	case VideoStatusFailed, VideoStatusCancelled:
		return nil
	case VideoStatusQueued:
		updated, err := w.repo.TransitionSimulationTask(ctx, current.ID, current.Version, VideoStatusQueued, VideoStatusRunning)
		if err != nil {
			if errors.Is(err, ErrVideoTaskTerminalConflict) {
				return nil
			}
			return err
		}
		// Stop after queued->running so status is observable between ticks.
		_ = updated
		return nil
	case VideoStatusRunning, VideoStatusSubmitted:
		// continue to finalize on this tick
	default:
		return nil
	}

	status := VideoStatusSucceeded
	errMsg := ""
	if w.fail != nil {
		if fail, reason := w.fail.ShouldFail(current); fail {
			status = VideoStatusFailed
			errMsg = reason
		}
	}

	result, err := w.repo.FinalizeSimulationTask(ctx, current.ID, current.Version, status, errMsg)
	if err != nil {
		if errors.Is(err, ErrVideoTaskTerminalConflict) {
			return nil
		}
		return err
	}
	if status != VideoStatusSucceeded {
		return nil
	}
	// Fail-open content capture: retry even when Finalize Applied=false (idempotent restart).
	finalTask := *current
	finalTask.Status = VideoStatusSucceeded
	if result.Version > 0 {
		finalTask.Version = result.Version
	}
	w.captureSucceeded(ctx, &finalTask)
	return nil
}

func (w *VideoSimulationWorker) captureSucceeded(ctx context.Context, task *VideoTask) {
	if w == nil || w.content == nil || task == nil {
		return
	}
	_ = w.content.CaptureTaskLinkedContent(ctx, task)
}

// VideoGatewayMockExclusionProbe exposes the real Seedance claim path for isolation tests.
type VideoGatewayMockExclusionProbe struct {
	repo interface {
		ClaimRunnableTasks(ctx context.Context, limit int, lease time.Duration) ([]*VideoTask, error)
	}
}

func NewVideoGatewayMockExclusionProbe(repo interface {
	ClaimRunnableTasks(ctx context.Context, limit int, lease time.Duration) ([]*VideoTask, error)
}) *VideoGatewayMockExclusionProbe {
	return &VideoGatewayMockExclusionProbe{repo: repo}
}

func (p *VideoGatewayMockExclusionProbe) ClaimRunnableTasks(ctx context.Context, limit int, lease time.Duration) ([]*VideoTask, error) {
	if p == nil || p.repo == nil {
		return nil, nil
	}
	return p.repo.ClaimRunnableTasks(ctx, limit, lease)
}
