package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type budgetCall struct {
	userID int64
	cost   float64
	taskID int64
}

// mockBudgetGuard records gate/charge calls and can force a fail-closed rejection.
type mockBudgetGuard struct {
	checkErr    error
	checkCalls  []budgetCall
	chargeCalls []budgetCall
}

func (m *mockBudgetGuard) CheckBudget(_ context.Context, userID int64, cost float64) error {
	m.checkCalls = append(m.checkCalls, budgetCall{userID: userID, cost: cost})
	return m.checkErr
}

func (m *mockBudgetGuard) Charge(_ context.Context, userID int64, cost float64, taskID int64) error {
	m.chargeCalls = append(m.chargeCalls, budgetCall{userID: userID, cost: cost, taskID: taskID})
	return nil
}

func approxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func cfgWithCostPerSecond(rate float64) *config.Config {
	return &config.Config{VideoGateway: config.VideoGatewayConfig{CostPerSecond: rate}}
}

type billingSeedanceAdapter struct {
	result *VideoAdapterResult
}

func (a *billingSeedanceAdapter) Provider() string { return VideoProviderSeedance }

func (a *billingSeedanceAdapter) CreateTask(_ context.Context, _ *VideoProviderAccount, task *VideoTask) (*VideoAdapterResult, error) {
	return &VideoAdapterResult{UpstreamTaskID: "seedance-billing-" + strconv.FormatInt(task.ID, 10), Status: VideoStatusSubmitted}, nil
}

func (a *billingSeedanceAdapter) PollTask(_ context.Context, _ *VideoProviderAccount, _ *VideoTask) (*VideoAdapterResult, error) {
	return cloneVideoAdapterResult(a.result), nil
}

func (a *billingSeedanceAdapter) CancelTask(_ context.Context, _ *VideoProviderAccount, _ *VideoTask) (*VideoAdapterResult, error) {
	return &VideoAdapterResult{Status: VideoStatusCancelled}, nil
}

func (a *billingSeedanceAdapter) NormalizeStatus(upstream string) string {
	return normalizeVideoStatus(upstream)
}

func (a *billingSeedanceAdapter) BuildCreatePayload(_ *VideoProviderAccount, _ *VideoTask) map[string]any {
	return map[string]any{}
}

type videoBillingDeductCall struct {
	userID int64
	amount float64
}

type recordingVideoBillingUserRepo struct {
	UserRepository
	calls []videoBillingDeductCall
	err   error
}

func (r *recordingVideoBillingUserRepo) DeductBalance(_ context.Context, userID int64, amount float64) error {
	r.calls = append(r.calls, videoBillingDeductCall{userID: userID, amount: amount})
	return r.err
}

type videoBillingSettingRepo struct {
	values map[string]string
}

func (r *videoBillingSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	if v, ok := r.values[key]; ok {
		return &Setting{Key: key, Value: v}, nil
	}
	return nil, ErrSettingNotFound
}

func (r *videoBillingSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if v, ok := r.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (r *videoBillingSettingRepo) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *videoBillingSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if v, ok := r.values[key]; ok {
			out[key] = v
		}
	}
	return out, nil
}

func (r *videoBillingSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *videoBillingSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *videoBillingSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

type recordingVideoBillingCache struct {
	billingCacheWorkerStub
	mu    sync.Mutex
	calls []videoBillingDeductCall
}

func (c *recordingVideoBillingCache) DeductUserBalance(_ context.Context, userID int64, amount float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, videoBillingDeductCall{userID: userID, amount: amount})
	return nil
}

func (c *recordingVideoBillingCache) deductCalls() []videoBillingDeductCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]videoBillingDeductCall(nil), c.calls...)
}

func newVideoBalanceBillingDeps() (*recordingVideoBillingUserRepo, *recordingVideoBillingCache, *SettingService, *BillingCacheService) {
	userRepo := &recordingVideoBillingUserRepo{}
	cache := &recordingVideoBillingCache{}
	settingSvc := NewSettingService(&videoBillingSettingRepo{values: map[string]string{
		SettingKeyUSDCNYRate: "7.20",
	}}, &config.Config{})
	cacheSvc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{})
	return userRepo, cache, settingSvc, cacheSvc
}

