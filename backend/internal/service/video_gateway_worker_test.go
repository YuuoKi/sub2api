package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

type noopVideoKeyEncryptor struct{}

func (noopVideoKeyEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (noopVideoKeyEncryptor) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

type memoryVideoGatewayRepo struct {
	nextProviderID int64
	nextTaskID     int64
	nextEventID    int64
	providers      map[int64]*VideoProviderAccount
	tasks          map[int64]*VideoTask
	events         []*VideoTaskEvent
	usage          []*VideoTask
}

func newMemoryVideoGatewayRepo() *memoryVideoGatewayRepo {
	return &memoryVideoGatewayRepo{
		nextProviderID: 1,
		nextTaskID:     1,
		nextEventID:    1,
		providers:      make(map[int64]*VideoProviderAccount),
		tasks:          make(map[int64]*VideoTask),
	}
}

func (r *memoryVideoGatewayRepo) seedMockProvider() int64 {
	id := r.nextProviderID
	r.nextProviderID++
	now := time.Now().UTC()
	r.providers[id] = &VideoProviderAccount{
		ID:                 id,
		Provider:           VideoProviderMock,
		DisplayName:        "Mock Provider",
		Enabled:            true,
		BaseURL:            "mock://video-gateway",
		DefaultModel:       "mock-video-v1",
		RateLimitPerMinute: 60,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return id
}

func (r *memoryVideoGatewayRepo) CreateProviderAccount(_ context.Context, account *VideoProviderAccount) error {
	now := time.Now().UTC()
	account.ID = r.nextProviderID
	r.nextProviderID++
	account.CreatedAt = now
	account.UpdatedAt = now
	r.providers[account.ID] = cloneVideoProvider(account)
	return nil
}

func (r *memoryVideoGatewayRepo) GetProviderAccount(_ context.Context, id int64) (*VideoProviderAccount, error) {
	account, ok := r.providers[id]
	if !ok {
		return nil, ErrVideoProviderNotFound
	}
	return cloneVideoProvider(account), nil
}

func (r *memoryVideoGatewayRepo) ListProviderAccounts(_ context.Context) ([]*VideoProviderAccount, error) {
	out := make([]*VideoProviderAccount, 0, len(r.providers))
	for _, account := range r.providers {
		out = append(out, cloneVideoProvider(account))
	}
	return out, nil
}

func (r *memoryVideoGatewayRepo) UpdateProviderAccount(_ context.Context, account *VideoProviderAccount) error {
	account.UpdatedAt = time.Now().UTC()
	r.providers[account.ID] = cloneVideoProvider(account)
	return nil
}

func (r *memoryVideoGatewayRepo) CreateTask(_ context.Context, task *VideoTask) error {
	now := time.Now().UTC()
	task.ID = r.nextTaskID
	r.nextTaskID++
	task.CreatedAt = now
	task.UpdatedAt = now
	r.tasks[task.ID] = cloneVideoTask(task)
	return nil
}

func (r *memoryVideoGatewayRepo) GetTask(_ context.Context, id int64) (*VideoTask, error) {
	task, ok := r.tasks[id]
	if !ok {
		return nil, ErrVideoTaskNotFound
	}
	return cloneVideoTask(task), nil
}

func (r *memoryVideoGatewayRepo) ListTasks(_ context.Context, _ VideoTaskListParams) ([]*VideoTask, int64, error) {
	out := make([]*VideoTask, 0, len(r.tasks))
	for _, task := range r.tasks {
		out = append(out, cloneVideoTask(task))
	}
	return out, int64(len(out)), nil
}

func (r *memoryVideoGatewayRepo) ListRunnableTasks(_ context.Context, limit int) ([]*VideoTask, error) {
	out := make([]*VideoTask, 0, limit)
	for _, task := range r.tasks {
		if IsTerminalVideoStatus(task.Status) {
			continue
		}
		out = append(out, cloneVideoTask(task))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *memoryVideoGatewayRepo) UpdateTask(_ context.Context, task *VideoTask) error {
	task.UpdatedAt = time.Now().UTC()
	r.tasks[task.ID] = cloneVideoTask(task)
	return nil
}

func (r *memoryVideoGatewayRepo) AddTaskEvent(_ context.Context, event *VideoTaskEvent) error {
	event.ID = r.nextEventID
	r.nextEventID++
	event.CreatedAt = time.Now().UTC()
	r.events = append(r.events, cloneVideoEvent(event))
	return nil
}

func (r *memoryVideoGatewayRepo) ListTaskEvents(_ context.Context, taskID int64, _ int) ([]*VideoTaskEvent, error) {
	out := make([]*VideoTaskEvent, 0)
	for _, event := range r.events {
		if event.VideoTaskID == taskID {
			out = append(out, cloneVideoEvent(event))
		}
	}
	return out, nil
}

func (r *memoryVideoGatewayRepo) InsertUsageLog(_ context.Context, task *VideoTask) error {
	r.usage = append(r.usage, cloneVideoTask(task))
	return nil
}

func (r *memoryVideoGatewayRepo) CountTasksSince(_ context.Context, _ time.Time) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (r *memoryVideoGatewayRepo) CountProviderTasksSince(_ context.Context, _ time.Time) (map[string]map[string]int64, error) {
	return map[string]map[string]int64{}, nil
}

func (r *memoryVideoGatewayRepo) ListRecentTasksByStatus(_ context.Context, _ string, _ int) ([]*VideoTask, error) {
	return []*VideoTask{}, nil
}

func (r *memoryVideoGatewayRepo) UsageSummarySince(_ context.Context, _ time.Time) ([]VideoUsageSummary, error) {
	return []VideoUsageSummary{}, nil
}

func TestVideoGatewayMockWorkerSuccessAndFailure(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)

	success, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "render a quiet enterprise dashboard demo",
		CreatedBy:         7,
	})
	if err != nil {
		t.Fatalf("create success task: %v", err)
	}
	for range 3 {
		if err := svc.ProcessRunnableTasks(ctx, 10, time.Minute); err != nil {
			t.Fatalf("process success task: %v", err)
		}
	}
	success, _, err = svc.GetTask(ctx, success.ID, 7, false)
	if err != nil {
		t.Fatalf("get success task: %v", err)
	}
	if success.Status != VideoStatusSucceeded {
		t.Fatalf("expected succeeded, got %s", success.Status)
	}
	if !strings.Contains(success.ResultURL, "mock.sub2api.local") {
		t.Fatalf("expected mock result url, got %q", success.ResultURL)
	}

	failed, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "mock:fail validation path",
		CreatedBy:         7,
	})
	if err != nil {
		t.Fatalf("create failure task: %v", err)
	}
	for range 3 {
		if err := svc.ProcessRunnableTasks(ctx, 10, time.Minute); err != nil {
			t.Fatalf("process failure task: %v", err)
		}
	}
	failed, events, err := svc.GetTask(ctx, failed.ID, 7, false)
	if err != nil {
		t.Fatalf("get failure task: %v", err)
	}
	if failed.Status != VideoStatusFailed {
		t.Fatalf("expected failed, got %s", failed.Status)
	}
	if failed.ErrorMessage == "" {
		t.Fatal("expected failure error message")
	}
	if len(events) < 4 {
		t.Fatalf("expected queued/submitted/running/failed events, got %d", len(events))
	}
	if len(repo.usage) != 2 {
		t.Fatalf("expected usage logs for terminal tasks, got %d", len(repo.usage))
	}
}

