package reviewguard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type RealCreateKind string

const (
	RealCreateImage RealCreateKind = "image"
	RealCreateVideo RealCreateKind = "video"

	MaxImages = 4
	MaxVideos = 4
)

var MaxReservedCNY = decimal.NewFromInt(60)

type RealCreateReservation struct {
	OperationID    string
	Kind           RealCreateKind
	ReservedCNY    decimal.Decimal
	PricingSource  string
	PricingVersion string
}

type RealCreateSnapshot struct {
	ImageUsed      int
	ImageRemaining int
	VideoUsed      int
	VideoRemaining int
	ReservedCNY    decimal.Decimal
	RemainingCNY   decimal.Decimal
	PricingVersion string
}

type RealCreateGuard interface {
	Reserve(ctx context.Context, reservation RealCreateReservation) error
	Snapshot(ctx context.Context) (RealCreateSnapshot, error)
}

type SessionGuard struct {
	statePath string
}

type FailClosedGuard struct{}

func NewSessionGuard(statePath string) *SessionGuard {
	return &SessionGuard{statePath: strings.TrimSpace(statePath)}
}

func NewFailClosedGuard() *FailClosedGuard {
	return &FailClosedGuard{}
}

func (g *FailClosedGuard) Reserve(context.Context, RealCreateReservation) error {
	return errors.New("REAL_REVIEW_SESSION_DISABLED: explicit review-session guard enablement is required")
}

func (g *FailClosedGuard) Snapshot(context.Context) (RealCreateSnapshot, error) {
	return RealCreateSnapshot{
		ImageRemaining: 0,
		VideoRemaining: 0,
		ReservedCNY:    decimal.Zero,
		RemainingCNY:   decimal.Zero,
	}, nil
}

func (g *SessionGuard) Reserve(ctx context.Context, reservation RealCreateReservation) error {
	_ = ctx
	if g == nil {
		return errors.New("REAL_REVIEW_SESSION_DISABLED: explicit review-session guard enablement is required")
	}
	if g.statePath == "" || !filepath.IsAbs(g.statePath) {
		return errors.New("REAL_REVIEW_STATE_PATH_REQUIRED: an explicit absolute state path is required")
	}
	opID := strings.TrimSpace(reservation.OperationID)
	if opID == "" {
		return errors.New("REAL_REVIEW_OPERATION_ID_REQUIRED: operation id is required for idempotent real creates")
	}
	if reservation.Kind != RealCreateImage && reservation.Kind != RealCreateVideo {
		return errors.New("REAL_REVIEW_KIND_INVALID: kind must be image or video")
	}
	if !reservation.ReservedCNY.IsPositive() {
		return errors.New("REAL_REVIEW_INVALID_COST: estimated CNY must be positive and finite")
	}
	if err := os.MkdirAll(filepath.Dir(g.statePath), 0700); err != nil {
		return fmt.Errorf("REAL_REVIEW_STATE_DIRECTORY_FAILED: %w", err)
	}

	lockPath := g.statePath + ".lock"
	lock, err := acquireLock(lockPath, 2*time.Second)
	if err != nil {
		return fmt.Errorf("REAL_REVIEW_LOCK_FAILED: %w", err)
	}
	defer func() {
		_ = lock.Close()
		_ = os.Remove(lockPath)
	}()

	state, err := g.loadStateLocked()
	if err != nil {
		return fmt.Errorf("REAL_REVIEW_STATE_INVALID: %w", err)
	}
	if existing, ok := state.Reservations[opID]; ok {
		if !reservationFingerprintEqual(existing, reservation) {
			return errors.New("REAL_REVIEW_IDEMPOTENCY_MISMATCH: operation id reused with different reservation params")
		}
		return nil
	}

	next := state
	if next.Reservations == nil {
		next.Reservations = map[string]persistedReservation{}
	}
	switch reservation.Kind {
	case RealCreateImage:
		if next.ImageAttempts >= MaxImages {
			return errors.New("REAL_REVIEW_IMAGE_LIMIT: maximum 4 image attempts reached")
		}
		next.ImageAttempts++
	case RealCreateVideo:
		if next.VideoAttempts >= MaxVideos {
			return errors.New("REAL_REVIEW_VIDEO_LIMIT: maximum 4 video attempts reached")
		}
		next.VideoAttempts++
	}
	nextCNY := next.ReservedCNY.Add(reservation.ReservedCNY)
	if nextCNY.GreaterThan(MaxReservedCNY) {
		return errors.New("REAL_REVIEW_BUDGET_LIMIT: cumulative reserved CNY would exceed 60")
	}
	next.ReservedCNY = nextCNY
	if ver := strings.TrimSpace(reservation.PricingVersion); ver != "" {
		next.PricingVersion = ver
	}
	next.Reservations[opID] = persistedReservation{
		Kind:           string(reservation.Kind),
		ReservedCNY:    decimalString{Decimal: reservation.ReservedCNY},
		PricingSource:  strings.TrimSpace(reservation.PricingSource),
		PricingVersion: strings.TrimSpace(reservation.PricingVersion),
	}
	return g.storeState(next)
}

