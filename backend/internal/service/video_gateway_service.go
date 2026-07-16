package service

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

var ErrVideoBudgetRejected = errors.New("video budget rejected")

type VideoAuthCacheInvalidator interface {
	InvalidateAuthCacheByUserID(context.Context, int64)
}

type VideoBillingCacheInvalidator interface {
	InvalidateUserBalance(context.Context, int64) error
	InvalidateAPIKeyRateLimit(context.Context, int64) error
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
	repo         VideoGatewayRuntimeRepository
	gate         *SingleSmokeAuthorization
	cfg          *config.Config
	authCache    VideoAuthCacheInvalidator
	billingCache VideoBillingCacheInvalidator
}

func videoMaximumUSD(cfg *config.Config) (float64, error) {
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
	return math.Round((pricing.TinyRealMaximumCNY/pricing.USDCNYExchangeRate)*1e8) / 1e8, nil
}

func NewVideoGatewayService(repo VideoGatewayRuntimeRepository, gate *SingleSmokeAuthorization, cfg *config.Config, authCache VideoAuthCacheInvalidator, billingCache VideoBillingCacheInvalidator) *VideoGatewayService {
	return &VideoGatewayService{repo: repo, gate: gate, cfg: cfg, authCache: authCache, billingCache: billingCache}
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
	if s.cfg == nil || !s.cfg.VideoGateway.WorkerEnabled || s.gate == nil || !s.gate.Allowed() {
		return nil, ErrVideoRealDispatchDenied
	}
	provider, err := s.repo.GetVideoProvider(ctx, cmd.ProviderAccountID, cmd.Scope.GroupID)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled || provider.Provider != "seedance" {
		return nil, ErrVideoProviderNotFound
	}
	maximumUSD, err := videoMaximumUSD(s.cfg)
	if err != nil {
		return nil, err
	}
	task := &VideoTask{APIKeyID: cmd.Scope.APIKeyID, GroupID: cmd.Scope.GroupID, ProviderAccountID: provider.ID,
		Provider: provider.Provider, Model: SeedanceModel, TaskType: "text_to_video", Prompt: strings.TrimSpace(cmd.Prompt),
		Status: VideoStatusQueued, CreationKey: strings.TrimSpace(cmd.CreationKey), CreatedBy: cmd.Scope.UserID,
		DurationSeconds: 4, Resolution: "720p", Currency: "USD", ReservedCostUSD: maximumUSD, ReservationState: VideoReservationReserved}
	if err := s.repo.ReserveAndCreateTask(ctx, task, maximumUSD); err != nil {
		return nil, err
	}
	invalidateVideoCaches(ctx, s.authCache, s.billingCache, task.CreatedBy, task.APIKeyID)
	return task, nil
}

func (s *VideoGatewayService) GetTask(ctx context.Context, id int64, scope VideoTaskScope) (*VideoTask, error) {
	return s.repo.GetTaskForScope(ctx, id, scope)
}

func (s *VideoGatewayService) CancelTask(ctx context.Context, id int64, scope VideoTaskScope) (*VideoTask, error) {
	task, err := s.repo.CancelTaskForScope(ctx, id, scope)
	if err == nil {
		invalidateVideoCaches(ctx, s.authCache, s.billingCache, scope.UserID, scope.APIKeyID)
	}
	return task, err
}
