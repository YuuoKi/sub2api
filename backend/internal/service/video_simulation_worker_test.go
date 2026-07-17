package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type alwaysFailSimulationStrategy struct {
	reason string
}

func (s alwaysFailSimulationStrategy) ShouldFail(*VideoTask) (bool, string) {
	return true, s.reason
}

func TestSimulationWorkerEmitsQueuedRunningSucceededEventsAndVersions(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)
	task, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "lifecycle", CreationKey: "sim-life-1",
	})
	require.NoError(t, err)
	require.Equal(t, VideoStatusQueued, task.Status)
	require.EqualValues(t, 1, task.Version)

	worker := NewVideoSimulationWorker(repo, repo)

	// Tick 1: queued -> running only (observable between ticks). Uses real claim filter.
	require.NoError(t, worker.RunOnce(context.Background()))
	mid, err := svc.GetTask(context.Background(), task.ID, 7)
	require.NoError(t, err)
	require.Equal(t, VideoStatusRunning, mid.Status)
	require.Greater(t, mid.Version, int64(1))
	require.Nil(t, mid.CompletedAt)

	// Tick 2: running -> succeeded.
	require.NoError(t, worker.RunOnce(context.Background()))

	got, err := svc.GetTask(context.Background(), task.ID, 7)
	require.NoError(t, err)
	require.Equal(t, VideoStatusSucceeded, got.Status)
	require.Greater(t, got.Version, mid.Version)
	require.NotNil(t, got.CompletedAt)
	require.False(t, got.UpdatedAt.IsZero())

	events, err := repo.ListVideoTaskEvents(context.Background(), task.ID)
	require.NoError(t, err)
	types := make([]string, 0, len(events))
	for _, ev := range events {
		types = append(types, ev.EventType)
	}
	require.Contains(t, types, VideoStatusQueued)
	require.Contains(t, types, VideoStatusRunning)
	require.Contains(t, types, VideoStatusSucceeded)
	require.Zero(t, repo.billing.balanceTouches)
	require.Zero(t, repo.billing.reservations)
	require.Zero(t, repo.billing.dispatches)
}

func TestSimulationWorkerRestartIsIdempotentNoDoubleFinalizeOrContent(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)
	task, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "restart", CreationKey: "sim-restart-1",
	})
	require.NoError(t, err)
	worker := NewVideoSimulationWorker(repo, repo)

	require.NoError(t, worker.RunOnce(context.Background()))
	require.NoError(t, worker.RunOnce(context.Background()))
	firstFinalizes := repo.finalizeCalls
	firstCaptures := repo.contentCaptures
	require.Equal(t, 1, firstFinalizes)
	require.Equal(t, 1, firstCaptures)
	require.True(t, repo.contentCaptured[task.ID])

	// Restart after success with content present: claim filter must not resurface the task.
	require.NoError(t, worker.RunOnce(context.Background()))
	require.Equal(t, firstFinalizes, repo.finalizeCalls)
	require.Equal(t, firstCaptures, repo.contentCaptures)

	got := repo.tasks[task.ID]
	require.Equal(t, VideoStatusSucceeded, got.Status)
}

func TestSimulationWorkerRetriesContentCaptureAfterIdempotentFinalize(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)
	task, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "capture retry", CreationKey: "sim-capture-retry",
	})
	require.NoError(t, err)
	worker := NewVideoSimulationWorker(repo, repo)

	require.NoError(t, worker.RunOnce(context.Background()))
	repo.contentCaptureErr = errors.New("transient content failure")
	require.NoError(t, worker.RunOnce(context.Background()))
	require.Equal(t, VideoStatusSucceeded, repo.tasks[task.ID].Status)
	require.Equal(t, 1, repo.contentCaptures)
	require.False(t, repo.contentCaptured[task.ID])

	// Production-shaped reclaim: succeeded-without-content must be claimable without stub override.
	claimed, err := worker.ClaimRunnable(context.Background(), 10, time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, VideoStatusSucceeded, claimed[0].Status)
	require.EqualValues(t, task.ID, claimed[0].ID)

	repo.contentCaptureErr = nil
	require.NoError(t, worker.RunOnce(context.Background()))
	require.Equal(t, VideoStatusSucceeded, repo.tasks[task.ID].Status)
	require.Equal(t, 2, repo.contentCaptures)
	require.True(t, repo.contentCaptured[task.ID])

	// After capture succeeds, claim must stop resurfacing the task.
	claimed, err = worker.ClaimRunnable(context.Background(), 10, time.Second)
	require.NoError(t, err)
	require.Empty(t, claimed)
	require.NoError(t, worker.RunOnce(context.Background()))
	require.Equal(t, 2, repo.contentCaptures)
}