func (g *SessionGuard) Snapshot(ctx context.Context) (RealCreateSnapshot, error) {
	_ = ctx
	if g == nil {
		return RealCreateSnapshot{}, errors.New("REAL_REVIEW_SESSION_DISABLED: explicit review-session guard enablement is required")
	}
	if g.statePath == "" || !filepath.IsAbs(g.statePath) {
		return RealCreateSnapshot{}, errors.New("REAL_REVIEW_STATE_PATH_REQUIRED: an explicit absolute state path is required")
	}
	lockPath := g.statePath + ".lock"
	lock, err := acquireLock(lockPath, 2*time.Second)
	if err != nil {
		return RealCreateSnapshot{}, fmt.Errorf("REAL_REVIEW_LOCK_FAILED: %w", err)
	}
	defer func() {
		_ = lock.Close()
		_ = os.Remove(lockPath)
	}()
	state, err := g.loadStateLocked()
	if err != nil {
		return RealCreateSnapshot{}, fmt.Errorf("REAL_REVIEW_STATE_INVALID: %w", err)
	}
	return snapshotFromState(state), nil
}

func reservationFingerprintEqual(existing persistedReservation, reservation RealCreateReservation) bool {
	return existing.Kind == string(reservation.Kind) &&
		existing.ReservedCNY.Decimal.Equal(reservation.ReservedCNY) &&
		existing.PricingSource == strings.TrimSpace(reservation.PricingSource) &&
		existing.PricingVersion == strings.TrimSpace(reservation.PricingVersion)
}

func snapshotFromState(state persistedState) RealCreateSnapshot {
	remainingCNY := MaxReservedCNY.Sub(state.ReservedCNY)
	if remainingCNY.IsNegative() {
		remainingCNY = decimal.Zero
	}
	imageRemaining := MaxImages - state.ImageAttempts
	if imageRemaining < 0 {
		imageRemaining = 0
	}
	videoRemaining := MaxVideos - state.VideoAttempts
	if videoRemaining < 0 {
		videoRemaining = 0
	}
	return RealCreateSnapshot{
		ImageUsed:      state.ImageAttempts,
		ImageRemaining: imageRemaining,
		VideoUsed:      state.VideoAttempts,
		VideoRemaining: videoRemaining,
		ReservedCNY:    state.ReservedCNY,
		RemainingCNY:   remainingCNY,
		PricingVersion: state.PricingVersion,
	}
}

type decimalString struct {
	decimal.Decimal
}

