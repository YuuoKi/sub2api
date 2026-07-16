package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	AssetHandoffImage AssetHandoffKind = "image"
	AssetHandoffVideo AssetHandoffKind = "video"

	assetHandoffLifetime = 5 * time.Minute
	assetHandoffMaxBytes = int64(30 * 1024 * 1024)
)

var (
	ErrAssetHandoffNotFound         = errors.New("asset handoff ticket not found")
	ErrAssetHandoffConsumed         = errors.New("asset handoff ticket already consumed")
	ErrAssetHandoffExpired          = errors.New("asset handoff ticket expired")
	ErrAssetHandoffInvalidIssuer    = errors.New("asset handoff issuer is required")
	ErrAssetHandoffInvalidKind      = errors.New("asset handoff kind must be image or video")
	ErrAssetHandoffTaskNotSucceeded = errors.New("asset handoff requires a succeeded task")
	ErrAssetHandoffAssetMissing     = errors.New("asset handoff source asset is missing")
	ErrAssetHandoffInvalidMIME      = errors.New("asset handoff asset MIME is not allowed")
	ErrAssetHandoffTooLarge         = errors.New("asset handoff asset exceeds 30 MB")
	ErrAssetHandoffUnverifiable     = errors.New("asset handoff asset metadata cannot be verified")
)

type AssetHandoffKind string

type AssetInspection struct {
	MIME      string
	SizeBytes int64
}

type AssetInspector interface {
	Inspect(context.Context, string) (AssetInspection, error)
}

type AssetHandoffTaskReader interface {
	GetVideoTaskAdmin(context.Context, int64) (*VideoTask, error)
}

type AssetHandoffManager interface {
	Issue(context.Context, int64, int64, AssetHandoffKind) (*IssuedAssetHandoff, error)
	Consume(context.Context, string) (*ConsumedAssetHandoff, error)
}

type IssuedAssetHandoff struct {
	Ticket       string           `json:"ticket"`
	SourceTaskID int64            `json:"source_task_id"`
	AssetKind    AssetHandoffKind `json:"asset_kind"`
	ExpiresAt    time.Time        `json:"expires_at"`
}

type ConsumedAssetHandoff struct {
	SourceTaskID int64            `json:"source_task_id"`
	AssetKind    AssetHandoffKind `json:"asset_kind"`
	URL          string           `json:"asset_url"`
	MIME         string           `json:"content_type"`
	SizeBytes    int64            `json:"size_bytes"`
	ExpiresAt    time.Time        `json:"expires_at"`
}

type assetHandoffRecord struct {
	// IssuerID records the authenticated Sub2API administrator that issued the
	// ticket. Cross-platform consumption deliberately does not accept a user ID:
	// the high-entropy one-time ticket is the bearer capability and the HTTP
	// boundary independently requires either a real loopback peer or the
	// explicitly enabled canonical Docker-NAT loopback contract.
	IssuerID  int64
	TaskID    int64
	Kind      AssetHandoffKind
	ExpiresAt time.Time
	Consumed  bool
}

type AssetHandoffService struct {
	repo      AssetHandoffTaskReader
	inspector AssetInspector
	now       func() time.Time
	random    io.Reader

	mu      sync.Mutex
	tickets map[string]assetHandoffRecord
}

func NewAssetHandoffService(repo AssetHandoffTaskReader, inspector AssetInspector, now func() time.Time, random io.Reader) *AssetHandoffService {
	if now == nil {
		now = time.Now
	}
	if random == nil {
		random = rand.Reader
	}
	return &AssetHandoffService{
		repo:      repo,
		inspector: inspector,
		now:       now,
		random:    random,
		tickets:   make(map[string]assetHandoffRecord),
	}
}

