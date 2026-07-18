package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

// VideoSimulationRepository is the mock-only persistence surface.
type VideoSimulationRepository interface {
	GetOrCreateMockProviderAccount(ctx context.Context) (*VideoProviderAccount, error)
	CreateSimulationTask(ctx context.Context, task *VideoTask) (created bool, err error)
	GetSimulationTaskForOwner(ctx context.Context, taskID, userID int64) (*VideoTask, error)
	CancelSimulationTaskForOwner(ctx context.Context, taskID, userID int64) (*VideoTask, error)
	ListSimulationTasksForOwner(ctx context.Context, userID int64) ([]*VideoTask, error)
	ClaimMockRunnableTasks(ctx context.Context, limit int, lease time.Duration) ([]*VideoTask, error)
	TransitionSimulationTask(ctx context.Context, taskID, expectedVersion int64, fromStatus, toStatus string) (*VideoTask, error)
	FinalizeSimulationTask(ctx context.Context, taskID, expectedVersion int64, status, errorMessage string) (VideoTaskFinalizationResult, error)
	InsertVideoTaskEvent(ctx context.Context, taskID int64, eventType string, payload map[string]any) error
	ListVideoTaskEvents(ctx context.Context, taskID int64) ([]VideoTaskEvent, error)
	CaptureTaskLinkedContent(ctx context.Context, task *VideoTask) error
	GetTask(ctx context.Context, id int64) (*VideoTask, error)
}

// VideoSimulationRuntime runs the mock-only worker without Seedance smoke gates.
type VideoSimulationRuntime struct {
	worker *VideoSimulationWorker
	cfg    *config.Config
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func ProvideVideoSimulationWorker(repo VideoSimulationRepository) *VideoSimulationWorker {
	return NewVideoSimulationWorker(repo, repo)
}

func ProvideVideoSimulationService(repo VideoSimulationRepository, keys *APIKeyService) *VideoSimulationService {
	return NewVideoSimulationService(repo, keys, repo)
}

func ProvideVideoSimulationRuntime(worker *VideoSimulationWorker, cfg *config.Config) *VideoSimulationRuntime {
	r := &VideoSimulationRuntime{worker: worker, cfg: cfg}
	r.Start()
	return r
}

func (r *VideoSimulationRuntime) Start() {
	if r == nil || r.worker == nil || r.cfg == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	done := make(chan struct{})
	r.done = done
	interval := time.Duration(r.cfg.VideoGateway.WorkerIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if err := r.worker.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
				logger.L().Error("video_simulation.worker_run_failed",
					zap.String("component", "service.video_simulation_runtime"),
					zap.String("error_code", "run_once_failed"),
					zap.Error(err))
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (r *VideoSimulationRuntime) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.cancel, r.done = nil, nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}