func TestVideoProviderKeyNeverReturnedInPlaintext(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)

	created, err := svc.CreateProviderAccount(ctx, VideoProviderCreateParams{
		Provider:     VideoProviderSeedance,
		DisplayName:  "Seedance Demo",
		Enabled:      false,
		APIKey:       "sk-real-looking-demo-value",
		DefaultModel: "seedance-2-0-pro",
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if created.PlainAPIKey != "" {
		t.Fatal("created provider response exposed plaintext key")
	}
	if created.MaskedKey == "" || created.MaskedKey == "sk-real-looking-demo-value" {
		t.Fatalf("expected masked key, got %q", created.MaskedKey)
	}

	items, err := svc.ListProviderAccounts(ctx)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one provider, got %d", len(items))
	}
	if items[0].PlainAPIKey != "" {
		t.Fatal("list provider response exposed plaintext key")
	}
	if !items[0].APIKeyConfigured {
		t.Fatal("expected api_key_configured")
	}
}

func cloneVideoProvider(in *VideoProviderAccount) *VideoProviderAccount {
	if in == nil {
		return nil
	}
	out := *in
	if in.Metadata != nil {
		out.Metadata = make(map[string]any, len(in.Metadata))
		for k, v := range in.Metadata {
			out.Metadata[k] = v
		}
	}
	return &out
}

func cloneVideoTask(in *VideoTask) *VideoTask {
	if in == nil {
		return nil
	}
	out := *in
	if in.CompletedAt != nil {
		completed := *in.CompletedAt
		out.CompletedAt = &completed
	}
	return &out
}

func cloneVideoEvent(in *VideoTaskEvent) *VideoTaskEvent {
	if in == nil {
		return nil
	}
	out := *in
	if in.Payload != nil {
		out.Payload = make(map[string]any, len(in.Payload))
		for k, v := range in.Payload {
			out.Payload[k] = v
		}
	}
	return &out
}