func TestSeedanceActualCostUsesProviderUsageTokens(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := seedSmokeAuthorizedSeedanceProvider(repo, "seedance-billing-test-key", "https://ark.example.test")
	tokens := int64(102960)
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	svc.adapters[VideoProviderSeedance] = &billingSeedanceAdapter{result: &VideoAdapterResult{
		Status:           VideoStatusSucceeded,
		ResultURL:        "https://ark-content.cn-beijing.volces.com/v/ok.mp4",
		UsageTotalTokens: &tokens,
		ActualResolution: "720p",
	}}
	guard := &mockBudgetGuard{}
	svc.SetBudgetGuard(guard)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Model:             "doubao-seedance-2-0-260128",
		Prompt:            "bill by tokens",
		Duration:          5,
		Resolution:        "720p",
		CreatedBy:         8,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for range 2 {
		if err := svc.ProcessRunnableTasks(ctx, 10, time.Minute); err != nil {
			t.Fatalf("process: %v", err)
		}
	}
	got, _, err := svc.GetTask(ctx, task.ID, 8, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	want := 46.0 * 0.10296
	if !approxEqual(got.CostEstimate, want) {
		t.Fatalf("actual cost = %.8f, want %.8f", got.CostEstimate, want)
	}
	if len(repo.usage) != 1 || !approxEqual(repo.usage[0].CostEstimate, want) {
		t.Fatalf("usage log cost = %#v, want %.8f", repo.usage, want)
	}
	if len(guard.chargeCalls) != 1 || !approxEqual(guard.chargeCalls[0].cost, want) {
		t.Fatalf("charge calls = %#v, want cost %.8f", guard.chargeCalls, want)
	}
}

func TestSeedanceSucceededTaskDeductsUserBalanceInUSDAndQueuesCache(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := seedSmokeAuthorizedSeedanceProvider(repo, "seedance-billing-test-key", "https://ark.example.test")
	tokens := int64(102960)
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	svc.adapters[VideoProviderSeedance] = &billingSeedanceAdapter{result: &VideoAdapterResult{
		Status:           VideoStatusSucceeded,
		ResultURL:        "https://ark-content.cn-beijing.volces.com/v/ok.mp4",
		UsageTotalTokens: &tokens,
		ActualResolution: "720p",
	}}
	userRepo, cache, settingSvc, cacheSvc := newVideoBalanceBillingDeps()
	defer cacheSvc.Stop()
	svc.userRepo = userRepo
	svc.settingService = settingSvc
	svc.billingCacheService = cacheSvc

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Model:             "doubao-seedance-2-0-260128",
		Prompt:            "deduct balance by tokens",
		Duration:          5,
		Resolution:        "720p",
		CreatedBy:         8,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for range 2 {
		if err := svc.ProcessRunnableTasks(ctx, 10, time.Minute); err != nil {
			t.Fatalf("process: %v", err)
		}
	}
	cacheSvc.Stop()

	wantUSD := (46.0 * 0.10296) / 7.20
	if len(userRepo.calls) != 1 || userRepo.calls[0].userID != 8 || !approxEqual(userRepo.calls[0].amount, wantUSD) {
		t.Fatalf("DeductBalance calls = %#v, want user=8 amount %.8f", userRepo.calls, wantUSD)
	}
	cacheCalls := cache.deductCalls()
	if len(cacheCalls) != 1 || cacheCalls[0].userID != 8 || !approxEqual(cacheCalls[0].amount, wantUSD) {
		t.Fatalf("QueueDeductBalance cache calls = %#v, want user=8 amount %.8f", cacheCalls, wantUSD)
	}
	if _, ok := repo.balanceClaims[task.ID]; !ok {
		t.Fatalf("expected task %d balance charge to be claimed", task.ID)
	}
}

func TestSeedanceBalanceChargeIsIdempotentOnWorkerRetry(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	task := &VideoTask{
		ID:               42,
		Provider:         VideoProviderSeedance,
		Model:            "doubao-seedance-2-0-260128",
		Status:           VideoStatusSucceeded,
		UsageTotalTokens: videoInt64Ptr(102960),
		Currency:         BillingCurrencyCNY,
		CreatedBy:        8,
	}
	repo.tasks[task.ID] = cloneVideoTask(task)
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	userRepo, cache, settingSvc, cacheSvc := newVideoBalanceBillingDeps()
	defer cacheSvc.Stop()
	svc.userRepo = userRepo
	svc.settingService = settingSvc
	svc.billingCacheService = cacheSvc

	svc.chargeForVideo(ctx, task)
	svc.chargeForVideo(ctx, task)
	cacheSvc.Stop()

	if len(userRepo.calls) != 1 {
		t.Fatalf("DeductBalance must be idempotent, calls=%#v", userRepo.calls)
	}
	if cacheCalls := cache.deductCalls(); len(cacheCalls) != 1 {
		t.Fatalf("cache balance deduction must be idempotent, calls=%#v", cacheCalls)
	}
}

func TestSeedanceBalanceChargeReleasesClaimAfterDeductFailure(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	task := &VideoTask{
		ID:               45,
		Provider:         VideoProviderSeedance,
		Model:            "doubao-seedance-2-0-260128",
		Status:           VideoStatusSucceeded,
		UsageTotalTokens: videoInt64Ptr(102960),
		Currency:         BillingCurrencyCNY,
		CreatedBy:        8,
	}
	repo.tasks[task.ID] = cloneVideoTask(task)
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	userRepo, cache, settingSvc, cacheSvc := newVideoBalanceBillingDeps()
	defer cacheSvc.Stop()
	userRepo.err = errors.New("deduct temporarily unavailable")
	svc.userRepo = userRepo
	svc.settingService = settingSvc
	svc.billingCacheService = cacheSvc

	svc.chargeForVideo(ctx, task)
	if _, ok := repo.balanceClaims[task.ID]; ok {
		t.Fatalf("deduct failure must release balance claim for task %d", task.ID)
	}

	userRepo.err = nil
	svc.chargeForVideo(ctx, task)
	cacheSvc.Stop()

	if len(userRepo.calls) != 2 {
		t.Fatalf("expected failed attempt plus retry deduction, calls=%#v", userRepo.calls)
	}
	if cacheCalls := cache.deductCalls(); len(cacheCalls) != 1 {
		t.Fatalf("only successful retry should enqueue cache deduction, calls=%#v", cacheCalls)
	}
	if _, ok := repo.balanceClaims[task.ID]; !ok {
		t.Fatalf("successful retry should leave balance charge claimed")
	}
}

func TestProcessRunnableTasksRetriesUnchargedSucceededBilling(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	task := &VideoTask{
		ID:               46,
		Provider:         VideoProviderSeedance,
		Model:            "doubao-seedance-2-0-260128",
		Status:           VideoStatusSucceeded,
		UsageTotalTokens: videoInt64Ptr(102960),
		Currency:         BillingCurrencyCNY,
		CreatedBy:        8,
	}
	repo.tasks[task.ID] = cloneVideoTask(task)
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	userRepo, cache, settingSvc, cacheSvc := newVideoBalanceBillingDeps()
	defer cacheSvc.Stop()
	svc.userRepo = userRepo
	svc.settingService = settingSvc
	svc.billingCacheService = cacheSvc

	if err := svc.ProcessRunnableTasks(ctx, 10, time.Minute); err != nil {
		t.Fatalf("process runnable tasks: %v", err)
	}
	cacheSvc.Stop()

	if len(userRepo.calls) != 1 {
		t.Fatalf("succeeded uncharged task should be retried for billing, calls=%#v", userRepo.calls)
	}
	if cacheCalls := cache.deductCalls(); len(cacheCalls) != 1 {
		t.Fatalf("successful retry should enqueue cache deduction, calls=%#v", cacheCalls)
	}
}

func TestSeedanceFailedTaskDoesNotDeductBalance(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	task := &VideoTask{
		ID:               43,
		Provider:         VideoProviderSeedance,
		Model:            "doubao-seedance-2-0-260128",
		Status:           VideoStatusFailed,
		UsageTotalTokens: videoInt64Ptr(102960),
		Currency:         BillingCurrencyCNY,
		CreatedBy:        8,
	}
	repo.tasks[task.ID] = cloneVideoTask(task)
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	userRepo, cache, settingSvc, cacheSvc := newVideoBalanceBillingDeps()
	defer cacheSvc.Stop()
	svc.userRepo = userRepo
	svc.settingService = settingSvc
	svc.billingCacheService = cacheSvc

	svc.chargeForVideo(ctx, task)
	cacheSvc.Stop()

	if len(userRepo.calls) != 0 {
		t.Fatalf("failed task must not deduct user balance, calls=%#v", userRepo.calls)
	}
	if cacheCalls := cache.deductCalls(); len(cacheCalls) != 0 {
		t.Fatalf("failed task must not deduct balance cache, calls=%#v", cacheCalls)
	}
	if _, ok := repo.balanceClaims[task.ID]; ok {
		t.Fatalf("failed task must not claim balance charge")
	}
}

func TestSeedanceActualCostSelectsVideoInputRate(t *testing.T) {
	tokens := int64(102960)
	svc := NewVideoGatewayService(newMemoryVideoGatewayRepo(), noopVideoKeyEncryptor{}, nil)
	cost := svc.calculateVideoActualCost(&VideoTask{
		Provider:         VideoProviderSeedance,
		Model:            "doubao-seedance-2-0-260128",
		Status:           VideoStatusSucceeded,
		UsageTotalTokens: &tokens,
		HasVideoInput:    true,
	})
	want := 28.0 * 0.10296
	if !approxEqual(cost, want) {
		t.Fatalf("video-input cost = %.8f, want %.8f", cost, want)
	}
}

func videoInt64Ptr(v int64) *int64 {
	return &v
}

func TestSeedanceFailedTaskCostsZeroEvenWithUsageTokens(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := seedSmokeAuthorizedSeedanceProvider(repo, "seedance-billing-test-key", "https://ark.example.test")
	tokens := int64(102960)
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	svc.adapters[VideoProviderSeedance] = &billingSeedanceAdapter{result: &VideoAdapterResult{
		Status:           VideoStatusFailed,
		ErrorMessage:     "provider failed",
		UsageTotalTokens: &tokens,
		CostEstimate:     99,
	}}
	guard := &mockBudgetGuard{}
	svc.SetBudgetGuard(guard)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Model:             "doubao-seedance-2-0-260128",
		Prompt:            "failed task is free",
		Duration:          5,
		Resolution:        "720p",
		CreatedBy:         8,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for range 2 {
		if err := svc.ProcessRunnableTasks(ctx, 10, time.Minute); err != nil {
			t.Fatalf("process: %v", err)
		}
	}
	got, _, err := svc.GetTask(ctx, task.ID, 8, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != VideoStatusFailed || got.CostEstimate != 0 {
		t.Fatalf("failed task status/cost = %s/%.4f, want failed/0", got.Status, got.CostEstimate)
	}
	if len(repo.usage) != 1 || repo.usage[0].CostEstimate != 0 {
		t.Fatalf("usage log cost = %#v, want zero", repo.usage)
	}
	if len(guard.chargeCalls) != 0 {
		t.Fatalf("failed task must not charge, got %#v", guard.chargeCalls)
	}
}

func TestSeedanceBudgetEstimateUsesReferenceTableWhenConfigRateUnset(t *testing.T) {
	svc := NewVideoGatewayService(newMemoryVideoGatewayRepo(), noopVideoKeyEncryptor{}, nil)
	cost := svc.estimateVideoCost(&VideoTask{
		Provider:   VideoProviderSeedance,
		Model:      "doubao-seedance-2-0-260128",
		Duration:   5,
		Resolution: "720p",
	})
	if !approxEqual(cost, 5.0) {
		t.Fatalf("720p 5s estimate = %.4f, want 5.0", cost)
	}
	fastCost := svc.estimateVideoCost(&VideoTask{
		Provider:   VideoProviderSeedance,
		Model:      "doubao-seedance-2-0-fast",
		Duration:   5,
		Resolution: "720p",
	})
	if !approxEqual(fastCost, 4.0) {
		t.Fatalf("fast 720p 5s estimate = %.4f, want 4.0", fastCost)
	}
}

func TestKlingBudgetEstimateUsesPerSecondCatalogWhenConfigRateUnset(t *testing.T) {
	svc := NewVideoGatewayService(newMemoryVideoGatewayRepo(), noopVideoKeyEncryptor{}, nil)

	stdCost := svc.estimateVideoCost(&VideoTask{
		Provider:   VideoProviderKling,
		Model:      "kling-v1",
		Duration:   5,
		Resolution: "720p", // std mode
	})
	if stdCost <= 0 {
		t.Fatalf("kling std estimate must be non-zero, got %.4f", stdCost)
	}

	proCost := svc.estimateVideoCost(&VideoTask{
		Provider:   VideoProviderKling,
		Model:      "kling-2.6-pro",
		Duration:   5,
		Resolution: "1080p", // pro mode
	})
	if proCost <= 0 {
		t.Fatalf("kling pro estimate must be non-zero, got %.4f", proCost)
	}
	if !(proCost > stdCost) {
		t.Fatalf("kling pro 5s estimate (%.4f) should exceed std 5s (%.4f)", proCost, stdCost)
	}

	task := &VideoTask{
		Provider:   VideoProviderKling,
		Model:      "kling-v1",
		Duration:   5,
		Resolution: "720p",
	}
	svc.applyVideoBillingMetadata(task)
	if task.PricingVersion != VideoPricingVersionKling202607 {
		t.Fatalf("PricingVersion = %q, want %q", task.PricingVersion, VideoPricingVersionKling202607)
	}
	if task.Currency != BillingCurrencyCNY {
		t.Fatalf("Currency = %q, want CNY", task.Currency)
	}
}

func TestKlingSucceededTaskSettlesPerSecondCostAndCarriesPricingVersion(t *testing.T) {
	svc := NewVideoGatewayService(newMemoryVideoGatewayRepo(), noopVideoKeyEncryptor{}, nil)
	task := &VideoTask{
		Provider:   VideoProviderKling,
		Model:      "kling-v2-6",
		Status:     VideoStatusSucceeded,
		Duration:   5,
		Resolution: "720p",
		CreatedBy:  8,
	}

	cost := svc.chargeableVideoCost(task)
	if cost <= 0 {
		t.Fatalf("kling settle cost must be non-zero, got %.4f", cost)
	}

	svc.applyVideoBillingMetadata(task)
	if task.PricingVersion != VideoPricingVersionKling202607 {
		t.Fatalf("PricingVersion = %q, want %q", task.PricingVersion, VideoPricingVersionKling202607)
	}
	if task.Currency != BillingCurrencyCNY {
		t.Fatalf("Currency = %q, want CNY", task.Currency)
	}

	// Settle must match estimate for the same task shape (Kling has no token usage).
	estimate := svc.estimateVideoCost(task)
	if !approxEqual(cost, estimate) {
		t.Fatalf("settle=%.8f estimate=%.8f; kling settle should equal per-second estimate", cost, estimate)
	}
}

func TestKlingCatalogRateCNYPerSecondSelectsStdAndPro(t *testing.T) {
	catalog := NewVideoPricingCatalog(nil)
	stdRate, version, ok := catalog.RateCNYPerSecond(&VideoTask{
		Provider:   VideoProviderKling,
		Model:      "kling-v1",
		Resolution: "720p",
	})
	if !ok || stdRate <= 0 {
		t.Fatalf("std rate ok=%v rate=%.4f", ok, stdRate)
	}
	if version != VideoPricingVersionKling202607 {
		t.Fatalf("version = %q, want %q", version, VideoPricingVersionKling202607)
	}

	proRate, _, ok := catalog.RateCNYPerSecond(&VideoTask{
		Provider:   VideoProviderKling,
		Model:      "kling-v1",
		Resolution: "1080p",
	})
	if !ok || proRate <= 0 {
		t.Fatalf("pro rate ok=%v rate=%.4f", ok, proRate)
	}
	if !(proRate > stdRate) {
		t.Fatalf("pro rate %.4f should exceed std rate %.4f", proRate, stdRate)
	}
}

// TestVideoBudgetGateAllowsWhenAffordable: sufficient budget => create proceeds and
// the gate was consulted with the duration-based estimated cost.
func TestVideoBudgetGateAllowsWhenAffordable(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfgWithCostPerSecond(0.5))
	guard := &mockBudgetGuard{} // checkErr nil => affordable
	svc.SetBudgetGuard(guard)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "affordable render",
		CreatedBy:         3,
		// Duration 0 => default 5s; cost = 0.5 × 5 = 2.5
	})
	if err != nil {
		t.Fatalf("expected create to proceed, got %v", err)
	}
	if task == nil || task.ID == 0 {
		t.Fatal("expected a persisted task")
	}
	if len(repo.tasks) != 1 {
		t.Fatalf("expected one task persisted, got %d", len(repo.tasks))
	}
	if len(guard.checkCalls) != 1 {
		t.Fatalf("expected budget gate consulted once, got %d", len(guard.checkCalls))
	}
	if guard.checkCalls[0].userID != 3 || !approxEqual(guard.checkCalls[0].cost, 2.5) {
		t.Fatalf("expected gate(user=3, cost=2.5), got %+v", guard.checkCalls[0])
	}
}

