//go:build linux

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLinuxVideoAssetOpenStaysBoundWhenDataRootPathIsReplaced(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "data")
	originalRoot := filepath.Join(parent, "data-original")
	attackerRoot := filepath.Join(parent, "attacker")
	writeLinuxRaceAsset(t, root, 42, []byte("trusted-original"))
	writeLinuxRaceAsset(t, attackerRoot, 42, []byte("attacker-content"))
	filesystem := newLinuxVideoAssetFilesystem(root, func() {
		require.NoError(t, os.Rename(root, originalRoot))
		require.NoError(t, os.Rename(attackerRoot, root))
	})
	store := &VideoAssetStore{filesystem: filesystem, now: time.Now, maxBytes: 1024}

	asset, err := store.Open(42, "assets/video/42/result.mp4")
	require.NoError(t, err)
	t.Cleanup(func() { _ = asset.File.Close() })
	got, err := io.ReadAll(asset.File)
	require.NoError(t, err)
	require.Equal(t, []byte("trusted-original"), got, "open handle must remain bound to the root inode opened before replacement")
}

func TestLinuxVideoAssetArchiveStaysBoundWhenDataRootPathIsReplaced(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "data")
	originalRoot := filepath.Join(parent, "data-original")
	attackerRoot := filepath.Join(parent, "attacker")
	require.NoError(t, os.Mkdir(root, 0o750))
	require.NoError(t, os.Mkdir(attackerRoot, 0o750))
	payload := testMP4Payload("trusted")
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"video/mp4"}}, ContentLength: int64(len(payload)), Body: io.NopCloser(bytes.NewReader(payload)), Request: request}, nil
	})}
	filesystem := newLinuxVideoAssetFilesystem(root, func() {
		require.NoError(t, os.Rename(root, originalRoot))
		require.NoError(t, os.Rename(attackerRoot, root))
	})
	store := &VideoAssetStore{client: client, filesystem: filesystem, now: time.Now, maxBytes: 1024}

	_, err := store.Archive(context.Background(), 42, "https://assets.example.test/result.mp4")
	require.NoError(t, err)
	written, err := os.ReadFile(filepath.Join(originalRoot, "assets", "video", "42", "result.mp4"))
	require.NoError(t, err)
	require.Equal(t, payload, written)
	_, err = os.Stat(filepath.Join(root, "assets", "video", "42", "result.mp4"))
	require.ErrorIs(t, err, os.ErrNotExist, "replacement root must never receive the archive")
}

func TestLinuxVideoAssetOpenRejectsSymlinkComponents(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "outside.mp4")
	require.NoError(t, os.WriteFile(target, testMP4Payload("outside"), 0o640))
	link := filepath.Join(root, "assets", "video", "42", "result.mp4")
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o750))
	require.NoError(t, os.Symlink(target, link))
	store := &VideoAssetStore{filesystem: newLinuxVideoAssetFilesystem(root, nil), now: time.Now, maxBytes: 1024}

	_, err := store.Open(42, "assets/video/42/result.mp4")
	require.ErrorIs(t, err, ErrVideoLocalAssetNotFound)
}

func writeLinuxRaceAsset(t *testing.T, root string, taskID int64, payload []byte) {
	t.Helper()
	target := filepath.Join(root, "assets", "video", strconv.FormatInt(taskID, 10), "result.mp4")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o750))
	require.NoError(t, os.WriteFile(target, payload, 0o640))
}
