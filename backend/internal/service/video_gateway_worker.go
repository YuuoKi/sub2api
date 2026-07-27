package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

type VideoGatewayWorker struct {
	repo          VideoGatewayRuntimeRepository
	encryptor     VideoKeyEncryptor
	clientFactory func(string, string, string) VideoProviderClient
	authCache     VideoAuthCacheInvalidator
	billingCache  VideoBillingCacheInvalidator
	cfg           *config.Config
	gate          *SingleSmokeAuthorization
	archiver      VideoAssetArchiver
}

func NewVideoGatewayWorker(repo VideoGatewayRuntimeRepository, encryptor VideoKeyEncryptor, factory func(string, string, string) VideoProviderClient, authCache VideoAuthCacheInvalidator, billingCache VideoBillingCacheInvalidator, cfg *config.Config, gate *SingleSmokeAuthorization, archivers ...VideoAssetArchiver) *VideoGatewayWorker {
	var archiver VideoAssetArchiver
	if len(archivers) > 0 {
		archiver = archivers[0]
	}
	return &VideoGatewayWorker{repo: repo, encryptor: encryptor, clientFactory: factory, authCache: authCache, billingCache: billingCache, cfg: cfg, gate: gate, archiver: archiver}
}

func (w *VideoGatewayWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.repo == nil || w.encryptor == nil || w.clientFactory == nil || w.cfg == nil {
		return errors.New("video worker dependencies are required")
	}
	if w.authCache == nil || w.billingCache == nil {
		return errors.New("video worker cache invalidators are required")
	}
	tasks, err := w.repo.ClaimRunnableTasks(ctx, 1, 30*time.Second)
	if err != nil || len(tasks) == 0 {
		return err
	}
	task := tasks[0]
	if task.Status != VideoStatusQueued && task.Status != VideoStatusSubmitted && task.Status != VideoStatusRunning {
		return nil
	}
	provider, err := w.repo.GetVideoProvider(ctx, task.ProviderAccountID, task.GroupID)
	if err != nil {
		return err
	}
	key, err := w.encryptor.Decrypt(provider.EncryptedAPIKey)
	if err != nil {
		return errors.New("video provider credential decryption failed")
	}
	if task.Status != VideoStatusQueued {
		polled, pollErr := w.clientFactory(provider.Provider, provider.BaseURL, key).Poll(ctx, task.UpstreamTaskID)
		if pollErr != nil {
			return pollErr
		}
		if polled.Status != VideoStatusSucceeded && polled.Status != VideoStatusFailed && polled.Status != VideoStatusCancelled {
			return w.repo.UpdateVideoProgress(ctx, task.ID, task.Version, polled.Status)
		}
		if polled.Status == VideoStatusSucceeded {
			if polled.CompletionTokens == nil || *polled.CompletionTokens <= 0 {
				err = w.finalize(ctx, task, videoFinalizationWithUpstreamEvidence(VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, ProviderErrorCode: "billing_usage_missing",
					ProviderErrorMessage: "provider success omitted billable completion tokens", ErrorMessage: "provider success omitted billable completion tokens", Currency: "USD", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()}, polled))
				return err
			}
			actualUSD, costErr := videoActualUSDForTask(*polled.CompletionTokens, task, w.cfg)
			if costErr != nil {
				err = w.finalize(ctx, task, videoFinalizationWithUpstreamEvidence(VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens,
					ProviderErrorCode: "billing_configuration_invalid", ProviderErrorMessage: "video billing configuration is invalid",
					ErrorMessage: "video billing configuration is invalid", Currency: "USD", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()}, polled))
				return err
			}
			if polled.ResultURL == "" {
				err = w.finalize(ctx, task, videoFinalizationWithUpstreamEvidence(VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens, ProviderActualCostUSD: actualUSD,
					ProviderErrorCode: "result_asset_missing", ProviderErrorMessage: "provider success omitted video asset",
					ErrorMessage: "provider success omitted video asset", Currency: "USD", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()}, polled))
				return err
			}
			if actualUSD-task.ReservedCostUSD > 0.00000001 {
				err = w.finalize(ctx, task, videoFinalizationWithUpstreamEvidence(VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens,
					CostAmount: task.ReservedCostUSD, ProviderActualCostUSD: actualUSD, Currency: "USD", Settlement: VideoSettlementCaptureReserved,
					ProviderErrorCode: "budget_actual_exceeded", ProviderErrorMessage: "provider cost exceeded reserved maximum",
					ErrorMessage: "provider cost exceeded reserved maximum", CompletedAt: time.Now().UTC()}, polled))
				return err
			}
			err = w.finalize(ctx, task, videoFinalizationWithUpstreamEvidence(VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: polled.Status,
				ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens,
				CostAmount: actualUSD, ProviderActualCostUSD: actualUSD, Currency: "USD", Settlement: VideoSettlementCaptureActual, CompletedAt: time.Now().UTC()}, polled))
			return err
		}
		err = w.finalize(ctx, task, videoFinalizationWithUpstreamEvidence(VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: polled.Status,
			ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL,
			ProviderErrorCode: polled.ErrorCode, ProviderErrorMessage: polled.ErrorMessage, ErrorMessage: polled.ErrorMessage,
			Currency: "USD", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()}, polled))
		return err
	}
	if w.gate == nil || !w.gate.Allowed() {
		finalizeErr := w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
			ProviderErrorCode: "process_gate_denied", ProviderErrorMessage: "process safety gate denied dispatch", ErrorMessage: "process safety gate denied dispatch", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
		if finalizeErr != nil {
			return finalizeErr
		}
		return ErrVideoRealDispatchDenied
	}
	started, err := w.repo.BeginRealDispatch(ctx, task.ID, task.Version)
	if provider.Provider == HCAtomSeedanceV3Provider {
		if !w.cfg.VideoGateway.HCAtomV3DispatchEnabled {
			started = false
			err = nil
		} else {
			started, err = w.repo.BeginHCAtomV3Dispatch(ctx, task.ID, task.Version)
		}
	}
	if err != nil {
		return err
	}
	if !started {
		finalizeErr := w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
			ProviderErrorCode: "gate_consumed", ProviderErrorMessage: "single smoke authorization already consumed", ErrorMessage: "single smoke authorization already consumed", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
		return finalizeErr
	}
	if err = w.gate.Consume(); err != nil {
		return w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version + 1, Status: VideoStatusFailed,
			ProviderErrorCode: "process_gate_denied", ProviderErrorMessage: "process safety gate denied dispatch", ErrorMessage: "process safety gate denied dispatch", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
	}
	request := task.CreateRequest
	if request.Prompt == "" && len(request.Content) == 0 {
		request = VideoCreateRequest{Prompt: task.Prompt, Duration: task.DurationSeconds, Resolution: task.Resolution, ReturnLastFrame: true}
	}
	created, err := w.clientFactory(provider.Provider, provider.BaseURL, key).Create(ctx, request)
	if err != nil {
		var transportErr *VideoProviderTransportError
		if errors.As(err, &transportErr) {
			return w.repo.MarkVideoDispatchUncertain(ctx, task.ID, task.Version+1, "provider_dispatch_uncertain", transportErr.UpstreamTaskID)
		}
		return w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version + 1, Status: VideoStatusFailed, ErrorMessage: "upstream provider dispatch failed", ProviderErrorCode: "provider_dispatch_failed", ProviderErrorMessage: "upstream provider dispatch failed", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
	}
	for {
		if persistErr := w.repo.MarkVideoSubmitted(ctx, task.ID, task.Version+1, created.UpstreamTaskID); persistErr == nil {
			return nil
		}
		if stored, readErr := w.repo.GetTask(ctx, task.ID); readErr == nil && stored.UpstreamTaskID == created.UpstreamTaskID && stored.DispatchState == "accepted" {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("accepted upstream task id persistence interrupted")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func videoFinalizationWithUpstreamEvidence(input VideoTaskFinalization, task *VideoProviderTask) VideoTaskFinalization {
	if task == nil {
		return input
	}
	input.UpstreamModel = task.UpstreamModel
	input.UpstreamDurationSeconds = task.UpstreamDurationSeconds
	input.UpstreamResolution = task.UpstreamResolution
	return input
}

func (w *VideoGatewayWorker) finalize(ctx context.Context, task *VideoTask, input VideoTaskFinalization) error {
	result, err := w.repo.FinalizeTask(ctx, input)
	if err != nil {
		return err
	}
	cacheErr := invalidateVideoCaches(ctx, w.authCache, w.billingCache, task.CreatedBy, task.APIKeyID)
	if result.Applied && input.Status == VideoStatusSucceeded && input.ResultURL != "" && w.archiver != nil {
		if archiveErr := w.archiver.Archive(ctx, task.ID, input.ResultURL); archiveErr != nil {
			logger.L().Error("video_gateway.local_asset_archive_failed",
				zap.String("component", "service.video_gateway_worker"),
				zap.String("error_code", "local_asset_archive_failed"),
				zap.Int64("video_task_id", task.ID),
				zap.Error(archiveErr))
		}
	}
	return cacheErr
}

func videoActualUSD(completionTokens int64, cfg *config.Config) (float64, error) {
	if cfg == nil || completionTokens <= 0 || cfg.VideoGateway.SeedanceCNYPerMillionTokens <= 0 || cfg.VideoGateway.USDCNYExchangeRate <= 0 {
		return 0, errors.New("video pricing configuration is incomplete")
	}
	cny := float64(completionTokens) * cfg.VideoGateway.SeedanceCNYPerMillionTokens / 1_000_000
	if cny <= 0 {
		return 0, errors.New("video actual cost is invalid")
	}
	actual := math.Round((cny/cfg.VideoGateway.USDCNYExchangeRate)*1e8) / 1e8
	if actual <= 0 {
		return 0, errors.New("video actual cost rounds to zero")
	}
	return actual, nil
}

func videoActualUSDForTask(completionTokens int64, task *VideoTask, legacyConfig *config.Config) (float64, error) {
	if task == nil {
		return 0, errors.New("video pricing task is required")
	}
	hasSnapshot := task.PricingSource != "" || task.PricingVersion != "" ||
		task.PricingCNYPerMillionCompletionTokens != nil || task.PricingUSDCNYExchangeRate != nil || task.PricingMaximumCNY != nil
	if !hasSnapshot {
		// Explicit compatibility path for tasks created before pricing provenance
		// existed. The record remains unknown; current config is not backfilled.
		return videoActualUSD(completionTokens, legacyConfig)
	}
	if task.Currency != "USD" || task.PricingSource != VideoPricingSourceConfig ||
		task.PricingVersion != VideoPricingVersionSeedanceCompletionTokensUSDV1 ||
		task.PricingCNYPerMillionCompletionTokens == nil || *task.PricingCNYPerMillionCompletionTokens <= 0 ||
		task.PricingUSDCNYExchangeRate == nil || *task.PricingUSDCNYExchangeRate <= 0 ||
		task.PricingMaximumCNY == nil || *task.PricingMaximumCNY <= 0 {
		return 0, errors.New("video pricing snapshot is incomplete or unsupported")
	}
	snapshotConfig := &config.Config{VideoGateway: config.VideoGatewayConfig{
		SeedanceCNYPerMillionTokens: *task.PricingCNYPerMillionCompletionTokens,
		USDCNYExchangeRate:          *task.PricingUSDCNYExchangeRate,
	}}
	return videoActualUSD(completionTokens, snapshotConfig)
}