// TestVideoBudgetGateRejectsFailClosed: insufficient budget => the gate's error is
// propagated, NO task is persisted and NO provider dispatch / charge occurs.
func TestVideoBudgetGateRejectsFailClosed(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfgWithCostPerSecond(0.5))
	denied := errors.New("INSUFFICIENT_BUDGET: over per-call cap")
	guard := &mockBudgetGuard{checkErr: denied}
	svc.SetBudgetGuard(guard)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "over-budget render",
		CreatedBy:         4,
	})
	if err == nil {
		t.Fatal("expected fail-closed rejection, got nil error")
	}
	if !errors.Is(err, denied) {
		t.Fatalf("expected the gate's denial error to propagate, got %v", err)
	}
	if task != nil {
		t.Fatalf("expected no task on rejection, got %#v", task)
	}
	if len(repo.tasks) != 0 {
		t.Fatalf("fail-closed must NOT persist a task; got %d persisted", len(repo.tasks))
	}
	if len(guard.chargeCalls) != 0 {
		t.Fatalf("rejected create must not charge; got %d charges", len(guard.chargeCalls))
	}
}

// TestVideoBudgetChargesOnSuccess: a delivered (succeeded) task triggers a single
// charge for the estimated cost against the creating user.
func TestVideoBudgetChargesOnSuccess(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfgWithCostPerSecond(2.0))
	guard := &mockBudgetGuard{}
	svc.SetBudgetGuard(guard)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "render and bill me",
		CreatedBy:         8,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// mock provider: queued -> submitted -> running -> succeeded across 3 ticks.
	for range 3 {
		if err := svc.ProcessRunnableTasks(ctx, 10, time.Minute); err != nil {
			t.Fatalf("process: %v", err)
		}
	}
	got, _, err := svc.GetTask(ctx, task.ID, 8, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != VideoStatusSucceeded {
		t.Fatalf("expected succeeded, got %s", got.Status)
	}
	if len(guard.chargeCalls) != 1 {
		t.Fatalf("expected exactly one charge on success, got %d", len(guard.chargeCalls))
	}
	c := guard.chargeCalls[0]
	if c.userID != 8 || c.taskID != task.ID || !approxEqual(c.cost, 10.0) { // 2.0 × 5s
		t.Fatalf("expected charge(user=8, task=%d, cost=10.0), got %+v", task.ID, c)
	}
}

