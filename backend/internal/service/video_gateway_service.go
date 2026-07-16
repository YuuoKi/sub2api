package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

var ErrVideoBudgetRejected = errors.New("video budget rejected")

type VideoBudgetGuard interface {
	AuthorizeVideo(context.Context, VideoTaskScope) (float64, error)
}

type VideoTaskCreateCommand struct {
	Scope                 VideoTaskScope
	ProviderAccountID     int64
	Prompt                string
	Duration              int
	Resolution            string
	SingleSmokeAuthorized bool
	CreationKey           string
}

type VideoGatewayService struct {
	repo   VideoGatewayRuntimeRepository
	gate   *SingleSmokeAuthorization
	budget VideoBudgetGuard
}

type VideoBalanceBudgetGuard struct {
	billing *BillingCacheService
	cfg     *config.Config
}

func NewVideoBalanceBudgetGuard(billing *BillingCacheService, cfg *config.Config) *VideoBalanceBudgetGuard {
	return &VideoBalanceBudgetGuard{billing: billing, cfg: cfg}
}

func (g *VideoBalanceBudgetGuard) AuthorizeVideo(ctx context.Context, scope VideoTaskScope) (float64, error) {
	if g == nil || g.billing == nil || g.cfg == nil {
		return 0, errors.New("video billing configuration is required")
	}
	estimateUSD, err := videoEstimateUSD(g.cfg)
	if err != nil {
		return 0, err
	}
	balance, err := g.billing.GetUserBalance(ctx, scope.UserID)
	if err != nil {
		return 0, err
	}
	if balance < estimateUSD {
		return 0, ErrVideoBudgetRejected
	}
	return estimateUSD, nil
}

func videoEstimateUSD(cfg *config.Config) (float64, error) {
	if cfg == nil {
		return 0, errors.New("video pricing configuration is incomplete")
	}
	pricing := cfg.VideoGateway
	if pricing.SeedanceCNYPerMillionTokens <= 0 || pricing.USDCNYExchangeRate <= 0 || pricing.TinyRealEstimateCNY <= 0 || pricing.TinyRealMaximumCNY <= 0 {
		return 0, errors.New("video pricing configuration is incomplete")
	}
	if pricing.TinyRealEstimateCNY > pricing.TinyRealMaximumCNY {
		return 0, ErrVideoBudgetRejected
	}
	return math.Round((pricing.TinyRealEstimateCNY/pricing.USDCNYExchangeRate)*1e8) / 1e8, nil
}

func NewVideoGatewayService(repo VideoGatewayRuntimeRepository, gate *SingleSmokeAuthorization, budget VideoBudgetGuard) *VideoGatewayService {
	return &VideoGatewayService{repo: repo, gate: gate, budget: budget}
}

func (s *VideoGatewayService) ListProviders(ctx context.Context, scope VideoTaskScope) ([]VideoProviderAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("video gateway repository is required")
	}
	return s.repo.ListEnabledVideoProviders(ctx, scope.GroupID)
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
	if s.gate == nil || !s.gate.Allowed() {
		return nil, ErrVideoRealDispatchDenied
	}
	provider, err := s.repo.GetVideoProvider(ctx, cmd.ProviderAccountID, cmd.Scope.GroupID)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled || provider.Provider != "seedance" {
		return nil, ErrVideoProviderNotFound
	}
	if s.budget == nil {
		return nil, errors.New("video budget guard is required")
	}
	if _, err := s.budget.AuthorizeVideo(ctx, cmd.Scope); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVideoBudgetRejected, err)
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
