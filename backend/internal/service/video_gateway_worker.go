package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type VideoGatewayWorker struct {
	repo          VideoGatewayRuntimeRepository
	encryptor     VideoKeyEncryptor
	clientFactory func(string, string) *SeedanceAdapter
	authCache     VideoAuthCacheInvalidator
	billingCache  VideoBillingCacheInvalidator
	cfg           *config.Config
	gate          *SingleSmokeAuthorization
}

func NewVideoGatewayWorker(repo VideoGatewayRuntimeRepository, encryptor VideoKeyEncryptor, factory func(string, string) *SeedanceAdapter, authCache VideoAuthCacheInvalidator, billingCache VideoBillingCacheInvalidator, cfg *config.Config, gate *SingleSmokeAuthorization) *VideoGatewayWorker {
	return &VideoGatewayWorker{repo: repo, encryptor: encryptor, clientFactory: factory, authCache: authCache, billingCache: billingCache, cfg: cfg, gate: gate}
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
		polled, pollErr := w.clientFactory(provider.BaseURL, key).Poll(ctx, task.UpstreamTaskID)
		if pollErr != nil {
			return pollErr
		}
		if polled.Status != VideoStatusSucceeded && polled.Status != VideoStatusFailed && polled.Status != VideoStatusCancelled {
			return w.repo.UpdateVideoProgress(ctx, task.ID, task.Version, polled.Status)
		}
		if polled.Status == VideoStatusSucceeded {
			if polled.CompletionTokens == nil || *polled.CompletionTokens <= 0 {
				err = w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, ProviderErrorCode: "billing_usage_missing",
					ProviderErrorMessage: "provider success omitted billable completion tokens", ErrorMessage: "provider success omitted billable completion tokens", Currency: "USD", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
				return err
			}
			actualUSD, costErr := videoActualUSD(*polled.CompletionTokens, w.cfg)
			if costErr != nil {
				err = w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens,
					ProviderErrorCode: "billing_configuration_invalid", ProviderErrorMessage: "video billing configuration is invalid",
					ErrorMessage: "video billing configuration is invalid", Currency: "USD", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
				return err
			}
			if polled.ResultURL == "" {
				err = w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens, ProviderActualCostUSD: actualUSD,
					ProviderErrorCode: "result_asset_missing", ProviderErrorMessage: "provider success omitted video asset",
					ErrorMessage: "provider success omitted video asset", Currency: "USD", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
				return err
			}
			if actualUSD-task.ReservedCostUSD > 0.00000001 {
				err = w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens,
					CostAmount: task.ReservedCostUSD, ProviderActualCostUSD: actualUSD, Currency: "USD", Settlement: VideoSettlementCaptureReserved,
					ProviderErrorCode: "budget_actual_exceeded", ProviderErrorMessage: "provider cost exceeded reserved maximum",
					ErrorMessage: "provider cost exceeded reserved maximum", CompletedAt: time.Now().UTC()})
				return err
			}
			err = w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: polled.Status,
				ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens,
				CostAmount: actualUSD, ProviderActualCostUSD: actualUSD, Currency: "USD", Settlement: VideoSettlementCaptureActual, CompletedAt: time.Now().UTC()})
			return err
		}
		err = w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: polled.Status,
			ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL,
			ProviderErrorCode: polled.ErrorCode, ProviderErrorMessage: polled.ErrorMessage, ErrorMessage: polled.ErrorMessage,
			Currency: "USD", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
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
	created, err := w.clientFactory(provider.BaseURL, key).Create(ctx, VideoCreateRequest{Prompt: task.Prompt, Duration: task.DurationSeconds, Resolution: task.Resolution, ReturnLastFrame: true})
	if err != nil {
		finalizeErr := w.finalize(ctx, task, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version + 1, Status: VideoStatusFailed,
			ErrorMessage: "upstream provider dispatch failed", ProviderErrorCode: "provider_dispatch_failed", ProviderErrorMessage: "upstream provider dispatch failed", Settlement: VideoSettlementRelease, CompletedAt: time.Now().UTC()})
		if finalizeErr != nil {
			return finalizeErr
		}
		return nil
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

func (w *VideoGatewayWorker) finalize(ctx context.Context, task *VideoTask, input VideoTaskFinalization) error {
	_, err := w.repo.FinalizeTask(ctx, input)
	if err == nil {
		err = invalidateVideoCaches(ctx, w.authCache, w.billingCache, task.CreatedBy, task.APIKeyID)
	}
	return err
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
