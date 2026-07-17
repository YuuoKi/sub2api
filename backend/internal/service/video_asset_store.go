package service

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	defaultVideoAssetMaxBytes        int64 = 200 << 20
	defaultVideoAssetDownloadTimeout       = 2 * time.Minute
)

var (
	ErrVideoAssetDownload       = errors.New("video asset download failed")
	ErrVideoAssetTooLarge       = errors.New("video asset exceeds retention limit")
	ErrVideoAssetInvalidContent = errors.New("video asset content is invalid")
	ErrVideoLocalAssetNotFound  = errors.New("video local asset not found")
)

type VideoAssetArchive struct {
	RelativePath string
	SavedAt      time.Time
	SizeBytes    int64
}

type VideoLocalAsset struct {
	File        *os.File
	SizeBytes   int64
	ModTime     time.Time
	ContentType string
}

type VideoAssetStore struct {
	client     *http.Client
	filesystem videoAssetFilesystem
	now        func() time.Time
	maxBytes   int64
}

func ProvideVideoAssetStore(cfg *config.Config) *VideoAssetStore {
	root := "./data"
	if cfg != nil && strings.TrimSpace(cfg.Pricing.DataDir) != "" {
		root = cfg.Pricing.DataDir
	}
	return newVideoAssetStore(root, newPublicAssetHTTPClient(defaultVideoAssetDownloadTimeout), time.Now, defaultVideoAssetMaxBytes)
}

func newVideoAssetStore(root string, client *http.Client, now func() time.Time, maxBytes int64) *VideoAssetStore {
	if now == nil {
		now = time.Now
	}
	if maxBytes <= 0 {
		maxBytes = defaultVideoAssetMaxBytes
	}
	return &VideoAssetStore{client: client, filesystem: newPlatformVideoAssetFilesystem(strings.TrimSpace(root)), now: now, maxBytes: maxBytes}
}

func (s *VideoAssetStore) Archive(ctx context.Context, taskID int64, rawURL string) (VideoAssetArchive, error) {
	if s == nil || s.client == nil || s.filesystem == nil || !s.filesystem.Supported() || taskID <= 0 {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || validateAssetSourceURL(parsed) != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	response, err := s.client.Do(request)
	if err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || strings.TrimSpace(response.Header.Get("Content-Range")) != "" {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	mediaType, _, mediaErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if mediaErr != nil || !strings.EqualFold(mediaType, "video/mp4") {
		return VideoAssetArchive{}, ErrVideoAssetInvalidContent
	}
	if response.ContentLength > s.maxBytes {
		return VideoAssetArchive{}, ErrVideoAssetTooLarge
	}

	temporary, err := s.filesystem.Create(taskID)
	if err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	defer temporary.Abort()

	written, copyErr := io.Copy(temporary.File, io.LimitReader(response.Body, s.maxBytes+1))
	if copyErr != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	if written > s.maxBytes {
		return VideoAssetArchive{}, ErrVideoAssetTooLarge
	}
	if written == 0 {
		return VideoAssetArchive{}, ErrVideoAssetInvalidContent
	}
	if _, err = temporary.File.Seek(0, io.SeekStart); err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	probe := make([]byte, 512)
	probeSize, probeErr := temporary.File.Read(probe)
	if probeErr != nil && !errors.Is(probeErr, io.EOF) {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	if !isMP4Content(probe[:probeSize]) {
		return VideoAssetArchive{}, ErrVideoAssetInvalidContent
	}
	if err = temporary.File.Sync(); err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	if err = temporary.File.Close(); err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	if err = temporary.Commit(); err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	savedAt := s.now().UTC()
	return VideoAssetArchive{
		RelativePath: path.Join("assets", "video", strconv.FormatInt(taskID, 10), "result.mp4"),
		SavedAt:      savedAt,
		SizeBytes:    written,
	}, nil
}

func (s *VideoAssetStore) Open(taskID int64, storedPath string) (*VideoLocalAsset, error) {
	if s == nil || taskID <= 0 || storedPath == "" || strings.ContainsAny(storedPath, "%\\\x00") || filepath.IsAbs(storedPath) || path.IsAbs(storedPath) {
		return nil, ErrVideoLocalAssetNotFound
	}
	expected := path.Join("assets", "video", strconv.FormatInt(taskID, 10), "result.mp4")
	if storedPath != expected || path.Clean(storedPath) != expected {
		return nil, ErrVideoLocalAssetNotFound
	}
	if s.filesystem == nil || !s.filesystem.Supported() {
		return nil, ErrVideoLocalAssetNotFound
	}
	file, err := s.filesystem.Open(taskID)
	if err != nil {
		return nil, ErrVideoLocalAssetNotFound
	}
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() {
		_ = file.Close()
		return nil, ErrVideoLocalAssetNotFound
	}
	return &VideoLocalAsset{File: file, SizeBytes: openedInfo.Size(), ModTime: openedInfo.ModTime(), ContentType: "video/mp4"}, nil
}

func isMP4Content(probe []byte) bool {
	return len(probe) >= 12 && string(probe[4:8]) == "ftyp"
}
