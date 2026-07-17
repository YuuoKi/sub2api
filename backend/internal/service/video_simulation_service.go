package service

import (
	"context"
	"strings"
)

type videoSimulationAPIKeyLookup interface {
	GetByID(ctx context.Context, id int64) (*APIKey, error)
}

type videoSimulationStore interface {
	GetOrCreateMockProviderAccount(ctx context.Context) (*VideoProviderAccount, error)
	CreateSimulationTask(ctx context.Context, task *VideoTask) (created bool, err error)
	GetSimulationTaskForOwner(ctx context.Context, taskID, userID int64) (*VideoTask, error)
	CancelSimulationTaskForOwner(ctx context.Context, taskID, userID int64) (*VideoTask, error)
	ListSimulationTasksForOwner(ctx context.Context, userID int64) ([]*VideoTask, error)
	InsertVideoTaskEvent(ctx context.Context, taskID int64, eventType string, payload map[string]any) error
}

type videoSimulationContentCapturer interface {
	CaptureTaskLinkedContent(ctx context.Context, task *VideoTask) error
}

// VideoSimulationService creates and owns mock video tasks without billing or network.
type VideoSimulationService struct {
	repo    videoSimulationStore
	keys    videoSimulationAPIKeyLookup
	content videoSimulationContentCapturer
}

func NewVideoSimulationService(repo videoSimulationStore, keys videoSimulationAPIKeyLookup, content videoSimulationContentCapturer) *VideoSimulationService {
	return &VideoSimulationService{repo: repo, keys: keys, content: content}
}

func (s *VideoSimulationService) CreateTask(ctx context.Context, cmd VideoSimulationCreateCommand) (*VideoTask, error) {
	if s == nil || s.repo == nil || s.keys == nil {
		return nil, ErrVideoTaskNotFound
	}
	key, err := s.keys.GetByID(ctx, cmd.APIKeyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrAPIKeyNotFound
	}
	if key.UserID != cmd.UserID {
		return nil, ErrVideoSimulationAPIKeyNotOwned
	}
	if key.Status != StatusAPIKeyActive {
		return nil, ErrVideoSimulationAPIKeyInactive
	}
	if key.GroupID == nil || *key.GroupID <= 0 {
		return nil, ErrVideoSimulationAPIKeyNotOwned
	}

	provider, err := s.repo.GetOrCreateMockProviderAccount(ctx)
	if err != nil {
		return nil, err
	}

	task := &VideoTask{
		APIKeyID:          cmd.APIKeyID,
		GroupID:           *key.GroupID,
		ProviderAccountID: provider.ID,
		Provider:          VideoProviderMock,
		Model:             VideoModelMockVideoV1,
		TaskType:          VideoSimulationTaskTypeTextToVideo,
		Prompt:            strings.TrimSpace(cmd.Prompt),
		Status:            VideoStatusQueued,
		DurationSeconds:   VideoSimulationDurationSeconds,
		Resolution:        VideoSimulationResolution,
		CostAmount:        0,
		Currency:          "USD",
		PricingSource:     VideoPricingSourceInternalSimulation,
		PricingVersion:    VideoPricingVersionSimulationV1,
		CreationKey:       strings.TrimSpace(cmd.CreationKey),
		Version:           1,
		DispatchState:     "pending",
		CreatedBy:         cmd.UserID,
		ReservedCostUSD:   0,
		ReservationState:  VideoReservationNone,
	}
	created, err := s.repo.CreateSimulationTask(ctx, task)
	if err != nil {
		return nil, err
	}
	if created {
		if err := s.repo.InsertVideoTaskEvent(ctx, task.ID, VideoStatusQueued, map[string]any{
			"provider": VideoProviderMock,
			"model":    VideoModelMockVideoV1,
		}); err != nil {
			return nil, err
		}
	}
	return task, nil
}

func (s *VideoSimulationService) GetTask(ctx context.Context, taskID, userID int64) (*VideoTask, error) {
	if s == nil || s.repo == nil {
		return nil, ErrVideoTaskNotFound
	}
	return s.repo.GetSimulationTaskForOwner(ctx, taskID, userID)
}

func (s *VideoSimulationService) ListTasks(ctx context.Context, userID int64) ([]*VideoTask, error) {
	if s == nil || s.repo == nil {
		return nil, ErrVideoTaskNotFound
	}
	return s.repo.ListSimulationTasksForOwner(ctx, userID)
}

func (s *VideoSimulationService) CancelTask(ctx context.Context, taskID, userID int64) (*VideoTask, error) {
	if s == nil || s.repo == nil {
		return nil, ErrVideoTaskNotFound
	}
	return s.repo.CancelSimulationTaskForOwner(ctx, taskID, userID)
}

func (s *VideoSimulationService) OpenSimulationResult(ctx context.Context, taskID, userID int64) (*VideoSimulationResult, error) {
	task, err := s.GetTask(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}
	return openMockSucceededResult(task)
}

func (s *VideoSimulationService) OpenSimulationResultAsAdmin(ctx context.Context, taskID int64) (*VideoSimulationResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrVideoTaskNotFound
	}
	type taskGetter interface {
		GetTask(context.Context, int64) (*VideoTask, error)
	}
	getter, ok := s.repo.(taskGetter)
	if !ok {
		return nil, ErrVideoTaskNotFound
	}
	task, err := getter.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return openMockSucceededResult(task)
}

func openMockSucceededResult(task *VideoTask) (*VideoSimulationResult, error) {
	if task == nil || task.Provider != VideoProviderMock {
		return nil, ErrVideoTaskNotFound
	}
	if task.Status != VideoStatusSucceeded {
		return nil, ErrVideoSimulationResultNotReady
	}
	return BuildSimulationResult(task), nil
}

// SimulationContract describes the internal mock video option for JWT employees.
func (s *VideoSimulationService) SimulationContract() map[string]any {
	return map[string]any{
		"provider":         VideoProviderMock,
		"model":            VideoModelMockVideoV1,
		"label":            "模拟视频结果",
		"media_kind":       "image",
		"duration_seconds": VideoSimulationDurationSeconds,
		"resolution":       VideoSimulationResolution,
		"currency":         "USD",
		"pricing_source":   VideoPricingSourceInternalSimulation,
		"pricing_version":  VideoPricingVersionSimulationV1,
		"cost_amount":      0,
		"network":          false,
		"billing":          false,
	}
}
