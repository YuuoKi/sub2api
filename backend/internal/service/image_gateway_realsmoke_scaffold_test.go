package service

import "testing"

// Image smoke harness scaffold (env-gated). Real NB2 calls require boss authorization.
// Set SUB2API_IMAGE_REAL_SMOKE_ENABLED=1 and provide production_authorized account
// before enabling; default path is skip-only so CI stays free of paid calls.

func TestImageRealSmokeHarnessScaffold(t *testing.T) {
	t.Parallel()
	if !imageRealSmokeEnabled() {
		t.Skip("image real smoke disabled (set SUB2API_IMAGE_REAL_SMOKE_ENABLED=1 after boss authorization)")
	}
	t.Fatal("image real smoke enabled but no authorized runner wired in this session — refuse to call providers")
}

func imageRealSmokeEnabled() bool {
	return false
}
