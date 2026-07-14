package reviewguard

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/shopspring/decimal"
)

func TestRealCreateGuardDefaultsClosedInNormalBuilds(t *testing.T) {
	guard := NewFailClosedGuard()
	err := guard.Reserve(context.Background(), RealCreateReservation{
		OperationID:    "op-1",
		Kind:           RealCreateImage,
		ReservedCNY:    decimal.NewFromInt(1),
		PricingSource:  "test",
		PricingVersion: "v1",
	})
	if err == nil {
		t.Fatal("expected fail-closed guard to reject real review creates")
	}
	if !strings.Contains(err.Error(), "REAL_REVIEW_SESSION_DISABLED") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRealCreateGuardRequiresAbsoluteStatePathWhenEnabled(t *testing.T) {
	guard := NewSessionGuard("relative/state.json")
	err := guard.Reserve(context.Background(), sampleReservation("op-abs", RealCreateVideo, "5"))
	if err == nil {
		t.Fatal("expected relative state path to fail closed")
	}
	if !strings.Contains(err.Error(), "REAL_REVIEW_STATE_PATH_REQUIRED") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRealCreateGuardRejectsCorruptStateFailClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	guard := NewSessionGuard(path)
	if err := guard.Reserve(context.Background(), sampleReservation("op-corrupt", RealCreateVideo, "1")); err == nil {
		t.Fatal("expected corrupt state to fail closed")
	}
}

func TestRealCreateGuardRejectsFifthImageVideoAndBudgetOverSixty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	guard := NewSessionGuard(path)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		if err := guard.Reserve(ctx, sampleReservation("img-"+strconv.Itoa(i), RealCreateImage, "5")); err != nil {
			t.Fatalf("reserve image %d: %v", i+1, err)
		}
		if err := guard.Reserve(ctx, sampleReservation("vid-"+strconv.Itoa(i), RealCreateVideo, "5")); err != nil {
			t.Fatalf("reserve video %d: %v", i+1, err)
		}
	}
	if err := guard.Reserve(ctx, sampleReservation("img-5", RealCreateImage, "1")); err == nil {
		t.Fatal("expected fifth image to be rejected")
	}
	if err := guard.Reserve(ctx, sampleReservation("vid-5", RealCreateVideo, "1")); err == nil {
		t.Fatal("expected fifth video to be rejected")
	}

	budgetPath := filepath.Join(t.TempDir(), "budget.json")
	budgetGuard := NewSessionGuard(budgetPath)
	if err := budgetGuard.Reserve(ctx, sampleReservation("budget-full", RealCreateVideo, "60")); err != nil {
		t.Fatalf("reserve full budget: %v", err)
	}
	if err := budgetGuard.Reserve(ctx, sampleReservation("budget-over", RealCreateImage, "0.01")); err == nil {
		t.Fatal("expected cumulative budget overflow to be rejected")
	}
}

func TestRealCreateGuardConcurrentReserveCannotOverspend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	guard := NewSessionGuard(path)
	ctx := context.Background()
	var wg sync.WaitGroup
	var admitted atomic.Int32
	for i := 0; i < 20; i++ {
		wg.Add(1)
		opID := "concurrent-" + strconv.Itoa(i)
		go func() {
			defer wg.Done()
			if guard.Reserve(ctx, sampleReservation(opID, RealCreateVideo, "10")) == nil {
				admitted.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := admitted.Load(); got != 4 {
		t.Fatalf("admitted videos = %d, want 4", got)
	}
	snap, err := guard.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.VideoUsed != 4 || !snap.ReservedCNY.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("snapshot = %+v, want 4 videos and 40 CNY", snap)
	}
}

func TestRealCreateGuardOperationIDIdempotentWithoutDoubleCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	guard := NewSessionGuard(path)
	ctx := context.Background()
	res := sampleReservation("same-op", RealCreateImage, "5")
	if err := guard.Reserve(ctx, res); err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if err := guard.Reserve(ctx, res); err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	snap, err := guard.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ImageUsed != 1 || !snap.ReservedCNY.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("double-count detected: %+v", snap)
	}
}

