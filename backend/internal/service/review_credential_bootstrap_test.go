package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestReviewCredentialBootstrapFailClosedWithoutSession(t *testing.T) {
	svc := NewVideoGatewayService(newMemoryVideoGatewayRepo(), noopVideoKeyEncryptor{}, &config.Config{})
	_, err := svc.ReviewCredentialBootstrap(context.Background())
	if err == nil {
		t.Fatal("expected fail-closed without review session")
	}
}

func TestReviewCredentialBootstrapUpsertsReviewOnlyWithoutLeakingSecrets(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "review-state.json")
	cfg := &config.Config{}
	cfg.RealReviewSession.Enabled = true
	cfg.RealReviewSession.StatePath = statePath
	repo := newMemoryVideoGatewayRepo()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, cfg)

	t.Setenv("GEMINI_API_KEY", "gemini-secret-should-not-appear")
	t.Setenv("SUB2API_SEEDANCE_SMOKE_API_KEY", "seedance-secret-should-not-appear")

	result, err := svc.ReviewCredentialBootstrap(context.Background())
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if result == nil || !result.Seedance.EnvPresent || result.Seedance.AccountID == 0 {
		t.Fatalf("unexpected seedance status: %+v", result)
	}
	if result.Gemini.Status != "ready" {
		t.Fatalf("gemini status=%q", result.Gemini.Status)
	}
	blob := result.Seedance.Message + result.Gemini.Message + result.Seedance.Status + result.Gemini.Status
	if strings.Contains(blob, "gemini-secret-should-not-appear") || strings.Contains(blob, "seedance-secret-should-not-appear") {
		t.Fatalf("secrets leaked in bootstrap result messages")
	}
	account, err := repo.GetProviderAccount(context.Background(), result.Seedance.AccountID)
	if err != nil {
		t.Fatal(err)
	}
	if !isReviewOnlyVideoAccount(account) {
		t.Fatal("seedance account must be review_only")
	}
}

func TestAllowProductMockBatchImageProviderNotInProduction(t *testing.T) {
	cfg := &config.Config{}
	cfg.Log.Environment = "production"
	cfg.Server.Mode = "release"
	if AllowProductMockBatchImageProvider(cfg) {
		t.Fatal("mock image provider must not register in ordinary production")
	}
	cfg.Log.Environment = "local"
	if !AllowProductMockBatchImageProvider(cfg) {
		t.Fatal("mock image provider should register in local")
	}
}
