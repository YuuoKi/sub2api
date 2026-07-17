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
	root     string
	client   *http.Client
	now      func() time.Time
	maxBytes int64
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
	return &VideoAssetStore{root: strings.TrimSpace(root), client: client, now: now, maxBytes: maxBytes}
}

func (s *VideoAssetStore) Archive(ctx context.Context, taskID int64, rawURL string) (VideoAssetArchive, error) {
	if s == nil || s.client == nil || taskID <= 0 {
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
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	mediaType, _, mediaErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if mediaErr != nil || !strings.EqualFold(mediaType, "video/mp4") {
		return VideoAssetArchive{}, ErrVideoAssetInvalidContent
	}
	if response.ContentLength > s.maxBytes {
		return VideoAssetArchive{}, ErrVideoAssetTooLarge
	}

	taskDir, err := s.ensureTaskDir(taskID)
	if err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	temporary, err := os.CreateTemp(taskDir, ".result-*.tmp")
	if err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	temporaryPath := temporary.Name()
	keepTemporary := false
	defer func() {
		_ = temporary.Close()
		if !keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	written, copyErr := io.Copy(temporary, io.LimitReader(response.Body, s.maxBytes+1))
	if copyErr != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	if written > s.maxBytes {
		return VideoAssetArchive{}, ErrVideoAssetTooLarge
	}
	if written == 0 {
		return VideoAssetArchive{}, ErrVideoAssetInvalidContent
	}
	if _, err = temporary.Seek(0, io.SeekStart); err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	probe := make([]byte, 512)
	probeSize, probeErr := temporary.Read(probe)
	if probeErr != nil && !errors.Is(probeErr, io.EOF) {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	if !isMP4Content(probe[:probeSize]) {
		return VideoAssetArchive{}, ErrVideoAssetInvalidContent
	}
	if err = temporary.Sync(); err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	if err = temporary.Close(); err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	target := filepath.Join(taskDir, "result.mp4")
	if err = os.Rename(temporaryPath, target); err != nil {
		return VideoAssetArchive{}, ErrVideoAssetDownload
	}
	keepTemporary = true
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
	root, err := filepath.Abs(s.root)
	if err != nil || root == "" {
		return nil, ErrVideoLocalAssetNotFound
	}
	if err := rejectSymlinkOrNonDirectory(root); err != nil {
		return nil, ErrVideoLocalAssetNotFound
	}
	current := root
	for _, component := range []string{"assets", "video", strconv.FormatInt(taskID, 10)} {
		current = filepath.Join(current, component)
		if err := rejectSymlinkOrNonDirectory(current); err != nil {
			return nil, ErrVideoLocalAssetNotFound
		}
	}
	target := filepath.Join(current, "result.mp4")
	info, err := os.Lstat(target)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, ErrVideoLocalAssetNotFound
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, ErrVideoLocalAssetNotFound
	}
	canonicalTarget, err := filepath.EvalSymlinks(target)
	if err != nil || !pathWithinRoot(canonicalRoot, canonicalTarget) {
		return nil, ErrVideoLocalAssetNotFound
	}
	file, err := os.Open(target)
	if err != nil {
		return nil, ErrVideoLocalAssetNotFound
	}
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		_ = file.Close()
		return nil, ErrVideoLocalAssetNotFound
	}
	return &VideoLocalAsset{File: file, SizeBytes: openedInfo.Size(), ModTime: openedInfo.ModTime(), ContentType: "video/mp4"}, nil
}

func (s *VideoAssetStore) ensureTaskDir(taskID int64) (string, error) {
	if s.root == "" {
		return "", ErrVideoAssetDownload
	}
	root, err := filepath.Abs(s.root)
	if err != nil {
		return "", ErrVideoAssetDownload
	}
	if err = ensurePlainDirectory(root, 0o750); err != nil {
		return "", err
	}
	current := root
	for _, component := range []string{"assets", "video", strconv.FormatInt(taskID, 10)} {
		current = filepath.Join(current, component)
		if err = ensurePlainDirectory(current, 0o750); err != nil {
			return "", err
		}
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	canonicalTaskDir, err := filepath.EvalSymlinks(current)
	if err != nil || !pathWithinRoot(canonicalRoot, canonicalTaskDir) {
		return "", ErrVideoAssetDownload
	}
	return current, nil
}

func ensurePlainDirectory(directory string, mode os.FileMode) error {
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(directory, mode); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err = os.Lstat(directory)
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return ErrVideoAssetDownload
	}
	return nil
}

func rejectSymlinkOrNonDirectory(directory string) error {
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return ErrVideoLocalAssetNotFound
	}
	return nil
}

func pathWithinRoot(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func isMP4Content(probe []byte) bool {
	return len(probe) >= 12 && string(probe[4:8]) == "ftyp"
}
