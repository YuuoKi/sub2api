package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSimulationRuntimeRunsWhenRealVideoWorkerIsDisabled(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)
	task, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "runtime", CreationKey: "runtime-with-real-worker-disabled",
	})
	require.NoError(t, err)

	runtime := ProvideVideoSimulationRuntime(
		NewVideoSimulationWorker(repo, repo),
		&config.Config{VideoGateway: config.VideoGatewayConfig{WorkerEnabled: false, WorkerIntervalSeconds: 1}},
	)
	defer runtime.Stop()

	require.Eventually(t, func() bool {
		current, getErr := svc.GetTask(context.Background(), task.ID, 7)
		return getErr == nil && current.Status == VideoStatusSucceeded
	}, 3*time.Second, 25*time.Millisecond)
}
