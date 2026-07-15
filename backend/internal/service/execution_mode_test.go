package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/reviewguard"
)

func TestNormalizeExecutionModeDefaultsToMock(t *testing.T) {
	got, err := NormalizeExecutionMode("")
	if err != nil {
		t.Fatalf("NormalizeExecutionMode: %v", err)
	}
	if got != ExecutionModeMock {
		t.Fatalf("got %q, want mock", got)
	}
}

func TestNormalizeExecutionModeRejectsUnknown(t *testing.T) {
	if _, err := NormalizeExecutionMode("prod"); err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestCreateTaskMockModeNeverAutoRoutesRealProviders(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	mockID := repo.seedMockProvider()
	seedanceID := repo.nextProviderID
	repo.nextProviderID++
	repo.providers[seedanceID] = &VideoProviderAccount{
		ID:             seedanceID,
		Provider:       VideoProviderSeedance,
		DisplayName:    "Seedance Real",
		Enabled:        true,
		DefaultModel:   "seedance-1-0-pro",
		RouteAvailable: true,
		APIKeyConfigured: true,
		EncryptedAPIKey:  "enc",
		Metadata:         map[string]any{"production_authorized": true},
	}
	// Force route availability decoration path: set fields used by resolve.
	repo.providers[mockID].RouteAvailable = true
	repo.providers[mockID].Enabled = true

	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, &config.Config{})
	task, err := svc.CreateTask(context.Background(), VideoTaskCreateParams{
		ExecutionMode:     ExecutionModeMock,
		ProviderAccountID: 0, // historical hole: least-inflight across ALL providers
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "mock only please",
		CreatedBy:         42,
		Duration:          5,
		Resolution:        "720p",
		AspectRatio:       "16:9",
	})
	if err != nil {
		t.Fatalf("CreateTask mock: %v", err)
	}
	if task.Provider != VideoProviderMock {
		t.Fatalf("provider=%q, want mock", task.Provider)
	}
	if task.ProviderAccountID != mockID {
		t.Fatalf("provider_account_id=%d, want mock %d", task.ProviderAccountID, mockID)
	}
	if task.ExecutionMode != ExecutionModeMock {
		t.Fatalf("execution_mode=%q, want mock", task.ExecutionMode)
	}
}

func TestCreateTaskIgnoresBareProviderAccountIDInMockMode(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	mockID := repo.seedMockProvider()
	repo.providers[mockID].RouteAvailable = true
	repo.providers[mockID].Enabled = true
	seedanceID := repo.nextProviderID
	repo.nextProviderID++
	repo.providers[seedanceID] = &VideoProviderAccount{
		ID:               seedanceID,
		Provider:         VideoProviderSeedance,
		DisplayName:      "Seedance Real",
		Enabled:          true,
		DefaultModel:     "seedance-1-0-pro",
		RouteAvailable:   true,
		APIKeyConfigured: true,
		EncryptedAPIKey:  "enc",
	}
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, &config.Config{})
	task, err := svc.CreateTask(context.Background(), VideoTaskCreateParams{
		ExecutionMode:     ExecutionModeMock,
		ProviderAccountID: seedanceID, // bare id must not win over mock mode
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "ignore bare id",
		CreatedBy:         7,
		Duration:          5,
		Resolution:        "720p",
		AspectRatio:       "16:9",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.ProviderAccountID != mockID || task.Provider != VideoProviderMock {
		t.Fatalf("routed to provider=%s id=%d, want mock id=%d", task.Provider, task.ProviderAccountID, mockID)
	}
}

func TestCreateTaskReviewRealRequiresSessionAndReviewAccount(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	mockID := repo.seedMockProvider()
	repo.providers[mockID].RouteAvailable = true
	repo.providers[mockID].Enabled = true
	reviewID := repo.nextProviderID
	repo.nextProviderID++
	repo.providers[reviewID] = &VideoProviderAccount{
		ID:               reviewID,
		Provider:         VideoProviderSeedance,
		DisplayName:      "Seedance Review",
		Enabled:          true,
		DefaultModel:     "seedance-1-0-pro",
		RouteAvailable:   true,
		APIKeyConfigured: true,
		EncryptedAPIKey:  "enc",
		Metadata:         map[string]any{"review_only": true, "production_authorized": true},
	}
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, &config.Config{})
	svc.SetRealCreateGuard(reviewguard.NewFailClosedGuard())

	_, err := svc.CreateTask(context.Background(), VideoTaskCreateParams{
		ExecutionMode: ExecutionModeReviewReal,
		TaskType:      VideoTaskTypeTextToVideo,
		Prompt:        "review real",
		CreatedBy:     9,
		Duration:      5,
		Resolution:    "720p",
		AspectRatio:   "16:9",
	})
	if err == nil {
		t.Fatal("expected session-disabled rejection before create")
	}
	if !strings.Contains(err.Error(), "REAL_REVIEW_SESSION_DISABLED") && !strings.Contains(err.Error(), "真实复核") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.tasks) != 0 {
		t.Fatalf("create count=%d, want 0 when session disabled", len(repo.tasks))
	}
}