// TestStaticBudgetGuardCheckBudget exercises the concrete interception primitive
// directly: an unconfigured cap fails closed, a cost over the cap is rejected, and a
// cost at or under the cap passes. This nails the comparison logic itself (the existing
// mockBudgetGuard only returns a canned error and never compares cost vs budget).
func TestStaticBudgetGuardCheckBudget(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name          string
		perCallBudget float64
		cost          float64
		wantErr       bool
	}{
		{"unconfigured cap fails closed", 0, 0, true},
		{"negative cap fails closed", -1, 0, true},
		{"NaN cap fails closed", math.NaN(), 1.0, true},
		{"Inf cap fails closed", math.Inf(1), 1.0, true},
		{"NaN cost fails closed", 3.0, math.NaN(), true},
		{"Inf cost fails closed", 3.0, math.Inf(1), true},
		{"cost over cap rejected", 3.0, 5.0, true},
		{"cost under cap allowed", 3.0, 2.5, false},
		{"cost equal to cap allowed (boundary)", 3.0, 3.0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewStaticBudgetGuard(tc.perCallBudget)
			err := g.CheckBudget(ctx, 1, tc.cost)
			if tc.wantErr && err == nil {
				t.Fatalf("expected rejection for cost=%v cap=%v, got nil", tc.cost, tc.perCallBudget)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected pass for cost=%v cap=%v, got %v", tc.cost, tc.perCallBudget, err)
			}
		})
	}
}