func TestRealCreateGuardMismatchedOperationParamsFailClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	guard := NewSessionGuard(path)
	ctx := context.Background()
	if err := guard.Reserve(ctx, sampleReservation("same-op", RealCreateImage, "5")); err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	mismatch := sampleReservation("same-op", RealCreateImage, "6")
	err := guard.Reserve(ctx, mismatch)
	if err == nil {
		t.Fatal("expected mismatched params to fail closed")
	}
	if !strings.Contains(err.Error(), "REAL_REVIEW_IDEMPOTENCY_MISMATCH") {
		t.Fatalf("unexpected error: %v", err)
	}
	snap, err := guard.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ImageUsed != 1 || !snap.ReservedCNY.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("mismatch must not mutate counters: %+v", snap)
	}
}

func TestRealCreateGuardAcceptsLegacyFloatReservedCNY(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	if err := os.WriteFile(path, []byte(`{"image_attempts":1,"video_attempts":2,"reserved_cny":20}`), 0600); err != nil {
		t.Fatal(err)
	}
	guard := NewSessionGuard(path)
	snap, err := guard.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.ImageUsed != 1 || snap.VideoUsed != 2 || !snap.ReservedCNY.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("legacy float state = %+v", snap)
	}
	if snap.ImageRemaining != 3 || snap.VideoRemaining != 2 || !snap.RemainingCNY.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("remaining counters = %+v", snap)
	}
}

func TestRealCreateGuardPersistsReservedCNYAsDecimalString(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.json")
	guard := NewSessionGuard(path)
	if err := guard.Reserve(context.Background(), sampleReservation("persist-1", RealCreateVideo, "10.5")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	var reserved any
	if err := json.Unmarshal(decoded["reserved_cny"], &reserved); err != nil {
		t.Fatal(err)
	}
	if _, ok := reserved.(string); !ok {
		t.Fatalf("reserved_cny must persist as string, got %T (%v)", reserved, reserved)
	}
}

func TestRealCreateGuardHelperProcess(t *testing.T) {
	if os.Getenv("SUB2API_REAL_CREATE_HELPER") != "1" {
		return
	}
	cost := os.Getenv("SUB2API_REAL_CREATE_HELPER_COST")
	state := os.Getenv("SUB2API_REAL_CREATE_HELPER_STATE")
	op := os.Getenv("SUB2API_REAL_CREATE_HELPER_OP")
	err := NewSessionGuard(state).Reserve(context.Background(), sampleReservation(op, RealCreateVideo, cost))
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestRealCreateGuardCompetingProcessesEnforceLimits(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "review.json")
	wantAdmitted := 4
	processes := 12
	commands := make([]*exec.Cmd, 0, processes)
	for i := 0; i < processes; i++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRealCreateGuardHelperProcess$")
		cmd.Env = append(os.Environ(),
			"SUB2API_REAL_CREATE_HELPER=1",
			"SUB2API_REAL_CREATE_HELPER_STATE="+statePath,
			"SUB2API_REAL_CREATE_HELPER_COST=10",
			"SUB2API_REAL_CREATE_HELPER_OP=proc-"+strconv.Itoa(i),
		)
		if err := cmd.Start(); err != nil {
			t.Fatalf("start helper %d: %v", i, err)
		}
		commands = append(commands, cmd)
	}
	admitted := 0
	for _, cmd := range commands {
		if err := cmd.Wait(); err == nil {
			admitted++
		}
	}
	if admitted != wantAdmitted {
		t.Fatalf("admitted processes = %d, want %d", admitted, wantAdmitted)
	}
	snap, err := NewSessionGuard(statePath).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.VideoUsed != wantAdmitted || !snap.ReservedCNY.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("persisted snapshot = %+v", snap)
	}
}

func sampleReservation(opID string, kind RealCreateKind, cny string) RealCreateReservation {
	return RealCreateReservation{
		OperationID:    opID,
		Kind:           kind,
		ReservedCNY:    decimal.RequireFromString(cny),
		PricingSource:  "test-source",
		PricingVersion: "test-version",
	}
}
