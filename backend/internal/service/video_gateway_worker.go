package service

import (
	"context"
	"errors"
	"time"
)

type VideoGatewayWorker struct {
	repo          VideoGatewayRuntimeRepository
	encryptor     VideoKeyEncryptor
	clientFactory func(string, string) *SeedanceAdapter
}

func NewVideoGatewayWorker(repo VideoGatewayRuntimeRepository, encryptor VideoKeyEncryptor, factory func(string, string) *SeedanceAdapter) *VideoGatewayWorker {
	return &VideoGatewayWorker{repo: repo, encryptor: encryptor, clientFactory: factory}
}

func (w *VideoGatewayWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.repo == nil || w.encryptor == nil || w.clientFactory == nil {
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
	provider, err := w.repo.GetVideoProvider(ctx, task.ProviderAccountID)
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
			return nil
		}
		_, err = w.repo.FinalizeTask(ctx, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version, Status: polled.Status,
			ResultURL: polled.ResultURL, LastFrameURL: polled.LastFrameURL, UsageTotalTokens: polled.UsageTotalTokens,
			Currency: "USD", CompletedAt: time.Now().UTC()})
		return err
	}
	started, err := w.repo.BeginRealDispatch(ctx, task.ID, task.Version)
	if err != nil {
		return err
	}
	if !started {
		return errors.New("video task real dispatch was already attempted")
	}
	created, err := w.clientFactory(provider.BaseURL, key).Create(ctx, VideoCreateRequest{Prompt: task.Prompt, Duration: task.DurationSeconds, Resolution: task.Resolution})
	if err != nil {
		_, finalizeErr := w.repo.FinalizeTask(ctx, VideoTaskFinalization{TaskID: task.ID, ExpectedVersion: task.Version + 1, Status: VideoStatusFailed,
			ErrorMessage: RedactVideoSecrets(err.Error(), key), ProviderErrorMessage: RedactVideoSecrets(err.Error(), key), CompletedAt: time.Now().UTC()})
		if finalizeErr != nil {
			return finalizeErr
		}
		return nil
	}
	return w.repo.MarkVideoSubmitted(ctx, task.ID, task.Version+1, created.UpstreamTaskID)
}
