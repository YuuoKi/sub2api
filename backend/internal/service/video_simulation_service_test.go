package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Phase 1 RED: these tests define the Task 2E simulation contract.
// Intended production symbols (CreateSimulationTask path) do not exist yet.

type simulationAPIKeyRepoStub struct {
	keys map[int64]*APIKey
}

func (s *simulationAPIKeyRepoStub) GetByID(_ context.Context, id int64) (*APIKey, error) {
	if s == nil || s.keys == nil {
		return nil, ErrAPIKeyNotFound
	}
	key, ok := s.keys[id]
	if !ok || key == nil {
		return nil, ErrAPIKeyNotFound
	}
	cp := *key
	return &cp, nil
}

type simulationBillingProbe struct {
	balanceTouches int
	frozenTouches  int
	quotaTouches   int
	usageTouches   int
	reservations   int
	dispatches     int
}

type simulationRepoStub struct {
	provider           VideoProviderAccount
	tasks              map[int64]*VideoTask
	byCreationKey      map[string]*VideoTask
	events             []VideoTaskEvent
	nextID             int64
	billing            *simulationBillingProbe
	createTouches      int
	reserveTouches     int
	transitionCalls    int
	finalizeCalls      int
	contentCaptures    int
	contentCaptureErr  error
	contentCaptured    map[int64]bool
	forceCreateErr     error
	claimedMock        []*VideoTask
	claimedReal        []*VideoTask
}

func newSimulationRepoStub() *simulationRepoStub {
	return &simulationRepoStub{
		provider: VideoProviderAccount{
			ID: 9001, Provider: VideoProviderMock, DisplayName: "Internal Mock Video",
			Enabled: true, DefaultModel: VideoModelMockVideoV1,
		},
		tasks:         map[int64]*VideoTask{},
		byCreationKey: map[string]*VideoTask{},
		nextID:        100,
		billing:       &simulationBillingProbe{},
	}
}

func (r *simulationRepoStub) GetOrCreateMockProviderAccount(context.Context) (*VideoProviderAccount, error) {
	cp := r.provider
	return &cp, nil
}

func (r *simulationRepoStub) ReserveAndCreateTask(_ context.Context, _ *VideoTask, _ float64) error {
	r.reserveTouches++
	if r.billing != nil {
		r.billing.reservations++
		r.billing.balanceTouches++
		r.billing.frozenTouches++
	}
	return errors.New("ReserveAndCreateTask must not be used for simulation")
}

func (r *simulationRepoStub) CreateSimulationTask(_ context.Context, task *VideoTask) (bool, error) {
	r.createTouches++
	if r.forceCreateErr != nil {
		return false, r.forceCreateErr
	}
	if task != nil && (task.CostAmount != 0 || task.ReservedCostUSD != 0 || task.ReservationState == VideoReservationReserved) {
		if r.billing != nil {
			r.billing.reservations++
			r.billing.balanceTouches++
		}
	}
	if task.CreationKey != "" {
		if existing, ok := r.byCreationKey[task.CreationKey]; ok {
			if existing.Provider != VideoProviderMock ||
				existing.CreatedBy != task.CreatedBy ||
				existing.APIKeyID != task.APIKeyID {
				return false, ErrVideoSimulationCreationKeyConflict
			}
			*task = *existing
			return false, nil
		}
	}
	r.nextID++
	task.ID = r.nextID
	task.CreatedAt = time.Now().UTC()
	task.UpdatedAt = task.CreatedAt
	cp := *task
	r.tasks[task.ID] = &cp
	if task.CreationKey != "" {
		r.byCreationKey[task.CreationKey] = &cp
	}
	return true, nil
}

func (r *simulationRepoStub) GetSimulationTaskForOwner(_ context.Context, taskID, userID int64) (*VideoTask, error) {
	task, ok := r.tasks[taskID]
	if !ok {
		return nil, ErrVideoTaskNotFound
	}
	if task.CreatedBy != userID {
		return nil, ErrVideoTaskForbidden
	}
	if task.Provider != VideoProviderMock {
		return nil, ErrVideoTaskNotFound
	}
	cp := *task
	return &cp, nil
}

