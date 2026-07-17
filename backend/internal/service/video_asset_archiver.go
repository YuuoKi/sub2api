package service

import "context"

type VideoAssetArchiver interface {
	Archive(context.Context, int64, string) error
}

type videoAssetArchiver struct {
	repo  VideoGatewayRuntimeRepository
	store *VideoAssetStore
}

func ProvideVideoAssetArchiver(repo VideoGatewayRuntimeRepository, store *VideoAssetStore) VideoAssetArchiver {
	return &videoAssetArchiver{repo: repo, store: store}
}

func (a *videoAssetArchiver) Archive(ctx context.Context, taskID int64, resultURL string) error {
	if a == nil || a.repo == nil || a.store == nil {
		return ErrVideoAssetDownload
	}
	archived, err := a.store.Archive(ctx, taskID, resultURL)
	if err != nil {
		return err
	}
	return a.repo.SetTaskLocalAsset(ctx, taskID, archived.RelativePath, archived.SavedAt)
}