// TestVideoBudgetGateInterceptsWhenCostExceedsBudget is the dry-run interception proof:
// with a real StaticBudgetGuard armed at a per-call cap, a create whose duration-based
// estimate exceeds the cap is rejected by the gate at CreateTask — no task is persisted
// (the request never enters the provider call path). No real provider/key/DB is touched
// (memory repo + mock provider).
func TestVideoBudgetGateInterceptsWhenCostExceedsBudget(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfgWithCostPerSecond(0.5))
	svc.SetBudgetGuard(NewStaticBudgetGuard(3.0)) // per-call cap 3.0

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "over-cap render",
		CreatedBy:         5,
		Duration:          10, // cost = 0.5 × 10 = 5.0 > cap 3.0 => intercepted
	})
	if err == nil {
		t.Fatal("expected the budget gate to intercept, got nil error")
	}
	if !strings.Contains(err.Error(), "exceeds per-call cap") {
		t.Fatalf("expected an over-cap interception error, got %v", err)
	}
	if task != nil {
		t.Fatalf("expected no task on interception, got %#v", task)
	}
	if len(repo.tasks) != 0 {
		t.Fatalf("intercepted create must NOT persist a task (no call path); got %d persisted", len(repo.tasks))
	}
}

// TestVideoBudgetGateAllowsWithinBudget is the complementary pass-through proof: the SAME
// guard config (per-call cap 3.0) admits a create whose estimate is within budget, and
// the task is persisted. Paired with the interception test above, this proves the gate
// decides on the actual cost-vs-budget comparison, not on guard presence alone.
func TestVideoBudgetGateAllowsWithinBudget(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfgWithCostPerSecond(0.5))
	svc.SetBudgetGuard(NewStaticBudgetGuard(3.0)) // same per-call cap 3.0

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "within-cap render",
		CreatedBy:         6,
		Duration:          4, // cost = 0.5 × 4 = 2.0 <= cap 3.0 => allowed
	})
	if err != nil {
		t.Fatalf("expected create within budget to proceed, got %v", err)
	}
	if task == nil || task.ID == 0 {
		t.Fatal("expected a persisted task")
	}
	if len(repo.tasks) != 1 {
		t.Fatalf("expected one task persisted, got %d", len(repo.tasks))
	}
}