func (r *simulationRepoStub) CancelSimulationTaskForOwner(_ context.Context, taskID, userID int64) (*VideoTask, error) {
	task, ok := r.tasks[taskID]
	if !ok {
		return nil, ErrVideoTaskNotFound
	}
	if task.CreatedBy != userID {
		return nil, ErrVideoTaskForbidden
	}
	if task.Provider != VideoProviderMock {
		return nil, ErrVideoTaskNotFound
	}
	switch task.Status {
	case VideoStatusSucceeded, VideoStatusFailed, VideoStatusCancelled:
		return nil, ErrVideoCancelConflict
	}
	task.Status = VideoStatusCancelled
	task.Version++
	now := time.Now().UTC()
	task.CompletedAt = &now
	task.UpdatedAt = now
	cp := *task
	return &cp, nil
}

func (r *simulationRepoStub) ListSimulationTasksForOwner(_ context.Context, userID int64) ([]*VideoTask, error) {
	out := make([]*VideoTask, 0)
	for _, task := range r.tasks {
		if task.CreatedBy == userID && task.Provider == VideoProviderMock {
			cp := *task
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *simulationRepoStub) ClaimMockRunnableTasks(context.Context, int, time.Duration) ([]*VideoTask, error) {
	if r.claimedMock != nil {
		return r.claimedMock, nil
	}
	out := make([]*VideoTask, 0)
	for _, task := range r.tasks {
		if task.Provider != VideoProviderMock {
			continue
		}
		switch task.Status {
		case VideoStatusQueued, VideoStatusSubmitted, VideoStatusRunning:
			out = append(out, cloneSimTask(task))
		case VideoStatusSucceeded:
			// Mirror production ClaimMockRunnableTasks: reclaim succeeded mock tasks
			// that still lack generation-content within the reclaim window.
			if r.contentCaptured[task.ID] {
				continue
			}
			if task.CompletedAt == nil {
				continue
			}
			if time.Since(*task.CompletedAt) > 30*time.Minute {
				continue
			}
			out = append(out, cloneSimTask(task))
		}
	}
	return out, nil
}

func (r *simulationRepoStub) ClaimRunnableTasks(context.Context, int, time.Duration) ([]*VideoTask, error) {
	if r.claimedReal != nil {
		return r.claimedReal, nil
	}
	out := make([]*VideoTask, 0)
	for _, task := range r.tasks {
		if task.Provider == VideoProviderMock {
			continue
		}
		switch task.Status {
		case VideoStatusQueued, VideoStatusSubmitted, VideoStatusRunning:
			out = append(out, cloneSimTask(task))
		}
	}
	return out, nil
}

func (r *simulationRepoStub) GetTask(_ context.Context, id int64) (*VideoTask, error) {
	task, ok := r.tasks[id]
	if !ok {
		return nil, ErrVideoTaskNotFound
	}
	cp := *task
	return &cp, nil
}

func (r *simulationRepoStub) TransitionSimulationTask(_ context.Context, taskID, expectedVersion int64, fromStatus, toStatus string) (*VideoTask, error) {
	r.transitionCalls++
	task, ok := r.tasks[taskID]
	if !ok {
		return nil, ErrVideoTaskNotFound
	}
	if task.Version != expectedVersion || task.Status != fromStatus {
		return nil, ErrVideoTaskTerminalConflict
	}
	task.Status = toStatus
	task.Version++
	task.UpdatedAt = time.Now().UTC()
	r.events = append(r.events, VideoTaskEvent{TaskID: taskID, EventType: toStatus, CreatedAt: task.UpdatedAt})
	cp := *task
	return &cp, nil
}

func (r *simulationRepoStub) FinalizeSimulationTask(_ context.Context, taskID, expectedVersion int64, status, errorMessage string) (VideoTaskFinalizationResult, error) {
	r.finalizeCalls++
	task, ok := r.tasks[taskID]
	if !ok {
		return VideoTaskFinalizationResult{}, ErrVideoTaskNotFound
	}
	if task.Status == VideoStatusSucceeded || task.Status == VideoStatusFailed || task.Status == VideoStatusCancelled {
		if task.Status == status {
			return VideoTaskFinalizationResult{Applied: false, Idempotent: true, Status: task.Status, Version: task.Version}, nil
		}
		return VideoTaskFinalizationResult{}, ErrVideoTaskTerminalConflict
	}
	if task.Version != expectedVersion {
		return VideoTaskFinalizationResult{}, ErrVideoTaskTerminalConflict
	}
	task.Status = status
	task.ErrorMessage = errorMessage
	task.Version++
	now := time.Now().UTC()
	task.CompletedAt = &now
	task.UpdatedAt = now
	r.events = append(r.events, VideoTaskEvent{TaskID: taskID, EventType: status, CreatedAt: now})
	return VideoTaskFinalizationResult{Applied: true, Status: status, Version: task.Version}, nil
}

func (r *simulationRepoStub) InsertVideoTaskEvent(_ context.Context, taskID int64, eventType string, _ map[string]any) error {
	r.events = append(r.events, VideoTaskEvent{TaskID: taskID, EventType: eventType, CreatedAt: time.Now().UTC()})
	return nil
}

func (r *simulationRepoStub) ListVideoTaskEvents(_ context.Context, taskID int64) ([]VideoTaskEvent, error) {
	out := make([]VideoTaskEvent, 0)
	for _, ev := range r.events {
		if ev.TaskID == taskID {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (r *simulationRepoStub) CaptureTaskLinkedContent(_ context.Context, task *VideoTask) error {
	r.contentCaptures++
	if r.contentCaptureErr != nil {
		return r.contentCaptureErr
	}
	if task != nil {
		if r.contentCaptured == nil {
			r.contentCaptured = make(map[int64]bool)
		}
		r.contentCaptured[task.ID] = true
	}
	return nil
}

func ownedActiveKey(userID, keyID, groupID int64) *APIKey {
	gid := groupID
	return &APIKey{ID: keyID, UserID: userID, GroupID: &gid, Status: StatusAPIKeyActive, Name: "owned-active"}
}

func TestSimulationCreateRequiresOwnedActiveAPIKey(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{
		11: ownedActiveKey(7, 11, 3),
	}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	task, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "a calm office hallway", CreationKey: "sim-create-1",
	})
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Greater(t, task.ID, int64(0))
	require.Equal(t, VideoProviderMock, task.Provider)
	require.Equal(t, VideoModelMockVideoV1, task.Model)
	require.Equal(t, VideoStatusQueued, task.Status)
	require.Equal(t, "USD", task.Currency)
	require.Equal(t, VideoPricingSourceInternalSimulation, task.PricingSource)
	require.Equal(t, VideoPricingVersionSimulationV1, task.PricingVersion)
	require.Nil(t, task.PricingCNYPerMillionCompletionTokens)
	require.Nil(t, task.PricingUSDCNYExchangeRate)
	require.Nil(t, task.PricingMaximumCNY)
	require.Equal(t, 4, task.DurationSeconds)
	require.Equal(t, "720p", task.Resolution)
	require.EqualValues(t, 7, task.CreatedBy)
	require.EqualValues(t, 11, task.APIKeyID)
	require.EqualValues(t, 3, task.GroupID)
	require.Equal(t, 0.0, task.CostAmount)
	require.Equal(t, 0.0, task.ReservedCostUSD)
	require.NotEqual(t, VideoReservationReserved, task.ReservationState)
	require.Zero(t, repo.billing.balanceTouches)
	require.Zero(t, repo.billing.frozenTouches)
	require.Zero(t, repo.billing.quotaTouches)
	require.Zero(t, repo.billing.usageTouches)
	require.Zero(t, repo.billing.reservations)
	require.Zero(t, repo.billing.dispatches)
}

func TestSimulationCreateRejectsDisabledAndForeignAPIKey(t *testing.T) {
	gid := int64(3)
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{
		12: {ID: 12, UserID: 7, GroupID: &gid, Status: StatusAPIKeyDisabled},
		13: ownedActiveKey(99, 13, 3),
	}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	_, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 12, Prompt: "disabled key", CreationKey: "sim-disabled",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrVideoSimulationAPIKeyInactive) || errors.Is(err, ErrAPIKeyNotFound) || errors.Is(err, ErrVideoTaskForbidden))

	_, err = svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 13, Prompt: "foreign key", CreationKey: "sim-foreign",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrVideoSimulationAPIKeyNotOwned) || errors.Is(err, ErrVideoTaskForbidden) || errors.Is(err, ErrAPIKeyNotFound))
	require.Empty(t, repo.tasks)
}

