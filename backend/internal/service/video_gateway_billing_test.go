package service

import (
	"context"
	"errors"
	"math"
	"strings"
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
