//go:build integration && realsmoke

package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// TestSeedanceRecoverExistingTaskThroughPostgresWorkerAndOutbox never creates an
// upstream generation. It creates the local reservation/task while no worker
// exists, then attaches an already-created upstream task ID and starts from the
// submitted state. The production worker can therefore only GET/Poll.
func TestSeedanceRecoverExistingTaskThroughPostgresWorkerAndOutbox(t *testing.T) {
	if strings.TrimSpace(os.Getenv("SUB2API_SEEDANCE_DB_RECOVER_REAL_SMOKE_RUN")) != "1" {
		t.Skip("database recovery disarmed: polls one existing task and never creates a generation")
	}
	apiKey := strings.TrimSpace(os.Getenv("SUB2API_SEEDANCE_SMOKE_API_KEY"))
	upstreamTaskID := strings.TrimSpace(os.Getenv("SUB2API_SEEDANCE_RECOVER_TASK_ID"))
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	auditPath := strings.TrimSpace(os.Getenv("SUB2API_VIDEO_REDACTED_EVENT_LOG"))
	if apiKey == "" || upstreamTaskID == "" || dataDir == "" || auditPath == "" {
		t.Fatal("database recovery requires key, existing task id, DATA_DIR, and redacted audit path; values intentionally not logged")
	}
	require.Equal(t, "1", strings.TrimSpace(os.Getenv("SUB2API_VIDEO_REAL_SMOKE_ENABLED")))
	require.NotEmpty(t, strings.TrimSpace(os.Getenv("SUB2API_VIDEO_URL_ALLOWLIST")))
	require.NoError(t, os.MkdirAll(dataDir, 0700))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	encryptionKey := make([]byte, 32)
	_, err := rand.Read(encryptionKey)
	require.NoError(t, err)
	cfg := &config.Config{
		VideoGateway: config.VideoGatewayConfig{
			EncryptionKey:       hex.EncodeToString(encryptionKey),
			WorkerEnabled:       true,
			PollIntervalSeconds: 2,
			TaskTimeoutMinutes:  10,
			WorkerBatchSize:     10,
			MaxPollAttempts:     72,
			CostPerSecond:       1.5,
			PerCallBudget:       30,
		},
		ReliabilityCore: config.ReliabilityCoreConfig{
			VideoEnabled:        true,
			ReservationTTLHours: 1,
			Outbox: config.DomainOutboxConfig{
				PollIntervalSeconds: 1,
				ClaimBatchSize:      50,
				LeaseSeconds:        30,
				MaxAttempts:         8,
				RetryBackoffSeconds: []int{1, 2, 4, 8},
			},
		},
	}
	cfg.Gateway.ContentCapture = config.ContentCaptureConfig{Enabled: true, PromptMaxBytes: 65536, ResponseMaxBytes: 65536}

	encryptor, err := NewVideoKeyEncryptor(cfg)
	require.NoError(t, err)
	gatewayRepo := NewVideoGatewayRepository(integrationDB)
	video := service.NewVideoGatewayService(gatewayRepo, encryptor, cfg)
	video.SetGenerationContentCollector(service.NewGenerationContentCollector(NewGenerationContentRepository(integrationDB), cfg))
	user := newVideoTaskCreationUser(t, 100)
	provider, err := video.CreateProviderAccount(ctx, service.VideoProviderCreateParams{
		Provider:           service.VideoProviderSeedance,
		DisplayName:        "real-review-recovery-" + uuid.NewString(),
		Enabled:            true,
		APIKey:             apiKey,
		DefaultModel:       "doubao-seedance-2-0-260128",
		RateLimitPerMinute: 60,
		Metadata:           map[string]any{"single_smoke_authorized": true, "recovery_only": true},
	})
	require.NoError(t, err)
	require.NotContains(t, provider.EncryptedAPIKey, apiKey)

	created, err := video.CreateTask(ctx, service.VideoTaskCreateParams{
		ProviderAccountID:   provider.ID,
		TaskType:            service.VideoTaskTypeTextToVideo,
		Model:               "doubao-seedance-2-0-260128",
		Prompt:              "Recover an existing 9:16 Seedance review task into the real product ledger and asset chain",
		AspectRatio:         "9:16",
		Duration:            5,
		Resolution:          "720p",
		CreatedBy:           user.ID,
		CreationKey:         service.HashIdempotencyKey("seedance-db-recovery-" + uuid.NewString()),
		CreationFingerprint: service.HashIdempotencyKey("seedance-db-recovery-payload-" + upstreamTaskID),
	})
	require.NoError(t, err)
	require.NotNil(t, created.ReservationID)
	trackReliabilityOwnedIDs(t, created.ID)

	// Critical safety boundary: no worker exists while the task is queued. Persist
	// the known upstream ID and submitted state before constructing the worker.
	created.Status = service.VideoStatusSubmitted
	created.UpstreamTaskID = upstreamTaskID
	require.NoError(t, gatewayRepo.UpdateTask(ctx, created))

	worker := service.NewVideoGatewayWorker(video, cfg)
	require.NoError(t, worker.ProcessOnce(ctx))
	terminal, err := gatewayRepo.GetTask(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, service.VideoStatusSucceeded, terminal.Status)
	require.Equal(t, upstreamTaskID, terminal.UpstreamTaskID)
	require.NotNil(t, terminal.UsageTotalTokens)
	require.Greater(t, *terminal.UsageTotalTokens, int64(0))
	require.Equal(t, "720p", terminal.ActualResolution)
	require.NotNil(t, terminal.ActualDuration)
	require.Equal(t, 5, *terminal.ActualDuration)
	require.Equal(t, service.VideoSettlementStatusSettled, terminal.SettlementStatus)
	require.NotNil(t, terminal.BalanceChargedAt)

	outboxRepo := NewDomainOutboxRepository(integrationDB)
	handlers := service.NewVideoOutboxHandlers(video, outboxRepo, nil, nil)
	now := time.Now().UTC()
	outboxWorker := service.NewDomainOutboxWorker(outboxRepo, handlers, cfg, service.DomainOutboxWorkerOptions{
		WorkerID: "seedance-db-recovery",
		Now:      func() time.Time { return now },
	})
	deadline := time.Now().Add(90 * time.Second)
	for {
		require.NoError(t, outboxWorker.RunOnce(ctx))
		terminal, err = gatewayRepo.GetTask(ctx, created.ID)
		require.NoError(t, err)
		if terminal.ArchiveStatus == "succeeded" && terminal.CaptureStatus == "succeeded" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("owned outbox did not complete: archive=%s capture=%s", terminal.ArchiveStatus, terminal.CaptureStatus)
		}
		now = now.Add(10 * time.Second)
	}

	require.NotEmpty(t, terminal.LocalAssetPath)
	abs, err := service.ResolveLocalAssetAbsPath(terminal.LocalAssetPath)
	require.NoError(t, err)
	info, err := os.Stat(abs)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
	require.Equal(t, ".mp4", strings.ToLower(filepath.Ext(abs)))

	var reservationStatus, settledAmount string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT status, settled_amount_usd::text FROM billing_reservations WHERE id = $1
	`, *terminal.ReservationID).Scan(&reservationStatus, &settledAmount))
	require.Equal(t, service.BillingReservationStatusSettled, reservationStatus)

	var transactionCount, usageCount, contentCount int
	var amountUSD, amountOriginal, currencyOriginal, balanceBefore, balanceAfter string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*), MIN(amount_usd::text), MIN(amount_original::text), MIN(currency_original),
		       MIN(balance_before::text), MIN(balance_after::text)
		FROM billing_transactions
		WHERE source_type = 'video_task' AND source_id = $1 AND transaction_kind = 'charge'
	`, terminal.ID).Scan(&transactionCount, &amountUSD, &amountOriginal, &currencyOriginal, &balanceBefore, &balanceAfter))
	require.Equal(t, 1, transactionCount)
	require.Equal(t, settledAmount, amountUSD)
	require.Equal(t, service.BillingCurrencyCNY, currencyOriginal)
	require.NotEqual(t, "0.0000000000", amountOriginal)
	require.NotEqual(t, balanceBefore, balanceAfter)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM video_usage_logs WHERE video_task_id = $1", terminal.ID).Scan(&usageCount))
	require.Equal(t, 1, usageCount)
	usageSummary, err := gatewayRepo.UsageSummarySince(ctx, terminal.CreatedAt.Add(-time.Minute))
	require.NoError(t, err)
	require.NotEmpty(t, usageSummary)
	require.Equal(t, service.BillingCurrencyCNY, usageSummary[0].Currency)
	require.Equal(t, service.PricingSourceProviderUsage, usageSummary[0].PricingSource)
	require.NotEmpty(t, usageSummary[0].PricingVersion)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM ai_generation_content WHERE task_id = $1", terminal.ID).Scan(&contentCount))
	require.Equal(t, 1, contentCount)

	var incompleteOutbox int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM domain_outbox
		WHERE aggregate_type = 'video_task' AND aggregate_id = $1 AND status <> 'completed'
	`, terminal.ID).Scan(&incompleteOutbox))
	require.Equal(t, 0, incompleteOutbox)

	audit, err := os.ReadFile(auditPath)
	require.NoError(t, err)
	require.NotContains(t, string(audit), apiKey)
	require.Contains(t, string(audit), `"phase":"poll"`)
	require.NotContains(t, string(audit), `"phase":"create"`)

	t.Logf("Seedance DB recovery complete: task_id=%d usage=%d asset_bytes=%d ledger_rows=%d usage_rows=%d content_rows=%d",
		terminal.ID, *terminal.UsageTotalTokens, info.Size(), transactionCount, usageCount, contentCount)
}