func TestSimulationCreateIdempotentOnCreationKey(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)
	cmd := VideoSimulationCreateCommand{UserID: 7, APIKeyID: 11, Prompt: "same key twice", CreationKey: "sim-idem-1"}

	first, err := svc.CreateTask(context.Background(), cmd)
	require.NoError(t, err)
	second, err := svc.CreateTask(context.Background(), cmd)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Len(t, repo.tasks, 1)
}

func TestSimulationCreateDoesNotTouchBillingOrDispatch(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	_, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "zero cost", CreationKey: "sim-billing-1",
	})
	require.NoError(t, err)
	require.Greater(t, repo.createTouches, 0, "create path must exercise CreateSimulationTask")
	require.Zero(t, repo.reserveTouches, "create must not call ReserveAndCreateTask")
	require.Zero(t, repo.billing.balanceTouches)
	require.Zero(t, repo.billing.frozenTouches)
	require.Zero(t, repo.billing.quotaTouches)
	require.Zero(t, repo.billing.usageTouches)
	require.Zero(t, repo.billing.reservations)
	require.Zero(t, repo.billing.dispatches)
}

func TestSimulationCreateRejectsOversizedPrompt(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	huge := strings.Repeat("x", maxGenerationPromptMaxBytes+1)
	_, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: huge, CreationKey: "sim-huge",
	})
	require.ErrorIs(t, err, ErrVideoSimulationPromptTooLarge)
	require.Empty(t, repo.tasks)
	require.Zero(t, repo.createTouches)
}