func (s *AssetHandoffService) Issue(ctx context.Context, issuerID, taskID int64, kind AssetHandoffKind) (*IssuedAssetHandoff, error) {
	if issuerID <= 0 {
		return nil, ErrAssetHandoffInvalidIssuer
	}
	if _, _, err := s.resolveAndInspect(ctx, taskID, kind); err != nil {
		return nil, err
	}

	raw := make([]byte, 32)
	if _, err := io.ReadFull(s.random, raw); err != nil {
		return nil, fmt.Errorf("generate asset handoff ticket: %w", err)
	}
	ticket := base64.RawURLEncoding.EncodeToString(raw)
	now := s.now().UTC()
	expiresAt := now.Add(assetHandoffLifetime)
	record := assetHandoffRecord{IssuerID: issuerID, TaskID: taskID, Kind: kind, ExpiresAt: expiresAt}

	s.mu.Lock()
	for digest, existing := range s.tickets {
		if existing.Consumed || !now.Before(existing.ExpiresAt) {
			delete(s.tickets, digest)
		}
	}
	s.tickets[ticketDigest(ticket)] = record
	s.mu.Unlock()

	return &IssuedAssetHandoff{Ticket: ticket, SourceTaskID: taskID, AssetKind: kind, ExpiresAt: expiresAt}, nil
}

func (s *AssetHandoffService) Consume(ctx context.Context, ticket string) (*ConsumedAssetHandoff, error) {
	digest := ticketDigest(strings.TrimSpace(ticket))
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.tickets[digest]
	if !ok {
		return nil, ErrAssetHandoffNotFound
	}
	if record.Consumed {
		return nil, ErrAssetHandoffConsumed
	}
	if !s.now().UTC().Before(record.ExpiresAt) {
		return nil, ErrAssetHandoffExpired
	}
	assetURL, inspection, err := s.resolveAndInspect(ctx, record.TaskID, record.Kind)
	if err != nil {
		return nil, err
	}
	record.Consumed = true
	s.tickets[digest] = record
	return &ConsumedAssetHandoff{
		SourceTaskID: record.TaskID,
		AssetKind:    record.Kind,
		URL:          assetURL,
		MIME:         inspection.MIME,
		SizeBytes:    inspection.SizeBytes,
		ExpiresAt:    record.ExpiresAt,
	}, nil
}

func (s *AssetHandoffService) resolveAndInspect(ctx context.Context, taskID int64, kind AssetHandoffKind) (string, AssetInspection, error) {
	if kind != AssetHandoffImage && kind != AssetHandoffVideo {
		return "", AssetInspection{}, ErrAssetHandoffInvalidKind
	}
	task, err := s.repo.GetVideoTaskAdmin(ctx, taskID)
	if err != nil {
		return "", AssetInspection{}, err
	}
	if task == nil || task.Status != VideoStatusSucceeded {
		return "", AssetInspection{}, ErrAssetHandoffTaskNotSucceeded
	}
	assetURL := strings.TrimSpace(task.ResultURL)
	expectedMIME := "video/mp4"
	if kind == AssetHandoffImage {
		assetURL = strings.TrimSpace(task.LastFrameURL)
		expectedMIME = "image/png"
	}
	if assetURL == "" {
		return "", AssetInspection{}, ErrAssetHandoffAssetMissing
	}
	inspection, err := s.inspector.Inspect(ctx, assetURL)
	if err != nil {
		return "", AssetInspection{}, fmt.Errorf("%w: %v", ErrAssetHandoffUnverifiable, err)
	}
	inspection.MIME = strings.ToLower(strings.TrimSpace(inspection.MIME))
	if inspection.MIME != expectedMIME {
		return "", AssetInspection{}, ErrAssetHandoffInvalidMIME
	}
	if inspection.SizeBytes <= 0 {
		return "", AssetInspection{}, ErrAssetHandoffUnverifiable
	}
	if inspection.SizeBytes > assetHandoffMaxBytes {
		return "", AssetInspection{}, ErrAssetHandoffTooLarge
	}
	return assetURL, inspection, nil
}

func ticketDigest(ticket string) string {
	sum := sha256.Sum256([]byte(ticket))
	return hex.EncodeToString(sum[:])
}
