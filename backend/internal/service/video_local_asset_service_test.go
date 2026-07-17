package service

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVideoGatewayServiceOpensReadyAssetOnlyForPersistedOwner(t *testing.T) {
	root := t.TempDir()
	payload := testMP4Payload("owned")
	target := filepath.Join(root, "assets", "video", "42", "result.mp4")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o750))
	require.NoError(t, os.WriteFile(target, payload, 0o640))
	savedAt := time.Now().UTC()
	repo := &workerRepoStub{task: &VideoTask{ID: 42, CreatedBy: 9, Status: VideoStatusSucceeded, ResultURL: "https://assets.example.test/result.mp4", LocalAssetPath: "assets/video/42/result.mp4", LocalAssetSavedAt: &savedAt}}
	store := newTestVideoAssetStore(root, nil, time.Now, 1024)
	svc := NewVideoGatewayService(repo, nil, nil, nil, nil, store)

	asset, err := svc.OpenOwnedLocalAsset(context.Background(), 42, 9)
	require.NoError(t, err)
	t.Cleanup(func() { _ = asset.File.Close() })
	got, err := io.ReadAll(asset.File)
	require.NoError(t, err)
	require.Equal(t, payload, got)

	_, err = svc.OpenOwnedLocalAsset(context.Background(), 42, 10)
	require.ErrorIs(t, err, ErrVideoTaskForbidden)

	repo.task.Status = VideoStatusRunning
	_, err = svc.OpenOwnedLocalAsset(context.Background(), 42, 9)
	require.ErrorIs(t, err, ErrVideoLocalAssetNotFound)
}

func TestVideoAdminServiceOpensReadyLocalAssetAndRejectsMissing(t *testing.T) {
	root := t.TempDir()
	payload := testMP4Payload("admin")
	target := filepath.Join(root, "assets", "video", "51", "result.mp4")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o750))
	require.NoError(t, os.WriteFile(target, payload, 0o640))
	savedAt := time.Now().UTC()
	repo := &fakeVideoAdminRepo{task: &VideoTask{ID: 51, Status: VideoStatusSucceeded, ResultURL: "https://assets.example.test/result.mp4", LocalAssetPath: "assets/video/51/result.mp4", LocalAssetSavedAt: &savedAt}}
	store := newTestVideoAssetStore(root, nil, time.Now, 1024)
	svc := NewVideoAdminService(repo, nil, store)

	asset, err := svc.OpenLocalAsset(context.Background(), 51)
	require.NoError(t, err)
	require.NoError(t, asset.File.Close())

	repo.task.LocalAssetPath = ""
	_, err = svc.OpenLocalAsset(context.Background(), 51)
	require.ErrorIs(t, err, ErrVideoLocalAssetNotFound)
}
