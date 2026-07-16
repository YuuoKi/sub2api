package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type VideoGatewayWorker struct {
	repo          VideoGatewayRuntimeRepository
	encryptor     VideoKeyEncryptor
	clientFactory func(string, string) *SeedanceAdapter
	billing       UsageBillingRepository
	cfg           *config.Config
}

func NewVideoGatewayWorker(repo VideoGatewayRuntimeRepository, encryptor VideoKeyEncryptor, factory func(string, string) *SeedanceAdapter, billing UsageBillingRepository, cfg *config.Config) *VideoGatewayWorker {
	return &VideoGatewayWorker{repo: repo, encryptor: encryptor, clientFactory: factory, billing: billing, cfg: cfg}
}

func (w *VideoGatewayWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.repo == nil || w.encryptor == nil || w.clientFactory == nil || w.billing == nil || w.cfg == nil {
		return errors.New("video worker dependencies are required")
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
				_, err = w.repo.FinalizeTask(ctx, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
					ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, ProviderErrorCode: "billing_usage_missing",
					ProviderErrorMessage: "provider success omitted billable completion tokens", ErrorMessage: "provider success omitted billable completion tokens", Currency: "USD", CompletedAt: time.Now().UTC()})
				return err
			}
			actualUSD, costErr := videoActualUSD(*polled.CompletionTokens, w.cfg)
			if costErr != nil {
				return costErr
			}
			cmd := &UsageBillingCommand{RequestID: fmt.Sprintf("video:%d", task.ID), APIKeyID: task.APIKeyID, UserID: task.CreatedBy,
				AccountID: 0, AccountType: AccountTypeAPIKey, Model: task.Model, BillingType: BillingTypeBalance,
				OutputTokens: int(*polled.CompletionTokens), MediaType: "video", BalanceCost: actualUSD, APIKeyQuotaCost: actualUSD,
				APIKeyRateLimitCost: actualUSD, AccountQuotaCost: 0,
				RequestPayloadHash: HashUsageRequestPayload([]byte(fmt.Sprintf("%d|%d|%d|%s|%d|%d|%s", task.ID, task.CreatedBy, task.APIKeyID, task.Model, *polled.CompletionTokens, task.DurationSeconds, task.Resolution)))}
			if _, billErr := w.billing.Apply(ctx, cmd); billErr != nil {
				return fmt.Errorf("video billing failed: %w", billErr)
			}
			_, err = w.repo.FinalizeTask(ctx, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: polled.Status,
				ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.CompletionTokens,
				CostAmount: actualUSD, Currency: "USD", CompletedAt: time.Now().UTC()})
			return err
		}
		_, err = w.repo.FinalizeTask(ctx, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: polled.Status,
			ProviderErrorCode: polled.ErrorCode, ProviderErrorMessage: polled.ErrorMessage, ErrorMessage: polled.ErrorMessage,
			Currency: "USD", CompletedAt: time.Now().UTC()})
		return err
	}
	started, err := w.repo.BeginRealDispatch(ctx, task.ID, task.Version)
	if err != nil {
		return err
	}
	if !started {
		_, finalizeErr := w.repo.FinalizeTask(ctx, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: VideoStatusFailed,
			ProviderErrorCode: "gate_consumed", ProviderErrorMessage: "single smoke authorization already consumed", ErrorMessage: "single smoke authorization already consumed", CompletedAt: time.Now().UTC()})
		return finalizeErr
	}
	created, err := w.clientFactory(provider.BaseURL, key).Create(ctx, VideoCreateRequest{Prompt: task.Prompt, Duration: task.DurationSeconds, Resolution: task.Resolution, ReturnLastFrame: true})
	if err != nil {
		_, finalizeErr := w.repo.FinalizeTask(ctx, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version + 1, Status: VideoStatusFailed,
			ErrorMessage: "upstream provider dispatch failed", ProviderErrorCode: "provider_dispatch_failed", ProviderErrorMessage: "upstream provider dispatch failed", CompletedAt: time.Now().UTC()})
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

func videoActualUSD(completionTokens int64, cfg *config.Config) (float64, error) {
	if cfg == nil || completionTokens <= 0 || cfg.VideoGateway.SeedanceCNYPerMillionTokens <= 0 || cfg.VideoGateway.USDCNYExchangeRate <= 0 {
		return 0, errors.New("video pricing configuration is incomplete")
	}
	cny := float64(completionTokens) * cfg.VideoGateway.SeedanceCNYPerMillionTokens / 1_000_000
	if cny <= 0 {
		return 0, errors.New("video actual cost is invalid")
	}
	return math.Round((cny/cfg.VideoGateway.USDCNYExchangeRate)*1e8) / 1e8, nil
}
