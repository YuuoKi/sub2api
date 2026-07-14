package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// Deterministic 1x1 PNG (product mock fixture; no upstream call).
var mockBatchImagePNG = mustDecodeMockPNG()

func mustDecodeMockPNG() []byte {
	// 1x1 transparent PNG
	const b64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO5W7W0AAAAASUVORK5CYII="
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(err)
	}
	return raw
}

// MockBatchImageProvider is a product-level local mock. It is registered only in
// local/review configs (never ordinary production).
type MockBatchImageProvider struct {
	mu      sync.Mutex
	jobs    map[string]*mockBatchImageJobState
	creates int
}

type mockBatchImageJobState struct {
	batchID string
	items   []BatchImageInputItem
	status  string
}

func NewMockBatchImageProvider() *MockBatchImageProvider {
	return &MockBatchImageProvider{jobs: make(map[string]*mockBatchImageJobState)}
}

func (p *MockBatchImageProvider) Name() string { return BatchImageProviderMock }

func (p *MockBatchImageProvider) SupportsAccount(account *Account) bool {
	if account == nil {
		return true
	}
	if truthyAny(account.Extra["mock_provider"]) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(account.GetCredential("api_key")), "local-mock") ||
		strings.Contains(strings.ToLower(account.Name), "mock")
}

func (p *MockBatchImageProvider) Submit(_ context.Context, job *BatchImageJob, _ *Account, input BatchImageInput) (*BatchProviderJob, error) {
	if p == nil {
		return nil, ErrBatchImageProviderSubmitFailed
	}
	batchID := ""
	if job != nil {
		batchID = strings.TrimSpace(job.BatchID)
	}
	if batchID == "" {
		batchID = strings.TrimSpace(input.BatchID)
	}
	if batchID == "" {
		return nil, ErrBatchImageProviderInvalidInput
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.jobs == nil {
		p.jobs = make(map[string]*mockBatchImageJobState)
	}
	p.creates++
	p.jobs[batchID] = &mockBatchImageJobState{
		batchID: batchID,
		items:   append([]BatchImageInputItem(nil), input.Items...),
		status:  BatchImageJobStatusCompleted,
	}
	name := "mock/jobs/" + batchID
	inRef := "mock/input/" + batchID
	outRef := "mock/output/" + batchID
	return &BatchProviderJob{
		ProviderJobName:   name,
		ProviderInputRef:  inRef,
		ProviderOutputRef: outRef,
	}, nil
}

func (p *MockBatchImageProvider) Get(_ context.Context, job *BatchImageJob, _ *Account) (*BatchProviderStatus, error) {
	if job == nil {
		return nil, ErrBatchImageProviderMissingJobName
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	state, ok := p.jobs[strings.TrimSpace(job.BatchID)]
	if !ok {
		return &BatchProviderStatus{
			RawState:      BatchImageJobStatusRunning,
			InternalState: BatchProviderStateRunning,
			Done:          false,
		}, nil
	}
	done := state.status == BatchImageJobStatusCompleted || state.status == BatchImageJobStatusFailed || state.status == BatchImageJobStatusCancelled
	internal := BatchProviderStateRunning
	if state.status == BatchImageJobStatusCompleted {
		internal = BatchProviderStateSucceeded
	} else if state.status == BatchImageJobStatusFailed {
		internal = BatchProviderStateFailed
	} else if state.status == BatchImageJobStatusCancelled {
		internal = BatchProviderStateCancelled
	}
	return &BatchProviderStatus{
		RawState:      state.status,
		InternalState: internal,
		Done:          done,
	}, nil
}

func (p *MockBatchImageProvider) Cancel(_ context.Context, job *BatchImageJob, _ *Account) error {
	if job == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if state, ok := p.jobs[strings.TrimSpace(job.BatchID)]; ok {
		state.status = BatchImageJobStatusCancelled
	}
	return nil
}

func (p *MockBatchImageProvider) OpenResult(_ context.Context, job *BatchImageJob, _ *Account) (io.ReadCloser, string, error) {
	if job == nil {
		return nil, "", ErrBatchImageProviderMissingResultRef
	}
	p.mu.Lock()
	state, ok := p.jobs[strings.TrimSpace(job.BatchID)]
	items := []BatchImageInputItem(nil)
	if ok {
		items = append([]BatchImageInputItem(nil), state.items...)
	}
	p.mu.Unlock()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for i, item := range items {
		customID := strings.TrimSpace(item.CustomID)
		if customID == "" {
			customID = fmt.Sprintf("item_%06d", i+1)
		}
		line := map[string]any{
			"custom_id": customID,
			"response": map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{
									"inlineData": map[string]any{
										"mimeType": "image/png",
										"data":     base64.StdEncoding.EncodeToString(mockBatchImagePNG),
									},
								},
							},
						},
					},
				},
			},
		}
		if err := enc.Encode(line); err != nil {
			return nil, "", err
		}
	}
	if buf.Len() == 0 {
		_ = enc.Encode(map[string]any{
			"custom_id": "item_000001",
			"response": map[string]any{
				"candidates": []any{
					map[string]any{
						"content": map[string]any{
							"parts": []any{
								map[string]any{
									"inlineData": map[string]any{
										"mimeType": "image/png",
										"data":     base64.StdEncoding.EncodeToString(mockBatchImagePNG),
									},
								},
							},
						},
					},
				},
			},
		})
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes())), "application/x-ndjson", nil
}

func (p *MockBatchImageProvider) Cleanup(context.Context, *BatchImageJob, *Account, CleanupTarget) error {
	return nil
}

func (p *MockBatchImageProvider) CreateCount() int {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.creates
}

// AllowProductMockBatchImageProvider reports whether the product mock image
// provider may be registered for this process config.
func AllowProductMockBatchImageProvider(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	env := strings.ToLower(strings.TrimSpace(cfg.Log.Environment))
	mode := strings.ToLower(strings.TrimSpace(cfg.Server.Mode))
	switch env {
	case "local", "review", "dev", "development", "test":
		return true
	}
	switch mode {
	case "debug", "test", "local", "review":
		return true
	}
	// Real-review session armed implies a review harness.
	if cfg.RealReviewSessionActive() {
		return true
	}
	return false
}

func NewBatchImageProviderRegistryFromConfig(cfg *config.Config) *BatchImageProviderRegistry {
	providers := []BatchImageProvider{
		NewGeminiAPIBatchImageProvider(nil),
		NewVertexBatchImageProviderFromConfig(cfg, nil, nil, nil),
	}
	if AllowProductMockBatchImageProvider(cfg) {
		providers = append(providers, NewMockBatchImageProvider())
	}
	return NewBatchImageProviderRegistry(providers...)
}