func (d decimalString) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *decimalString) UnmarshalJSON(data []byte) error {
	if d == nil {
		return errors.New("nil decimalString")
	}
	data = bytesTrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		d.Decimal = decimal.Zero
		return nil
	}
	if data[0] == '"' {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		parsed, err := decimal.NewFromString(strings.TrimSpace(raw))
		if err != nil {
			return err
		}
		d.Decimal = parsed
		return nil
	}
	var asFloat float64
	if err := json.Unmarshal(data, &asFloat); err != nil {
		return err
	}
	d.Decimal = decimal.NewFromFloat(asFloat)
	return nil
}

func bytesTrimSpace(data []byte) []byte {
	return []byte(strings.TrimSpace(string(data)))
}

type persistedReservation struct {
	Kind           string        `json:"kind"`
	ReservedCNY    decimalString `json:"reserved_cny"`
	PricingSource  string        `json:"pricing_source,omitempty"`
	PricingVersion string        `json:"pricing_version,omitempty"`
}

type persistedState struct {
	ImageAttempts  int                             `json:"image_attempts"`
	VideoAttempts  int                             `json:"video_attempts"`
	ReservedCNY    decimal.Decimal                 `json:"-"`
	ReservedRaw    decimalString                   `json:"reserved_cny"`
	PricingVersion string                          `json:"pricing_version,omitempty"`
	Reservations   map[string]persistedReservation `json:"reservations,omitempty"`
}

func (s persistedState) MarshalJSON() ([]byte, error) {
	type alias struct {
		ImageAttempts  int                             `json:"image_attempts"`
		VideoAttempts  int                             `json:"video_attempts"`
		ReservedCNY    decimalString                   `json:"reserved_cny"`
		PricingVersion string                          `json:"pricing_version,omitempty"`
		Reservations   map[string]persistedReservation `json:"reservations,omitempty"`
	}
	return json.Marshal(alias{
		ImageAttempts:  s.ImageAttempts,
		VideoAttempts:  s.VideoAttempts,
		ReservedCNY:    decimalString{Decimal: s.ReservedCNY},
		PricingVersion: s.PricingVersion,
		Reservations:   s.Reservations,
	})
}

func (s *persistedState) UnmarshalJSON(data []byte) error {
	type alias struct {
		ImageAttempts  int                             `json:"image_attempts"`
		VideoAttempts  int                             `json:"video_attempts"`
		ReservedCNY    decimalString                   `json:"reserved_cny"`
		PricingVersion string                          `json:"pricing_version"`
		Reservations   map[string]persistedReservation `json:"reservations"`
	}
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.ImageAttempts = raw.ImageAttempts
	s.VideoAttempts = raw.VideoAttempts
	s.ReservedCNY = raw.ReservedCNY.Decimal
	s.ReservedRaw = raw.ReservedCNY
	s.PricingVersion = raw.PricingVersion
	s.Reservations = raw.Reservations
	if s.Reservations == nil {
		s.Reservations = map[string]persistedReservation{}
	}
	return nil
}

func (g *SessionGuard) loadStateLocked() (persistedState, error) {
	data, err := os.ReadFile(g.statePath)
	if errors.Is(err, os.ErrNotExist) {
		return persistedState{Reservations: map[string]persistedReservation{}, ReservedCNY: decimal.Zero}, nil
	}
	if err != nil {
		return persistedState{}, err
	}
	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return persistedState{}, err
	}
	if state.Reservations == nil {
		state.Reservations = map[string]persistedReservation{}
	}
	if state.ImageAttempts < 0 || state.ImageAttempts > MaxImages ||
		state.VideoAttempts < 0 || state.VideoAttempts > MaxVideos ||
		state.ReservedCNY.IsNegative() || state.ReservedCNY.GreaterThan(MaxReservedCNY) {
		return persistedState{}, errors.New("state contains invalid counters")
	}
	return state, nil
}

func (g *SessionGuard) storeState(state persistedState) error {
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

func acquireLock(path string, timeout time.Duration) (*os.File, error) {
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
