package service

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

type VideoGatewayRuntime struct {
	worker *VideoGatewayWorker
	cfg    *config.Config
	gate   *SingleSmokeAuthorization
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func ProvideVideoGatewayWorker(repo VideoGatewayRuntimeRepository, encryptor VideoKeyEncryptor, authCache VideoAuthCacheInvalidator, billingCache VideoBillingCacheInvalidator, cfg *config.Config, gate *SingleSmokeAuthorization, archiver VideoAssetArchiver) *VideoGatewayWorker {
	timeout := time.Duration(cfg.VideoGateway.HTTPTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	return NewVideoGatewayWorker(repo, encryptor, func(baseURL, key string) *SeedanceAdapter { return NewSeedanceAdapter(client, baseURL, key) }, authCache, billingCache, cfg, gate, archiver)
}

func ProvideVideoGatewayRuntime(worker *VideoGatewayWorker, cfg *config.Config, gate *SingleSmokeAuthorization) *VideoGatewayRuntime {
	r := &VideoGatewayRuntime{worker: worker, cfg: cfg, gate: gate}
	r.Start()
	return r
}

func (r *VideoGatewayRuntime) Start() {
	if r == nil || r.worker == nil || r.cfg == nil || !r.cfg.VideoGateway.WorkerEnabled || r.gate == nil || !r.gate.Allowed() {
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
				logger.L().Error("video_gateway.worker_run_failed",
					zap.String("component", "service.video_gateway_runtime"),
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

func (r *VideoGatewayRuntime) Stop() {
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

func (r *VideoGatewayRuntime) Running() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancel != nil
}

type DisabledVideoKeyEncryptor struct{}

func (DisabledVideoKeyEncryptor) Encrypt(string) (string, error) {
	return "", errors.New("video worker is disabled")
}
func (DisabledVideoKeyEncryptor) Decrypt(string) (string, error) {
	return "", errors.New("video worker is disabled")
}
