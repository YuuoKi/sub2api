package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVideoAssetStoreArchiveWritesVerifiedMP4Atomically(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 7, 17, 8, 9, 10, 0, time.UTC)
	payload := testMP4Payload("video-result")
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, request.Method)
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        http.Header{"Content-Type": []string{"video/mp4"}},
			ContentLength: int64(len(payload)),
			Body:          io.NopCloser(bytes.NewReader(payload)),
			Request:       request,
		}, nil
	})}
	store := newVideoAssetStore(root, client, func() time.Time { return now }, 1024)

	archived, err := store.Archive(context.Background(), 42, "https://assets.example.test/result.mp4")
	require.NoError(t, err)
	require.Equal(t, "assets/video/42/result.mp4", archived.RelativePath)
	require.Equal(t, now, archived.SavedAt)
	require.Equal(t, int64(len(payload)), archived.SizeBytes)
	written, err := os.ReadFile(filepath.Join(root, "assets", "video", "42", "result.mp4"))
	require.NoError(t, err)
	require.Equal(t, payload, written)
	matches, err := filepath.Glob(filepath.Join(root, "assets", "video", "42", ".result-*.tmp"))
	require.NoError(t, err)
	require.Empty(t, matches)
}

func TestVideoAssetStoreArchiveRejectsUntrustedOrInvalidResponses(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		contentType   string
		contentLength int64
		body          []byte
		maxBytes      int64
		wantErr       error
	}{
		{name: "http failure", status: http.StatusBadGateway, contentType: "video/mp4", body: testMP4Payload("bad"), maxBytes: 1024, wantErr: ErrVideoAssetDownload},
		{name: "declared too large", status: http.StatusOK, contentType: "video/mp4", contentLength: 2048, body: testMP4Payload("large"), maxBytes: 1024, wantErr: ErrVideoAssetTooLarge},
		{name: "stream exceeds limit", status: http.StatusOK, contentType: "video/mp4", contentLength: -1, body: testMP4Payload(strings.Repeat("x", 1024)), maxBytes: 128, wantErr: ErrVideoAssetTooLarge},
		{name: "html disguised as video", status: http.StatusOK, contentType: "video/mp4", body: []byte("<!doctype html><title>not video</title>"), maxBytes: 1024, wantErr: ErrVideoAssetInvalidContent},
		{name: "wrong declared mime", status: http.StatusOK, contentType: "text/plain", body: testMP4Payload("video"), maxBytes: 1024, wantErr: ErrVideoAssetInvalidContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				length := tt.contentLength
				if length == 0 {
					length = int64(len(tt.body))
				}
				return &http.Response{StatusCode: tt.status, Header: http.Header{"Content-Type": []string{tt.contentType}}, ContentLength: length, Body: io.NopCloser(bytes.NewReader(tt.body)), Request: request}, nil
			})}
			store := newVideoAssetStore(root, client, time.Now, tt.maxBytes)

			_, err := store.Archive(context.Background(), 7, "https://assets.example.test/result.mp4")
			require.ErrorIs(t, err, tt.wantErr)
			_, statErr := os.Stat(filepath.Join(root, "assets", "video", "7", "result.mp4"))
			require.ErrorIs(t, statErr, os.ErrNotExist)
		})
	}
}

func TestPublicAssetHTTPClientRejectsUnsafeRedirects(t *testing.T) {
	client := newPublicAssetHTTPClient(5 * time.Second)
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/private.mp4", nil)
	require.NoError(t, err)
	require.Error(t, client.CheckRedirect(request, nil))

	request, err = http.NewRequest(http.MethodGet, "https://user:secret@assets.example.test/result.mp4", nil)
	require.NoError(t, err)
	require.Error(t, client.CheckRedirect(request, nil))
}

func TestVideoAssetStoreOpenAcceptsOnlyCanonicalRegularTaskFile(t *testing.T) {
	root := t.TempDir()
	validPath := filepath.Join(root, "assets", "video", "42", "result.mp4")
	require.NoError(t, os.MkdirAll(filepath.Dir(validPath), 0o750))
	payload := testMP4Payload("stored")
	require.NoError(t, os.WriteFile(validPath, payload, 0o640))
	store := newVideoAssetStore(root, nil, time.Now, 1024)

	asset, err := store.Open(42, "assets/video/42/result.mp4")
	require.NoError(t, err)
	t.Cleanup(func() { _ = asset.File.Close() })
	require.Equal(t, int64(len(payload)), asset.SizeBytes)
	require.Equal(t, "video/mp4", asset.ContentType)
	read, err := io.ReadAll(asset.File)
	require.NoError(t, err)
	require.Equal(t, payload, read)

	invalid := []string{
		"", "../42/result.mp4", "assets/video/../42/result.mp4", "assets/video/%2e%2e/42/result.mp4",
		"assets\\video\\42\\result.mp4", "assets/video/42/%72esult.mp4", "assets/video/42/result.mp4\x00",
		"assets/video/41/result.mp4", "assets/video/42/other.mp4", validPath,
	}
	for _, storedPath := range invalid {
		t.Run(storedPath, func(t *testing.T) {
			_, err := store.Open(42, storedPath)
			require.ErrorIs(t, err, ErrVideoLocalAssetNotFound)
		})
	}

	directoryPath := filepath.Join(root, "assets", "video", "43", "result.mp4")
	require.NoError(t, os.MkdirAll(directoryPath, 0o750))
	_, err = store.Open(43, "assets/video/43/result.mp4")
	require.ErrorIs(t, err, ErrVideoLocalAssetNotFound)
}

func TestVideoAssetStoreOpenRejectsSymlinkEscapeOnWindows(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "outside.mp4")
	require.NoError(t, os.WriteFile(target, testMP4Payload("outside"), 0o640))
	link := filepath.Join(root, "assets", "video", "42", "result.mp4")
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o750))
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Windows account cannot create symlink: %v", err)
	}
	store := newVideoAssetStore(root, nil, time.Now, 1024)

	_, err := store.Open(42, "assets/video/42/result.mp4")
	require.ErrorIs(t, err, ErrVideoLocalAssetNotFound)
}

func testMP4Payload(suffix string) []byte {
	return append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 2, 0, 'i', 's', 'o', 'm', 'i', 's', 'o', '2'}, []byte(suffix)...)
}
