package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrVideoBudgetRejected = errors.New("video budget rejected")

type VideoBudgetGuard interface {
	AuthorizeVideo(context.Context, VideoTaskScope, float64) error
}

type VideoTaskCreateCommand struct {
	Scope                 VideoTaskScope
	ProviderAccountID     int64
	Prompt                string
	Duration              int
	Resolution            string
	SingleSmokeAuthorized bool
	CreationKey           string
	MaximumCost           float64
}

type VideoGatewayService struct {
	repo   VideoGatewayRuntimeRepository
	gate   *SingleSmokeAuthorization
	budget VideoBudgetGuard
}

type VideoBalanceBudgetGuard struct{ billing *BillingCacheService }

func NewVideoBalanceBudgetGuard(billing *BillingCacheService) *VideoBalanceBudgetGuard {
	return &VideoBalanceBudgetGuard{billing: billing}
}

func (g *VideoBalanceBudgetGuard) AuthorizeVideo(ctx context.Context, scope VideoTaskScope, maximumCost float64) error {
	if g == nil || g.billing == nil {
		return errors.New("billing cache service is required")
	}
	balance, err := g.billing.GetUserBalance(ctx, scope.UserID)
	if err != nil {
		return err
	}
	if maximumCost <= 0 || balance < maximumCost {
		return ErrVideoBudgetRejected
	}
	return nil
}

func NewVideoGatewayService(repo VideoGatewayRuntimeRepository, gate *SingleSmokeAuthorization, budget VideoBudgetGuard) *VideoGatewayService {
	return &VideoGatewayService{repo: repo, gate: gate, budget: budget}
}

func (s *VideoGatewayService) ListProviders(ctx context.Context) ([]VideoProviderAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("video gateway repository is required")
	}
	return s.repo.ListEnabledVideoProviders(ctx)
}

func (s *VideoGatewayService) CreateTask(ctx context.Context, cmd VideoTaskCreateCommand) (*VideoTask, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("video gateway repository is required")
	}
	input := VideoCreateRequest{Prompt: cmd.Prompt, Duration: cmd.Duration, Resolution: cmd.Resolution}
	if err := ValidateTinyRealContract(input); err != nil {
		return nil, err
	}
	if cmd.Scope.UserID <= 0 || cmd.Scope.APIKeyID <= 0 || cmd.Scope.GroupID <= 0 {
		return nil, errors.New("complete employee video scope is required")
	}
	if !cmd.SingleSmokeAuthorized {
		return nil, ErrVideoRealDispatchDenied
	}
	if s.budget == nil {
		return nil, errors.New("video budget guard is required")
	}
	if cmd.MaximumCost <= 0 {
		return nil, ErrVideoBudgetRejected
	}
	if err := s.budget.AuthorizeVideo(ctx, cmd.Scope, cmd.MaximumCost); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVideoBudgetRejected, err)
	}
	if err := s.gate.Consume(); err != nil {
		return nil, err
	}
	provider, err := s.repo.GetVideoProvider(ctx, cmd.ProviderAccountID)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled || provider.Provider != "seedance" {
		return nil, errors.New("seedance provider is unavailable")
	}
	task := &VideoTask{APIKeyID: cmd.Scope.APIKeyID, GroupID: cmd.Scope.GroupID, ProviderAccountID: provider.ID,
		Provider: provider.Provider, Model: SeedanceModel, TaskType: "text_to_video", Prompt: strings.TrimSpace(cmd.Prompt),
		Status: VideoStatusQueued, CreationKey: strings.TrimSpace(cmd.CreationKey), CreatedBy: cmd.Scope.UserID,
		DurationSeconds: 4, Resolution: "720p", Currency: "USD"}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *VideoGatewayService) GetTask(ctx context.Context, id int64, scope VideoTaskScope) (*VideoTask, error) {
	return s.repo.GetTaskForScope(ctx, id, scope)
}

func (s *VideoGatewayService) CancelTask(ctx context.Context, id int64, scope VideoTaskScope) (*VideoTask, error) {
	return s.repo.CancelTaskForScope(ctx, id, scope)
}