func TestSimulationListCapsItemsAndTruncatesPrompt(t *testing.T) {
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, nil, repo)
	huge := strings.Repeat("p", VideoSimulationListPromptMaxBytes+2048)
	for i := 0; i < VideoSimulationListMaxItems+25; i++ {
		id := int64(1000 + i)
		repo.tasks[id] = &VideoTask{
			ID: id, Provider: VideoProviderMock, CreatedBy: 7, Status: VideoStatusQueued, Prompt: huge,
		}
	}

	listed, err := svc.ListTasks(context.Background(), 7)
	require.NoError(t, err)
	require.LessOrEqual(t, len(listed), VideoSimulationListMaxItems)
	require.NotEmpty(t, listed)
	for _, task := range listed {
		require.LessOrEqual(t, len(task.Prompt), VideoSimulationListPromptMaxBytes)
	}
}

func TestSimulationCancelOwnerNonterminalOKForeignAndTerminalReject(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	queued, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "cancel me", CreationKey: "sim-cancel-1",
	})
	require.NoError(t, err)

	cancelled, err := svc.CancelTask(context.Background(), queued.ID, 7)
	require.NoError(t, err)
	require.Equal(t, VideoStatusCancelled, cancelled.Status)

	_, err = svc.CancelTask(context.Background(), queued.ID, 7)
	require.ErrorIs(t, err, ErrVideoCancelConflict)

	other, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "foreign cancel", CreationKey: "sim-cancel-2",
	})
	require.NoError(t, err)
	_, err = svc.CancelTask(context.Background(), other.ID, 8)
	require.ErrorIs(t, err, ErrVideoTaskForbidden)
}

func TestSimulationOwnedSeedanceTaskHiddenFromSimulationAPIs(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	seedance := &VideoTask{
		ID: 601, Provider: "seedance", Model: SeedanceModel, Status: VideoStatusQueued,
		CreatedBy: 7, APIKeyID: 11, GroupID: 3, Version: 1, Prompt: "real seedance",
	}
	repo.tasks[601] = seedance

	_, err := svc.GetTask(context.Background(), 601, 7)
	require.ErrorIs(t, err, ErrVideoTaskNotFound)

	_, err = svc.CancelTask(context.Background(), 601, 7)
	require.ErrorIs(t, err, ErrVideoTaskNotFound)
	require.Equal(t, VideoStatusQueued, repo.tasks[601].Status, "cancel must not mutate Seedance tasks")

	_, err = svc.OpenSimulationResult(context.Background(), 601, 7)
	require.ErrorIs(t, err, ErrVideoTaskNotFound)
}

