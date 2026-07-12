//go:build realsmoke

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type realReviewKind string

const (
	realReviewImage realReviewKind = "image"
	realReviewVideo realReviewKind = "video"

	realReviewMaxImages = 4
	realReviewMaxVideos = 4
	realReviewMaxCNY    = 60.0
)

type realReviewSessionState struct {
	ImageAttempts int     `json:"image_attempts"`
	VideoAttempts int     `json:"video_attempts"`
	ReservedCNY   float64 `json:"reserved_cny"`
}

type realReviewSessionGuard struct {
	statePath string
	enabled   bool
}

func newRealReviewSessionGuard(statePath string, enabled bool) *realReviewSessionGuard {
	return &realReviewSessionGuard{statePath: strings.TrimSpace(statePath), enabled: enabled}
}

func (g *realReviewSessionGuard) ReserveBefore(kind realReviewKind, estimatedCNY float64, call func()) error {
	if err := g.Reserve(kind, estimatedCNY); err != nil {
		return err
	}
	call()
	return nil
}

func (g *realReviewSessionGuard) Reserve(kind realReviewKind, estimatedCNY float64) error {
	if g == nil || !g.enabled {
		return errors.New("REAL_REVIEW_SESSION_DISABLED: explicit review-session guard enablement is required")
	}
	if g.statePath == "" || !filepath.IsAbs(g.statePath) {
		return errors.New("REAL_REVIEW_STATE_PATH_REQUIRED: an explicit absolute state path is required")
	}
	if estimatedCNY <= 0 || math.IsNaN(estimatedCNY) || math.IsInf(estimatedCNY, 0) {
		return errors.New("REAL_REVIEW_INVALID_COST: estimated CNY must be positive and finite")
	}
	if err := os.MkdirAll(filepath.Dir(g.statePath), 0700); err != nil {
		return fmt.Errorf("REAL_REVIEW_STATE_DIRECTORY_FAILED: %w", err)
	}

	lockPath := g.statePath + ".lock"
	lock, err := acquireRealReviewLock(lockPath, 2*time.Second)
	if err != nil {
		return fmt.Errorf("REAL_REVIEW_LOCK_FAILED: %w", err)
	}
	defer func() {
		_ = lock.Close()
		_ = os.Remove(lockPath)
	}()

	state, err := g.loadState()
	if err != nil {
		return fmt.Errorf("REAL_REVIEW_STATE_INVALID: %w", err)
	}
	next := state
	switch kind {
	case realReviewImage:
		if next.ImageAttempts >= realReviewMaxImages {
			return errors.New("REAL_REVIEW_IMAGE_LIMIT: maximum 4 image attempts reached")
		}
		next.ImageAttempts++
	case realReviewVideo:
		if next.VideoAttempts >= realReviewMaxVideos {
			return errors.New("REAL_REVIEW_VIDEO_LIMIT: maximum 4 video attempts reached")
		}
		next.VideoAttempts++
	default:
		return errors.New("REAL_REVIEW_KIND_INVALID: kind must be image or video")
	}
	if next.ReservedCNY+estimatedCNY > realReviewMaxCNY {
		return errors.New("REAL_REVIEW_BUDGET_LIMIT: cumulative reserved CNY would exceed 60")
	}
	next.ReservedCNY += estimatedCNY
	return g.storeState(next)
}

func acquireRealReviewLock(path string, timeout time.Duration) (*os.File, error) {
	deadline := time.Now().Add(timeout)
	for {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			payload := fmt.Sprintf("pid=%d\ncreated_at_utc=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
			if _, writeErr := file.WriteString(payload); writeErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				return nil, writeErr
			}
			if syncErr := file.Sync(); syncErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				return nil, syncErr
			}
			return file, nil
		}
		if !errors.Is(err, os.ErrExist) || time.Now().After(deadline) {
			if errors.Is(err, os.ErrExist) {
				return nil, fmt.Errorf("review lock %q already exists; fail-closed: manually audit the state file and recorded PID/process before removing the lock: %w", path, err)
			}
			return nil, err
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (g *realReviewSessionGuard) loadState() (realReviewSessionState, error) {
	data, err := os.ReadFile(g.statePath)
	if errors.Is(err, os.ErrNotExist) {
		return realReviewSessionState{}, nil
	}
	if err != nil {
		return realReviewSessionState{}, err
	}
	var state realReviewSessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return realReviewSessionState{}, err
	}
	if state.ImageAttempts < 0 || state.ImageAttempts > realReviewMaxImages ||
		state.VideoAttempts < 0 || state.VideoAttempts > realReviewMaxVideos ||
		state.ReservedCNY < 0 || state.ReservedCNY > realReviewMaxCNY ||
		math.IsNaN(state.ReservedCNY) || math.IsInf(state.ReservedCNY, 0) {
		return realReviewSessionState{}, errors.New("state contains invalid counters")
	}
	return state, nil
}

func (g *realReviewSessionGuard) storeState(state realReviewSessionState) error {
	if err := os.MkdirAll(filepath.Dir(g.statePath), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(g.statePath), ".real-review-state-*.tmp")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	committed := false
	defer func() {
		if !committed {
			_ = file.Close()
			_ = os.Remove(tempPath)
		}
	}()
	if err := file.Chmod(0600); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, g.statePath); err != nil {
		return err
	}
	committed = true
	return nil
}

func (g *realReviewSessionGuard) loadStateForTest() (realReviewSessionState, error) {
	return g.loadState()
}
