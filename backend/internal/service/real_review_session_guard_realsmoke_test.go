//go:build realsmoke

package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRealReviewSessionGuardDefaultsClosed(t *testing.T) {
	guard := newRealReviewSessionGuard("", false)
	if err := guard.Reserve(realReviewImage, 1); err == nil {
		t.Fatal("expected disabled guard to fail closed")
	}
}

func TestRealReviewSessionGuardLimitsAttemptsAndBudgetBeforeCall(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "review.json")
	guard := newRealReviewSessionGuard(statePath, true)
	for i := 0; i < 4; i++ {
		if err := guard.Reserve(realReviewImage, 5); err != nil {
			t.Fatalf("reserve image %d: %v", i+1, err)
		}
		if err := guard.Reserve(realReviewVideo, 5); err != nil {
			t.Fatalf("reserve video %d: %v", i+1, err)
		}
	}

	var calls atomic.Int32
	if err := guard.ReserveBefore(realReviewImage, 1, func() { calls.Add(1) }); err == nil {
		t.Fatal("expected fifth image to be rejected")
	}
	if err := guard.ReserveBefore(realReviewVideo, 1, func() { calls.Add(1) }); err == nil {
		t.Fatal("expected fifth video to be rejected")
	}
	if calls.Load() != 0 {
		t.Fatalf("rejected reservations reached call boundary %d times", calls.Load())
	}

	budgetGuard := newRealReviewSessionGuard(filepath.Join(t.TempDir(), "budget.json"), true)
	if err := budgetGuard.Reserve(realReviewVideo, 60); err != nil {
		t.Fatalf("reserve full budget: %v", err)
	}
	if err := budgetGuard.ReserveBefore(realReviewImage, 0.01, func() { calls.Add(1) }); err == nil {
		t.Fatal("expected cumulative budget overflow to be rejected")
	}
	if calls.Load() != 0 {
		t.Fatal("budget rejection must happen before callback/socket boundary")
	}
}

func TestRealReviewSessionGuardConcurrentReserveIsAtomicAndPersistsFailures(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "review.json")
	guard := newRealReviewSessionGuard(statePath, true)
	var wg sync.WaitGroup
	var admitted atomic.Int32
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if guard.Reserve(realReviewVideo, 10) == nil {
				admitted.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := admitted.Load(); got != 4 {
		t.Fatalf("admitted videos = %d, want 4", got)
	}

	reopened := newRealReviewSessionGuard(statePath, true)
	if err := reopened.Reserve(realReviewVideo, 1); err == nil {
		t.Fatal("persisted attempts must block a later process/guard")
	}
	state, err := reopened.loadStateForTest()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state.VideoAttempts != 4 || state.ReservedCNY != 40 {
		t.Fatalf("state = %#v, want 4 attempts and 40 CNY permanently reserved", state)
	}
}

func TestRealReviewSessionGuardRejectsCorruptOrMissingStatePath(t *testing.T) {
	if err := newRealReviewSessionGuard("", true).Reserve(realReviewVideo, 1); err == nil {
		t.Fatal("expected missing explicit state path to fail closed")
	}
	path := filepath.Join(t.TempDir(), "review.json")
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := newRealReviewSessionGuard(path, true).Reserve(realReviewVideo, 1); err == nil {
		t.Fatal("expected corrupt persisted state to fail closed")
	}
}

func TestRealReviewSessionGuardStaleLockFailsClosed(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "review.json")
	lockPath := statePath + ".lock"
	if err := os.WriteFile(lockPath, []byte("pid=999999\ncreated_at_utc=2000-01-01T00:00:00Z\n"), 0600); err != nil {
		t.Fatal(err)
	}
	err := newRealReviewSessionGuard(statePath, true).Reserve(realReviewVideo, 1)
	if err == nil {
		t.Fatal("expected stale lock to fail closed")
	}
	message := err.Error()
	for _, want := range []string{"fail-closed", "manually audit", "PID/process"} {
		if !strings.Contains(message, want) {
			t.Fatalf("stale-lock error %q does not contain %q", message, want)
		}
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("guard must not auto-remove stale lock: %v", err)
	}
}

func TestRealReviewSessionGuardHelperProcess(t *testing.T) {
	if os.Getenv("SUB2API_REAL_REVIEW_HELPER") != "1" {
		return
	}
	cost, err := strconv.ParseFloat(os.Getenv("SUB2API_REAL_REVIEW_HELPER_COST"), 64)
	if err != nil {
		os.Exit(3)
	}
	err = newRealReviewSessionGuard(os.Getenv("SUB2API_REAL_REVIEW_HELPER_STATE"), true).Reserve(realReviewVideo, cost)
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestRealReviewSessionGuardCompetingProcessesEnforceLimits(t *testing.T) {
	runCompetition := func(t *testing.T, cost float64, processes, wantAdmitted int) {
		t.Helper()
		statePath := filepath.Join(t.TempDir(), "review.json")
		commands := make([]*exec.Cmd, 0, processes)
		for i := 0; i < processes; i++ {
			cmd := exec.Command(os.Args[0], "-test.run=^TestRealReviewSessionGuardHelperProcess$")
			cmd.Env = append(os.Environ(),
				"SUB2API_REAL_REVIEW_HELPER=1",
				"SUB2API_REAL_REVIEW_HELPER_STATE="+statePath,
				"SUB2API_REAL_REVIEW_HELPER_COST="+strconv.FormatFloat(cost, 'f', -1, 64),
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
		state, err := newRealReviewSessionGuard(statePath, true).loadStateForTest()
		if err != nil {
			t.Fatal(err)
		}
		if state.VideoAttempts != wantAdmitted || state.ReservedCNY != float64(wantAdmitted)*cost {
			t.Fatalf("persisted state = %#v", state)
		}
	}

	t.Run("video attempt cap admits only four", func(t *testing.T) {
		runCompetition(t, 10, 12, 4)
	})
	t.Run("CNY cap admits only sixty", func(t *testing.T) {
		runCompetition(t, 20, 12, 3)
	})
}