func TestSimulationCreationKeyConflictCrossUserAndSeedance(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{
		11: ownedActiveKey(7, 11, 3),
		21: ownedActiveKey(8, 21, 4),
	}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	first, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "owner a", CreationKey: "shared-key",
	})
	require.NoError(t, err)
	require.NotNil(t, first)

	_, err = svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 8, APIKeyID: 21, Prompt: "owner b", CreationKey: "shared-key",
	})
	require.ErrorIs(t, err, ErrVideoSimulationCreationKeyConflict)

	seedanceKey := "seedance-collision"
	repo.byCreationKey[seedanceKey] = &VideoTask{
		ID: 777, Provider: "seedance", Model: SeedanceModel, Status: VideoStatusQueued,
		CreatedBy: 7, APIKeyID: 11, CreationKey: seedanceKey, Prompt: "must not leak",
	}
	_, err = svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "mock collide", CreationKey: seedanceKey,
	})
	require.ErrorIs(t, err, ErrVideoSimulationCreationKeyConflict)

	replay, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "owner a", CreationKey: "shared-key",
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, replay.ID)

	queuedEvents := 0
	for _, ev := range repo.events {
		if ev.TaskID == first.ID && ev.EventType == VideoStatusQueued {
			queuedEvents++
		}
	}
	require.Equal(t, 1, queuedEvents, "creation_key replay must not re-insert queued event")
}

func TestSimulationResultRequiresMockSucceeded(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{11: ownedActiveKey(7, 11, 3)}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	queued, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "not ready", CreationKey: "sim-result-queued",
	})
	require.NoError(t, err)

	_, err = svc.OpenSimulationResult(context.Background(), queued.ID, 7)
	require.ErrorIs(t, err, ErrVideoSimulationResultNotReady)

	repo.tasks[queued.ID].Status = VideoStatusRunning
	_, err = svc.OpenSimulationResult(context.Background(), queued.ID, 7)
	require.ErrorIs(t, err, ErrVideoSimulationResultNotReady)

	repo.tasks[queued.ID].Status = VideoStatusSucceeded
	result, err := svc.OpenSimulationResult(context.Background(), queued.ID, 7)
	require.NoError(t, err)
	require.NotNil(t, result)

	repo.tasks[queued.ID].Status = VideoStatusSucceeded
	repo.tasks[queued.ID].Provider = "seedance"
	_, err = svc.OpenSimulationResult(context.Background(), queued.ID, 7)
	require.ErrorIs(t, err, ErrVideoTaskNotFound)

	_, err = svc.OpenSimulationResultAsAdmin(context.Background(), queued.ID)
	require.ErrorIs(t, err, ErrVideoTaskNotFound)

	mockSucceeded := &VideoTask{
		ID: 902, Provider: VideoProviderMock, Status: VideoStatusSucceeded,
		CreatedBy: 7, Prompt: "admin ok",
	}
	repo.tasks[902] = mockSucceeded
	adminResult, err := svc.OpenSimulationResultAsAdmin(context.Background(), 902)
	require.NoError(t, err)
	require.NotNil(t, adminResult)

	repo.tasks[903] = &VideoTask{ID: 903, Provider: VideoProviderMock, Status: VideoStatusQueued, CreatedBy: 7}
	_, err = svc.OpenSimulationResultAsAdmin(context.Background(), 903)
	require.ErrorIs(t, err, ErrVideoSimulationResultNotReady)
}

func TestSimulationListOwnMockTasksOnly(t *testing.T) {
	keys := &simulationAPIKeyRepoStub{keys: map[int64]*APIKey{
		11: ownedActiveKey(7, 11, 3),
		21: ownedActiveKey(8, 21, 4),
	}}
	repo := newSimulationRepoStub()
	svc := NewVideoSimulationService(repo, keys, repo)

	mine, err := svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 7, APIKeyID: 11, Prompt: "mine", CreationKey: "list-mine",
	})
	require.NoError(t, err)
	_, err = svc.CreateTask(context.Background(), VideoSimulationCreateCommand{
		UserID: 8, APIKeyID: 21, Prompt: "theirs", CreationKey: "list-theirs",
	})
	require.NoError(t, err)
	repo.tasks[999] = &VideoTask{ID: 999, Provider: "seedance", CreatedBy: 7, Status: VideoStatusQueued}

	listed, err := svc.ListTasks(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, mine.ID, listed[0].ID)
	require.Equal(t, VideoProviderMock, listed[0].Provider)
}
