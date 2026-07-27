package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

var ErrVideoBudgetRejected = errors.New("video budget rejected")

type VideoAuthCacheInvalidator interface {
	InvalidateAuthCacheByUserID(context.Context, int64) error
}

type VideoBillingCacheInvalidator interface {
	InvalidateUserBalance(context.Context, int64) error
	InvalidateAPIKeyRateLimit(context.Context, int64) error
}

type VideoTaskCreateCommand struct {
	Scope             VideoTaskScope
	ProviderAccountID int64
	Prompt            string
	Model             string
	Content           []VideoContentItem
	Ratio             string
	GenerateAudio     bool
	ReturnLastFrame   bool
	Watermark         bool
	Duration          int
	Resolution        string
	CreationKey       string
}

type VideoGatewayService struct {
	repo          VideoGatewayRuntimeRepository
	gate          *SingleSmokeAuthorization
	cfg           *config.Config
	authCache     VideoAuthCacheInvalidator
	billingCache  VideoBillingCacheInvalidator
	assetStore    *VideoAssetStore
	encryptor     VideoKeyEncryptor
	clientFactory func(string, string, string) VideoProviderClient
}

// ConfigureProviderClientFactory wires dispatched HC cancellation without
// changing the existing constructor used by legacy callers.
func (s *VideoGatewayService) ConfigureProviderClientFactory(encryptor VideoKeyEncryptor, factory func(string, string, string) VideoProviderClient) {
	if s != nil {
		s.encryptor, s.clientFactory = encryptor, factory
	}
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

func NewVideoGatewayService(repo VideoGatewayRuntimeRepository, gate *SingleSmokeAuthorization, cfg *config.Config, authCache VideoAuthCacheInvalidator, billingCache VideoBillingCacheInvalidator, stores ...*VideoAssetStore) *VideoGatewayService {
	var assetStore *VideoAssetStore
	if len(stores) > 0 {
		assetStore = stores[0]
	}
	return &VideoGatewayService{repo: repo, gate: gate, cfg: cfg, authCache: authCache, billingCache: billingCache, assetStore: assetStore}
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
	if cmd.Scope.UserID <= 0 || cmd.Scope.APIKeyID <= 0 || cmd.Scope.GroupID <= 0 {
		return nil, errors.New("complete employee video scope is required")
	}
	if s.cfg == nil || !s.cfg.VideoGateway.WorkerEnabled || s.gate == nil || !s.gate.Allowed() {
		return nil, ErrVideoRealDispatchDenied
	}
	provider, err := s.repo.GetVideoProvider(ctx, cmd.ProviderAccountID, cmd.Scope.GroupID)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled || (provider.Provider != "seedance" && provider.Provider != HCAtomSeedanceV3Provider) {
		return nil, ErrVideoProviderNotFound
	}
	input := VideoCreateRequest{Model: cmd.Model, Prompt: cmd.Prompt, Content: cmd.Content, Ratio: cmd.Ratio, GenerateAudio: cmd.GenerateAudio, ReturnLastFrame: cmd.ReturnLastFrame, Watermark: cmd.Watermark, Duration: cmd.Duration, Resolution: cmd.Resolution}
	if provider.Provider == "seedance" {
		if err := ValidateTinyRealContract(input); err != nil {
			return nil, err
		}
	} else {
		content, err := normalizeHCAtomV3Content(input)
		if err != nil {
			return nil, err
		}
		input.Content = content
	}
	maximumUSD, err := videoMaximumUSD(s.cfg)
	if err != nil {
		return nil, err
	}
	priceCNYPerMillionCompletionTokens := s.cfg.VideoGateway.SeedanceCNYPerMillionTokens
	usdCNYExchangeRate := s.cfg.VideoGateway.USDCNYExchangeRate
	maximumCNY := s.cfg.VideoGateway.TinyRealMaximumCNY
	task := &VideoTask{APIKeyID: cmd.Scope.APIKeyID, GroupID: cmd.Scope.GroupID, ProviderAccountID: provider.ID,
		Provider: provider.Provider, Model: SeedanceModel, TaskType: "text_to_video", Prompt: strings.TrimSpace(cmd.Prompt), CreateRequest: input,
		Status: VideoStatusQueued, CreationKey: strings.TrimSpace(cmd.CreationKey), CreatedBy: cmd.Scope.UserID,
		DurationSeconds: 4, Resolution: "720p", Currency: "USD", ReservedCostUSD: maximumUSD, ReservationState: VideoReservationReserved,
		PricingSource: VideoPricingSourceConfig, PricingVersion: VideoPricingVersionSeedanceCompletionTokensUSDV1,
		PricingCNYPerMillionCompletionTokens: &priceCNYPerMillionCompletionTokens,
		PricingUSDCNYExchangeRate:            &usdCNYExchangeRate, PricingMaximumCNY: &maximumCNY}
	if provider.Provider == HCAtomSeedanceV3Provider {
		task.Model = HCAtomSeedanceV3PublicModel
		task.DurationSeconds, task.Resolution = cmd.Duration, cmd.Resolution
	}
	if err := s.repo.ReserveAndCreateTask(ctx, task, maximumUSD); err != nil {
		return nil, err
	}
	if err := invalidateVideoCaches(ctx, s.authCache, s.billingCache, task.CreatedBy, task.APIKeyID); err != nil {
		return task, err
	}
	return task, nil
}

func (s *VideoGatewayService) GetTask(ctx context.Context, id int64, scope VideoTaskScope) (*VideoTask, error) {
	return s.repo.GetTaskForScope(ctx, id, scope)
}

func (s *VideoGatewayService) CancelTask(ctx context.Context, id int64, scope VideoTaskScope) (*VideoTask, error) {
	current, err := s.repo.GetTaskForScope(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	if (current.Status == VideoStatusSubmitted || current.Status == VideoStatusRunning) && current.Provider == HCAtomSeedanceV3Provider {
		if s.encryptor == nil || s.clientFactory == nil {
			return nil, ErrVideoCancelConflict
		}
		provider, err := s.repo.GetVideoProvider(ctx, current.ProviderAccountID, current.GroupID)
		if err != nil {
			return nil, err
		}
		key, err := s.encryptor.Decrypt(provider.EncryptedAPIKey)
		if err != nil {
			return nil, errors.New("video provider credential decryption failed")
		}
		cancelled, err := s.clientFactory(provider.Provider, provider.BaseURL, key).Cancel(ctx, current.UpstreamTaskID)
		if err != nil {
			return nil, err
		}
		if cancelled == nil || cancelled.Status != VideoStatusCancelled {
			return nil, ErrVideoCancelConflict
		}
		_, err = s.repo.FinalizeTask(ctx, VideoTaskFinalization{TaskID: current.ID, ExpectedVersion: current.Version, Status: VideoStatusCancelled, ProviderErrorCode: "cancelled", ProviderErrorMessage: "cancelled by employee", ErrorMessage: "cancelled by employee", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
		if err != nil {
			return nil, err
		}
		current.Status = VideoStatusCancelled
		return current, invalidateVideoCaches(ctx, s.authCache, s.billingCache, scope.UserID, scope.APIKeyID)
	}
	task, err := s.repo.CancelTaskForScope(ctx, id, scope)
	if err == nil {
		err = invalidateVideoCaches(ctx, s.authCache, s.billingCache, scope.UserID, scope.APIKeyID)
	}
	return task, err
}

func (s *VideoGatewayService) OpenOwnedLocalAsset(ctx context.Context, id, userID int64) (*VideoLocalAsset, error) {
	if s == nil || s.repo == nil || s.assetStore == nil {
		return nil, ErrVideoLocalAssetNotFound
	}
	task, err := s.repo.GetTaskForOwner(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if !videoTaskHasReadyLocalAsset(task) {
		return nil, ErrVideoLocalAssetNotFound
	}
	return s.assetStore.Open(task.ID, task.LocalAssetPath)
}

func videoTaskHasReadyLocalAsset(task *VideoTask) bool {
	return task != nil && task.Status == VideoStatusSucceeded && strings.TrimSpace(task.ResultURL) != "" &&
		strings.TrimSpace(task.LocalAssetPath) != "" && task.LocalAssetSavedAt != nil
}
