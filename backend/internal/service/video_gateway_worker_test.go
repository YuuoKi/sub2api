package service

import (
	"context"
	"fmt"
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

func (r *memoryVideoGatewayRepo) ListTasks(_ context.Context, params VideoTaskListParams) ([]*VideoTask, int64, error) {
	out := make([]*VideoTask, 0, len(r.tasks))
	for _, task := range r.tasks {
		if params.Status != "" && task.Status != params.Status {
			continue
		}
		if params.Provider != "" && task.Provider != params.Provider {
			continue
		}
		if !params.IsAdmin && task.CreatedBy != params.CreatedBy {
			continue
		}
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

func (r *memoryVideoGatewayRepo) ProviderAccountTaskStatsSince(_ context.Context, since time.Time) (map[int64]VideoProviderRuntimeStats, error) {
	out := map[int64]VideoProviderRuntimeStats{}
	for _, task := range r.tasks {
		item := out[task.ProviderAccountID]
		if !task.CreatedAt.Before(since) {
			item.TodayTasks++
			if task.Status == VideoStatusFailed {
				item.TodayFailures++
			}
		}
		switch task.Status {
		case VideoStatusQueued, VideoStatusSubmitted, VideoStatusRunning:
			item.CurrentInflight++
		}
		if task.Status == VideoStatusFailed && task.ErrorMessage != "" {
			if item.LastErrorAt == nil || task.UpdatedAt.After(*item.LastErrorAt) {
				updatedAt := task.UpdatedAt
				item.LastError = task.ErrorMessage
				item.LastErrorAt = &updatedAt
			}
		}
		out[task.ProviderAccountID] = item
	}
	return out, nil
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

func TestVideoProviderMaskedPlaceholderDoesNotMarkRealProviderConfigured(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	mockID := repo.seedMockProvider()
	now := time.Now().UTC()
	realProviderID := repo.nextProviderID
	repo.nextProviderID++
	repo.providers[realProviderID] = &VideoProviderAccount{
		ID:           realProviderID,
		Provider:     VideoProviderSeedance,
		DisplayName:  "Seedance Placeholder Only",
		Enabled:      true,
		MaskedKey:    "sdnc***demo",
		DefaultModel: defaultVideoModel(VideoProviderSeedance),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)

	items, err := svc.ListProviderAccounts(ctx)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	var mockProvider, realProvider *VideoProviderAccount
	for _, item := range items {
		switch item.ID {
		case mockID:
			mockProvider = item
		case realProviderID:
			realProvider = item
		}
	}
	if mockProvider == nil || !mockProvider.APIKeyConfigured || !mockProvider.RouteAvailable {
		t.Fatalf("expected mock provider to remain demo-ready, got %#v", mockProvider)
	}
	if realProvider == nil {
		t.Fatal("expected real provider in list")
	}
	if realProvider.APIKeyConfigured {
		t.Fatalf("masked placeholder should not mark real provider configured: %#v", realProvider)
	}
	if realProvider.KeyStatus != videoKeyStatusMissing || realProvider.RouteAvailable {
		t.Fatalf("expected missing-key real provider to be unavailable, got key_status=%s route_available=%v", realProvider.KeyStatus, realProvider.RouteAvailable)
	}
}

func TestVideoGatewayAutoRouteSkipsUnavailableAccounts(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	availableID := repo.seedMockProvider()
	now := time.Now().UTC()
	unavailable := []*VideoProviderAccount{
		{
			ID:           repo.nextProviderID,
			Provider:     VideoProviderSeedance,
			DisplayName:  "Seedance Missing Key",
			Enabled:      true,
			DefaultModel: defaultVideoModel(VideoProviderSeedance),
			Metadata: map[string]any{
				"key_status":      videoKeyStatusMissing,
				"health_status":   videoHealthStatusNeedsKey,
				"diagnostic_type": "Key 未配置",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:           repo.nextProviderID + 1,
			Provider:     VideoProviderKling,
			DisplayName:  "Kling Disabled",
			Enabled:      false,
			DefaultModel: defaultVideoModel(VideoProviderKling),
			Metadata: map[string]any{
				"key_status":    videoKeyStatusDisabled,
				"health_status": videoHealthStatusDisabled,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:           repo.nextProviderID + 2,
			Provider:     VideoProviderSeedance,
			DisplayName:  "Seedance Auth Failed",
			Enabled:      true,
			MaskedKey:    "sdnc***demo",
			DefaultModel: defaultVideoModel(VideoProviderSeedance),
			Metadata: map[string]any{
				"key_status":      videoKeyStatusAuthFailed,
				"health_status":   videoHealthStatusAuthFailed,
				"diagnostic_type": "鉴权失败",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:           repo.nextProviderID + 3,
			Provider:     VideoProviderKling,
			DisplayName:  "Kling Rate Limited",
			Enabled:      true,
			MaskedKey:    "klng***demo",
			DefaultModel: defaultVideoModel(VideoProviderKling),
			Metadata: map[string]any{
				"key_status":      videoKeyStatusRateLimited,
				"health_status":   videoHealthStatusRateLimited,
				"diagnostic_type": "触发限流",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	for _, account := range unavailable {
		repo.providers[account.ID] = cloneVideoProvider(account)
		repo.nextProviderID = account.ID + 1
	}
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)

	var task *VideoTask
	for i := 0; i < 10; i++ {
		var err error
		task, err = svc.CreateTask(ctx, VideoTaskCreateParams{
			TaskType:  VideoTaskTypeTextToVideo,
			Prompt:    "create a short launch video for an enterprise API workflow",
			CreatedBy: 101,
		})
		if err != nil {
			t.Fatalf("auto route create task %d: %v", i+1, err)
		}
	}
	if task.ProviderAccountID != availableID {
		t.Fatalf("expected available provider %d, got %d", availableID, task.ProviderAccountID)
	}
	if len(repo.tasks) != 10 {
		t.Fatalf("expected ten routed tasks, got %d", len(repo.tasks))
	}
	events, err := repo.ListTaskEvents(ctx, task.ID, 20)
	if err != nil {
		t.Fatalf("list task events: %v", err)
	}
	if len(events) == 0 || events[0].EventType != "routed" {
		t.Fatalf("expected first routed event, got %#v", events)
	}
	if events[0].Payload["strategy"] != VideoRouteStrategyLeastInflight {
		t.Fatalf("expected least_inflight strategy, got %#v", events[0].Payload["strategy"])
	}
	skipped, ok := events[0].Payload["skipped_accounts"].([]videoRouteSkip)
	if !ok {
		t.Fatalf("expected skipped account records, got %#v", events[0].Payload["skipped_accounts"])
	}
	if len(skipped) != 4 {
		t.Fatalf("expected four skipped accounts, got %d", len(skipped))
	}
}

func TestVideoGatewayTaskOwnershipBoundary(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		TaskType:  VideoTaskTypeTextToVideo,
		Prompt:    "ownership check",
		CreatedBy: 101,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if _, _, err := svc.GetTask(ctx, task.ID, 202, false); err == nil {
		t.Fatal("expected a different non-admin user to be blocked")
	}
	if _, _, err := svc.GetTask(ctx, task.ID, 202, true); err != nil {
		t.Fatalf("expected admin to read task: %v", err)
	}
}

func TestVideoGatewayTaskListFiltersByEmployeeUnlessAdmin(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)

	if _, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		TaskType:  VideoTaskTypeTextToVideo,
		Prompt:    "employee one task",
		CreatedBy: 101,
	}); err != nil {
		t.Fatalf("create first task: %v", err)
	}
	if _, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		TaskType:  VideoTaskTypeTextToVideo,
		Prompt:    "employee two task",
		CreatedBy: 202,
	}); err != nil {
		t.Fatalf("create second task: %v", err)
	}

	employeeTasks, total, err := svc.ListTasks(ctx, VideoTaskListParams{CreatedBy: 101, IsAdmin: false})
	if err != nil {
		t.Fatalf("list employee tasks: %v", err)
	}
	if total != 1 || len(employeeTasks) != 1 || employeeTasks[0].CreatedBy != 101 {
		t.Fatalf("expected only employee 101 task, total=%d tasks=%#v", total, employeeTasks)
	}

	adminTasks, total, err := svc.ListTasks(ctx, VideoTaskListParams{CreatedBy: 101, IsAdmin: true})
	if err != nil {
		t.Fatalf("list admin tasks: %v", err)
	}
	if total != 2 || len(adminTasks) != 2 {
		t.Fatalf("expected admin to see both tasks, total=%d tasks=%#v", total, adminTasks)
	}
}

func TestVideoAdapterContractSafeProviderBehavior(t *testing.T) {
	ctx := context.Background()
	registry := NewVideoAdapterRegistry()
	task := &VideoTask{
		ID:          42,
		Model:       "mock-video-v1",
		TaskType:    VideoTaskTypeTextToVideo,
		Prompt:      "safe adapter contract preview",
		AspectRatio: "16:9",
		Duration:    5,
		Resolution:  "720p",
	}

	mock := registry[VideoProviderMock]
	if mock == nil {
		t.Fatal("mock adapter is not registered")
	}
	mockPayload := mock.BuildCreatePayload(&VideoProviderAccount{Provider: VideoProviderMock}, task)
	if containsSecretField(mockPayload) {
		t.Fatalf("mock payload should not expose secret-like fields: %#v", mockPayload)
	}
	result, err := mock.CreateTask(ctx, &VideoProviderAccount{Provider: VideoProviderMock, Enabled: true}, task)
	if err != nil {
		t.Fatalf("mock create task: %v", err)
	}
	if result.Status != VideoStatusSubmitted || result.UpstreamTaskID == "" {
		t.Fatalf("unexpected mock create result: %#v", result)
	}

	for _, provider := range []string{VideoProviderSeedance, VideoProviderKling} {
		adapter := registry[provider]
		if adapter == nil {
			t.Fatalf("%s adapter is not registered", provider)
		}
		preview := adapter.BuildCreatePayload(&VideoProviderAccount{
			Provider:         provider,
			APIKeyConfigured: true,
			PlainAPIKey:      "placeholder-key-should-not-leak",
			BaseURL:          "demo://safe-provider",
		}, task)
		if containsSecretField(preview) || strings.Contains(fmtAny(preview), "placeholder-key-should-not-leak") {
			t.Fatalf("%s payload preview should not expose credentials: %#v", provider, preview)
		}
		if _, err := adapter.CreateTask(ctx, &VideoProviderAccount{Provider: provider, Enabled: true}, task); err == nil || !strings.Contains(err.Error(), "api key is not configured") {
			t.Fatalf("%s without key should be safely disabled, err=%v", provider, err)
		}
		if _, err := adapter.CreateTask(ctx, &VideoProviderAccount{
			Provider:         provider,
			Enabled:          true,
			APIKeyConfigured: true,
			PlainAPIKey:      "placeholder-key-should-not-leak",
		}, task); err == nil || !strings.Contains(strings.ToLower(err.Error()), "disabled") {
			t.Fatalf("%s real call should remain disabled, err=%v", provider, err)
		}
	}
}

func containsSecretField(payload map[string]any) bool {
	for key, value := range payload {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "api_key") ||
			strings.Contains(lowerKey, "authorization") ||
			strings.Contains(lowerKey, "cookie") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "secret") ||
			strings.Contains(lowerKey, "password") {
			return true
		}
		if nested, ok := value.(map[string]any); ok && containsSecretField(nested) {
			return true
		}
		if nestedSlice, ok := value.([]map[string]any); ok {
			for _, nested := range nestedSlice {
				if containsSecretField(nested) {
					return true
				}
			}
		}
	}
	return false
}

func fmtAny(value any) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(fmt.Sprintf("%#v", value))), "\n", " "), "\t", " "))
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