func TestSimulationWorkerInjectedFailureStrategyReturnsRealFailureReason(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)
	magicPrompt := "FAIL_SIMULATION_PLEASE"
	task, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: magicPrompt, CreationKey: "sim-fail-1",
	})
	require.NoError(t, err)

	okWorker := NewVideoSimulationWorker(repo, repo)
	require.NoError(t, okWorker.RunOnce(context.Background()))
	require.NoError(t, okWorker.RunOnce(context.Background()))
	require.Equal(t, VideoStatusSucceeded, repo.tasks[task.ID].Status)

	task2, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: magicPrompt, CreationKey: "sim-fail-2",
	})
	require.NoError(t, err)
	failWorker := NewVideoSimulationWorker(repo, repo).WithFailureStrategy(alwaysFailSimulationStrategy{reason: "injected_simulation_failure"})
	require.NoError(t, failWorker.RunOnce(context.Background()))
	require.NoError(t, failWorker.RunOnce(context.Background()))
	failed := repo.tasks[task2.ID]
	require.Equal(t, VideoStatusFailed, failed.Status)
	require.Equal(t, "injected_simulation_failure", failed.ErrorMessage)
	require.NotContains(t, failed.ErrorMessage, magicPrompt)
}

func TestSimulationContentCaptureFailureIsFailOpen(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	repo.contentCaptureErr = errors.New("content store unavailable")
	svc := NewVideoSimulationService(repo, keys, repo)
	task, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "content fail-open", CreationKey: "sim-content-1",
	})
	require.NoError(t, err)
	worker := NewVideoSimulationWorker(repo, repo)
	require.NoError(t, worker.RunOnce(context.Background()))
	require.NoError(t, worker.RunOnce(context.Background()))
	require.Equal(t, VideoStatusSucceeded, repo.tasks[task.ID].Status)
	require.Equal(t, 1, repo.contentCaptures)
	require.False(t, repo.contentCaptured[task.ID])
}

func TestSimulationClaimIsolationBetweenMockAndSeedanceWorkers(t *testing.T) {
	repo := newSimulationRepoStub()
	mockTask := &VideoTask{ID: 501, Provider: VideoProviderMock, Model: VideoModelMockVideoV1, Status: VideoStatusQueued, Version: 1}
	seedanceTask := &VideoTask{ID: 502, Provider: "seedance", Model: SeedanceModel, Status: VideoStatusQueued, Version: 1}
	repo.tasks[501] = mockTask
	repo.tasks[502] = seedanceTask
	// Leave claimedMock/claimedReal nil so stub filters from tasks map.

	mockWorker := NewVideoSimulationWorker(repo, repo)
	claimedMock, err := mockWorker.ClaimRunnable(context.Background(), 10, time.Second)
	require.NoError(t, err)
	require.Len(t, claimedMock, 1)
	require.Equal(t, VideoProviderMock, claimedMock[0].Provider)
	require.EqualValues(t, 501, claimedMock[0].ID)

	realClaimer := NewVideoGatewayMockExclusionProbe(repo)
	claimedReal, err := realClaimer.ClaimRunnableTasks(context.Background(), 10, time.Second)
	require.NoError(t, err)
	require.Len(t, claimedReal, 1)
	require.Equal(t, "seedance", claimedReal[0].Provider)
	require.EqualValues(t, 502, claimedReal[0].ID)
}

func TestSimulationClaimDoesNotReclaimSeedanceSucceededWithoutContent(t *testing.T) {
	repo := newSimulationRepoStub()
	now := time.Now().UTC()
	repo.tasks[601] = &VideoTask{
		ID: 601, Provider: "seedance", Model: SeedanceModel, Status: VideoStatusSucceeded,
		Version: 3, CompletedAt: &now,
	}
	repo.tasks[602] = &VideoTask{
		ID: 602, Provider: VideoProviderMock, Model: VideoModelMockVideoV1, Status: VideoStatusSucceeded,
		Version: 3, CompletedAt: &now,
	}

	mockWorker := NewVideoSimulationWorker(repo, repo)
	claimed, err := mockWorker.ClaimRunnable(context.Background(), 10, time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, VideoProviderMock, claimed[0].Provider)
	require.EqualValues(t, 602, claimed[0].ID)
}

func cloneSimTask(task *VideoTask) *VideoTask {
	if task == nil {
		return nil
	}
	cp := *task
	return &cp
}