func TestVideoRealCreateGuardReservesOnlyReviewRealMode(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.nextProviderID
	repo.nextProviderID++
	repo.providers[providerID] = &VideoProviderAccount{
		ID: providerID, Provider: VideoProviderSeedance, DisplayName: "Seedance", Enabled: true, DefaultModel: "seedance-1-0-pro",
		Metadata: map[string]any{"review_only": true},
	}
	task := &VideoTask{
		ID:                9501,
		ProviderAccountID: providerID,
		Provider:          VideoProviderSeedance,
		Model:             "seedance-1-0-pro",
		Status:            VideoStatusQueued,
		Version:           1,
		DispatchState:     "pending",
		Duration:          5,
		CostEstimate:      10,
		Currency:          "CNY",
		PricingSource:     "seedance_catalog",
		PricingVersion:    "v1",
		ExecutionMode:     ExecutionModeReviewReal,
	}
	repo.tasks[task.ID] = cloneVideoTask(task)
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, &config.Config{})
	svc.SetRealCreateGuard(reviewguard.NewFailClosedGuard())
	adapter := &recordingDispatchAdapter{result: &VideoAdapterResult{UpstreamTaskID: "must-not-create", Status: VideoStatusSubmitted}}

	err := svc.submitTask(context.Background(), adapter, repo.providers[providerID], task)
	if err == nil {
		t.Fatal("expected review_real fail-closed guard to reject")
	}
	if adapter.calls() != 0 {
		t.Fatalf("adapter create calls = %d, want 0", adapter.calls())
	}

	// internal_real must NOT consume the 4+4/¥60 session guard.
	internalTask := &VideoTask{
		ID:                9502,
		ProviderAccountID: providerID,
		Provider:          VideoProviderSeedance,
		Model:             "seedance-1-0-pro",
		Status:            VideoStatusQueued,
		Version:           1,
		DispatchState:     "pending",
		Duration:          5,
		CostEstimate:      10,
		Currency:          "CNY",
		ExecutionMode:     ExecutionModeInternalReal,
	}
	repo.providers[providerID].Metadata = map[string]any{} // formal account
	repo.tasks[internalTask.ID] = cloneVideoTask(internalTask)
	adapter2 := &recordingDispatchAdapter{result: &VideoAdapterResult{UpstreamTaskID: "internal-ok", Status: VideoStatusSubmitted}}
	if err := svc.submitTask(context.Background(), adapter2, repo.providers[providerID], internalTask); err != nil {
		t.Fatalf("internal_real must skip session guard: %v", err)
	}
	if adapter2.calls() != 1 {
		t.Fatalf("adapter create calls = %d, want 1", adapter2.calls())
	}
}

func TestAPIKeySeedanceRealCreatesExcludeReviewOnlyAndRequireInternalReal(t *testing.T) {
	t.Setenv("SUB2API_VIDEO_REAL_SMOKE_ENABLED", "1")
	t.Setenv("SUB2API_VIDEO_REDACTED_EVENT_LOG", t.TempDir()+"/audit.log")
	t.Setenv("SUB2API_MEDIA_URL_ALLOWLIST", "provider.invalid")
	repo := newMemoryVideoGatewayRepo()
	reviewID := repo.nextProviderID
	repo.nextProviderID++
	repo.providers[reviewID] = &VideoProviderAccount{
		ID:               reviewID,
		Provider:         VideoProviderSeedance,
		DisplayName:      "Seedance Review Only",
		Enabled:          true,
		DefaultModel:     "doubao-seedance-2-0-260128",
		RouteAvailable:   true,
		APIKeyConfigured: true,
		EncryptedAPIKey:  "enc:review",
		Metadata: map[string]any{
			"review_only":             true,
			"production_authorized":   true,
			"single_smoke_authorized": true,
		},
	}
	formalID := seedSmokeAuthorizedSeedanceProvider(repo, "formal-key", "https://provider.invalid")
	repo.providers[formalID].Metadata["production_authorized"] = true
	repo.providers[formalID].RouteAvailable = true
	repo.providers[formalID].APIKeyConfigured = true

	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, &config.Config{})
	armPermissiveInternalRealPolicy(svc)

	t.Run("production skips review_only and stores internal_real", func(t *testing.T) {
		task, err := svc.CreateAPIKeySeedanceProductionTask(context.Background(), VideoProviderSeedance, VideoTaskCreateParams{
			TaskType:  VideoTaskTypeTextToVideo,
			Model:     "doubao-seedance-2-0-260128",
			Prompt:    "api-key production gate",
			Duration:  5,
			CreatedBy: 42,
		})
		if err != nil {
			t.Fatalf("CreateAPIKeySeedanceProductionTask: %v", err)
		}
		if task.ProviderAccountID != formalID {
			t.Fatalf("provider_account_id = %d, want formal %d (not review_only %d)", task.ProviderAccountID, formalID, reviewID)
		}
		if task.ExecutionMode != ExecutionModeInternalReal {
			t.Fatalf("execution_mode = %q, want %q", task.ExecutionMode, ExecutionModeInternalReal)
		}
	})

	t.Run("trial rejects when only review_only accounts exist", func(t *testing.T) {
		onlyReview := newMemoryVideoGatewayRepo()
		id := onlyReview.nextProviderID
		onlyReview.nextProviderID++
		onlyReview.providers[id] = &VideoProviderAccount{
			ID: id, Provider: VideoProviderSeedance, DisplayName: "Review", Enabled: true,
			DefaultModel: "doubao-seedance-2-0-260128", RouteAvailable: true, APIKeyConfigured: true,
			EncryptedAPIKey: "enc:review",
			Metadata:        map[string]any{"review_only": true, "single_smoke_authorized": true},
		}
		trialSvc := NewVideoGatewayService(onlyReview, noopVideoKeyEncryptor{}, &config.Config{})
		armPermissiveInternalRealPolicy(trialSvc)
		_, err := trialSvc.CreateAPIKeySeedanceTinyTrialTask(context.Background(), VideoProviderSeedance, VideoTaskCreateParams{
			TaskType: VideoTaskTypeTextToVideo, Model: "doubao-seedance-2-0-260128",
			Prompt: "must fail", Duration: 5, CreatedBy: 7,
		})
		if err == nil {
			t.Fatal("expected trial create to fail when only review_only accounts exist")
		}
	})
}
