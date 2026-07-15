package service

import (
	"context"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// memoryProviderBillingStore is a test double. It deliberately has no hooks to
// mutate users.balance or billing_transactions.
type memoryProviderBillingStore struct {
	mu                   sync.Mutex
	nextImportID         int64
	imports              map[int64]*ProviderBillingImportRecord
	sha256s              map[string]bool
	existingExternalIDs  map[string]bool
	matches              map[int64][]ProviderBillingMatchResult
	videoByUpstream      map[string]ProviderBillingInternalTask
	batchByJobName       map[string]ProviderBillingInternalTask
	internalOnly         []ProviderBillingInternalTask
	balanceTouched       bool
	billingTxTouched     bool
}

func newMemoryProviderBillingStore() *memoryProviderBillingStore {
	return &memoryProviderBillingStore{
		imports:             map[int64]*ProviderBillingImportRecord{},
		sha256s:             map[string]bool{},
		existingExternalIDs: map[string]bool{},
		matches:             map[int64][]ProviderBillingMatchResult{},
		videoByUpstream:     map[string]ProviderBillingInternalTask{},
		batchByJobName:      map[string]ProviderBillingInternalTask{},
	}
}

func (m *memoryProviderBillingStore) HasFileSHA256(_ context.Context, sha256hex string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sha256s[sha256hex], nil
}

func (m *memoryProviderBillingStore) HasProviderExternalLineID(_ context.Context, provider, externalLineID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.existingExternalIDs[provider+"|"+externalLineID], nil
}

func (m *memoryProviderBillingStore) CreateImport(_ context.Context, rec *ProviderBillingImportRecord) (*ProviderBillingImportRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextImportID++
	cp := *rec
	cp.ID = m.nextImportID
	cp.CreatedAt = time.Now().UTC()
	lines := append([]ProviderBillingNormalizedLine{}, rec.Lines...)
	cp.Lines = lines
	m.imports[cp.ID] = &cp
	m.sha256s[cp.FileSHA256] = true
	for _, line := range lines {
		m.existingExternalIDs[cp.Provider+"|"+line.ExternalLineID] = true
	}
	out := cp
	out.Lines = append([]ProviderBillingNormalizedLine{}, lines...)
	return &out, nil
}

func (m *memoryProviderBillingStore) GetImport(_ context.Context, id int64) (*ProviderBillingImportRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.imports[id]
	if !ok {
		return nil, nil
	}
	out := *rec
	out.Lines = append([]ProviderBillingNormalizedLine{}, rec.Lines...)
	return &out, nil
}

func (m *memoryProviderBillingStore) ListImports(_ context.Context, provider string, limit int) ([]ProviderBillingImportRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ProviderBillingImportRecord, 0)
	for _, rec := range m.imports {
		if provider != "" && rec.Provider != provider {
			continue
		}
		cp := *rec
		cp.Lines = nil
		out = append(out, cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memoryProviderBillingStore) ReplaceMatches(_ context.Context, importID int64, matches []ProviderBillingMatchResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := append([]ProviderBillingMatchResult{}, matches...)
	m.matches[importID] = cp
	return nil
}

func (m *memoryProviderBillingStore) ListMatches(_ context.Context, importID int64, status string) ([]ProviderBillingMatchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := m.matches[importID]
	if status == "" {
		return append([]ProviderBillingMatchResult{}, all...), nil
	}
	out := make([]ProviderBillingMatchResult, 0)
	for _, match := range all {
		if string(match.MatchStatus) == status {
			out = append(out, match)
		}
	}
	return out, nil
}

func (m *memoryProviderBillingStore) FindVideoTaskByUpstreamID(_ context.Context, upstreamTaskID string) (*ProviderBillingInternalTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.videoByUpstream[upstreamTaskID]
	if !ok {
		return nil, nil
	}
	cp := task
	return &cp, nil
}

func (m *memoryProviderBillingStore) FindBatchImageJobByProviderJobName(_ context.Context, providerJobName string) (*ProviderBillingInternalTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.batchByJobName[providerJobName]
	if !ok {
		return nil, nil
	}
	cp := task
	return &cp, nil
}

func (m *memoryProviderBillingStore) ListInternalTasksForPeriod(_ context.Context, provider, _ string, _, _ time.Time) ([]ProviderBillingInternalTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := append([]ProviderBillingInternalTask{}, m.internalOnly...)
	if provider == "seedance" {
		for _, task := range m.videoByUpstream {
			out = append(out, task)
		}
	}
	if provider == "gemini" {
		for _, task := range m.batchByJobName {
			out = append(out, task)
		}
	}
	return out, nil
}

func (m *memoryProviderBillingStore) PeriodSummary(_ context.Context, _, _ time.Time) ([]ProviderBillingPeriodSummary, error) {
	return m.BossConclusions(context.Background())
}

func (m *memoryProviderBillingStore) BossConclusions(_ context.Context) ([]ProviderBillingPeriodSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.imports) == 0 {
		return []ProviderBillingPeriodSummary{{
			Conclusion: "not_uploaded",
		}}, nil
	}
	hasDiff := false
	matched := 0
	diff := 0
	for _, list := range m.matches {
		for _, item := range list {
			if item.MatchStatus == ProviderBillingMatchMatched {
				matched++
			} else {
				diff++
				hasDiff = true
			}
		}
	}
	conclusion := "reconciled"
	if hasDiff {
		conclusion = "has_diff"
	}
	return []ProviderBillingPeriodSummary{{
		ImportCount: len(m.imports),
		Matched:     matched,
		HasDiff:     diff,
		Conclusion:  conclusion,
	}}, nil
}

// Ensure unused decimal import stays available for future store helpers.
var _ = decimal.Zero
