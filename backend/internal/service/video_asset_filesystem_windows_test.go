//go:build windows

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVideoAssetStoreFailsClosedOnWindowsBeforeNetworkOrFilesystemRead(t *testing.T) {
	root := t.TempDir()
	payload := testMP4Payload("windows")
	requestCount := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount++
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"video/mp4"}}, ContentLength: int64(len(payload)), Body: io.NopCloser(bytes.NewReader(payload)), Request: request}, nil
	})}
	store := newVideoAssetStore(root, client, time.Now, 1024)

	_, err := store.Archive(context.Background(), 42, "https://assets.example.test/result.mp4")
	require.ErrorIs(t, err, ErrVideoAssetDownload)
	require.Zero(t, requestCount, "unsupported platform must fail before any outbound request")

	target := filepath.Join(root, "assets", "video", "42", "result.mp4")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o750))
	require.NoError(t, os.WriteFile(target, payload, 0o640))
	_, err = store.Open(42, "assets/video/42/result.mp4")
	require.ErrorIs(t, err, ErrVideoLocalAssetNotFound)
}