// cfgWithVideoBudget builds a config with BOTH the VA1 per-second rate and the per-call
// budget cap set, to exercise the production DI wiring (ProvideVideoGatewayService).
func cfgWithVideoBudget(rate, perCallBudget float64) *config.Config {
	return &config.Config{VideoGateway: config.VideoGatewayConfig{CostPerSecond: rate, PerCallBudget: perCallBudget}}
}

// TestProvideVideoGatewayServiceArmsGuardFromConfig proves the PRODUCTION wiring path:
// ProvideVideoGatewayService (the single DI seam, also called by wire_gen.go) reads
// video_gateway.per_call_budget and injects a StaticBudgetGuard armed at that cap. With
// cost_per_second=1.5 and per_call_budget=2 a 5s create estimates 7.5 > 2 and is rejected
// at CreateTask — no task persisted, no provider/key/DB touched. This is the unit-level
// mirror of the B-1 "empty-brake" (空踩刹车) check against the real DI wrapper.
func TestProvideVideoGatewayServiceArmsGuardFromConfig(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := ProvideVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfgWithVideoBudget(1.5, 2.0), nil, nil, nil)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "empty-brake render",
		CreatedBy:         7,
		Duration:          5, // cost = 1.5 × 5 = 7.5 > cap 2.0 => intercepted
	})
	if err == nil {
		t.Fatal("expected ProvideVideoGatewayService to arm the guard and intercept, got nil error")
	}
	if !strings.Contains(err.Error(), "exceeds per-call cap") {
		t.Fatalf("expected an over-cap interception error, got %v", err)
	}
	if task != nil {
		t.Fatalf("expected no task on interception, got %#v", task)
	}
	if len(repo.tasks) != 0 {
		t.Fatalf("armed guard must NOT persist an over-cap task; got %d persisted", len(repo.tasks))
	}
}

// TestProvideVideoGatewayServiceUnarmedWhenBudgetZero proves the default is inert: with
// per_call_budget=0 the wiring injects NO guard, so the gateway behaves exactly as before
// (the create proceeds and is persisted). This guarantees installing the field changes no
// production behaviour until a real cap is set.
func TestProvideVideoGatewayServiceUnarmedWhenBudgetZero(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := ProvideVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfgWithVideoBudget(1.5, 0), nil, nil, nil)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "unarmed render",
		CreatedBy:         7,
		Duration:          5,
	})
	if err != nil {
		t.Fatalf("expected unarmed gate to allow create, got %v", err)
	}
	if task == nil || task.ID == 0 {
		t.Fatal("expected a persisted task when the gate is unarmed")
	}
	if len(repo.tasks) != 1 {
		t.Fatalf("expected one task persisted, got %d", len(repo.tasks))
	}
}

// TestProvideVideoGatewayServiceAdmitsNormalClipAtRealCap proves the B-2 production config
// (cost_per_second=1.5, per_call_budget=30) admits a normal single 5s clip: estimate 7.5 <=
// cap 30 => allowed and persisted. The brake stops over-budget bursts, not the intended
// single smoke.
func TestProvideVideoGatewayServiceAdmitsNormalClipAtRealCap(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryVideoGatewayRepo()
	providerID := repo.seedMockProvider()
	svc := ProvideVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfgWithVideoBudget(1.5, 30.0), nil, nil, nil)

	task, err := svc.CreateTask(ctx, VideoTaskCreateParams{
		ProviderAccountID: providerID,
		TaskType:          VideoTaskTypeTextToVideo,
		Prompt:            "real-cap single clip",
		CreatedBy:         7,
		Duration:          5, // cost = 1.5 × 5 = 7.5 <= cap 30 => allowed
	})
	if err != nil {
		t.Fatalf("expected the real cap (30) to admit a normal 5s clip, got %v", err)
	}
	if task == nil || task.ID == 0 {
		t.Fatal("expected a persisted task within the real cap")
	}
	if len(repo.tasks) != 1 {
		t.Fatalf("expected one task persisted, got %d", len(repo.tasks))
	}
}
