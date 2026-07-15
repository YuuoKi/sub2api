//go:build integration

package repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// TestGeminiBatchImageZeroCreateProductRecovery proves development-phase G3:
// recover an already-submitted Gemini batch through the product processor
// (Get → OpenResult → Index → Settle) with Submit/create = 0, using only a
// desensitized recorded fixture client — never a live Provider.
func TestGeminiBatchImageZeroCreateProductRecovery(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	rdb := testRedis(t) // harness Redis must be live for product download limiter

	fixtureJSONL := mustLoadDesensitizedBatchImageResultJSONL(t)
	upstreamJobName := "batches/fixture-recover-" + uuid.NewString()
	outputRef := "files/fixture-recover-output-" + uuid.NewString()

	fixtureClient := &countingGeminiFixtureClient{
		jobName:    upstreamJobName,
		outputRef:  outputRef,
		resultBody: fixtureJSONL,
	}
	geminiProvider := service.NewGeminiAPIBatchImageProvider(fixtureClient)
	provider := &countingBatchImageProvider{inner: geminiProvider}

	const (
		startBalance   = 10.0
		unitPrice      = 0.05
		holdAmount     = 0.12 // covers 2 success items at unitPrice with headroom
		expectedActual = 0.10 // 2 * unitPrice
	)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("gemini-recover-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      startBalance,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-fixture-gemini-recover-" + uuid.NewString(),
		Name:   "gemini-recover",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name:     "gemini-recover-" + uuid.NewString(),
		Platform: service.PlatformGemini,
		Type:     service.AccountTypeAPIKey,
		// Synthetic fixture credential only — never a real Provider secret.
		Credentials: map[string]any{"api_key": "fixture-gemini-api-key"},
	})

	batchID, err := service.NewBatchImageID()
	require.NoError(t, err)
	requestHash := "fixture-request-hash-" + uuid.NewString()
	holdID := service.BatchImageHoldRequestID(batchID)
	holdPtr := holdAmount
	apiKeyID := apiKey.ID
	accountID := account.ID
	providerJobName := upstreamJobName
	requestHashPtr := requestHash
	holdIDPtr := holdID

	repo := NewBatchImageRepository(integrationDB)
	billingRepo := NewUsageBillingRepository(client, integrationDB)
	usageLogRepo := NewUsageLogRepository(client, integrationDB)
	accountRepo := NewAccountRepository(client, integrationDB, nil)

	job, err := repo.CreateBatchImageJob(ctx, service.CreateBatchImageJobParams{
		BatchID:                 batchID,
		UserID:                  user.ID,
		APIKeyID:                &apiKeyID,
		AccountID:               &accountID,
		Provider:                service.BatchImageProviderGeminiAPI,
		Model:                   "gemini-2.5-flash-image",
		TaskName:                "g3-zero-create-recovery",
		Status:                  service.BatchImageJobStatusSubmitted,
		ProviderJobName:         &providerJobName,
		ItemCount:               2,
		EstimatedCost:           holdAmount,
		HoldAmount:              &holdPtr,
		HoldID:                  &holdIDPtr,
		BaseUnitPrice:           unitPrice,
		GroupRateMultiplier:     1,
		AccountRateMultiplier:   1,
		BatchDiscountMultiplier: 1,
		HoldMultiplier:          1,
		BillableUnitPrice:       unitPrice,
		HoldUnitPrice:           unitPrice,
		PricingSnapshotVersion:  1,
		Currency:                "USD",
		RequestHash:             &requestHashPtr,
	})
	require.NoError(t, err)
	require.Equal(t, service.BatchImageJobStatusSubmitted, job.Status)
	require.NotNil(t, job.ProviderJobName)
	require.Equal(t, upstreamJobName, *job.ProviderJobName)

	require.NoError(t, repo.BulkCreateBatchImageItems(ctx, []service.CreateBatchImageItemParams{
		{JobID: batchID, CustomID: "cover_001", Status: service.BatchImageItemStatusPending},
		{JobID: batchID, CustomID: "cover_002", Status: service.BatchImageItemStatusPending},
	}))

	reserveResult, err := billingRepo.ReserveBatchImageBalance(ctx, &service.BatchImageBalanceHoldCommand{
		RequestID:          holdID,
		APIKeyID:           apiKey.ID,
		UserID:             user.ID,
		BatchID:            batchID,
		HoldAmount:         holdAmount,
		RequestPayloadHash: requestHash,
	})
	require.NoError(t, err)
	require.True(t, reserveResult.Applied)
	assertUserBalances(t, ctx, user.ID, startBalance-holdAmount, holdAmount)

	cfg := &config.Config{BatchImage: config.BatchImageConfig{
		Enabled:                           true,
		MaxDownloadItemsZip:               10,
		MaxDownloadDurationSeconds:        60,
		MaxDownloadConcurrencyPerUser:     2,
		OutputRetentionAfterTerminalHours: 72,
	}}
	registry := service.NewBatchImageProviderRegistry(provider)
	accountResolver := &service.BatchImageAccountRepositoryResolver{Repo: accountRepo}
	processor := &service.BatchImagePipelineProcessor{
		ProviderProcessor: &service.BatchImageProviderProcessor{
			Repo:             repo,
			ProviderRegistry: registry,
			AccountResolver:  accountResolver,
			BillingRepo:      billingRepo,
		},
		SettlementService: &service.BatchImageSettlementService{
			Repo:         repo,
			BillingRepo:  billingRepo,
			UsageLogRepo: usageLogRepo,
			Pricing:      &fixedBatchImagePricing{unitPrice: unitPrice},
			Config:       cfg,
		},
	}
	downloadSvc := &service.BatchImageDownloadService{
		Repo:             repo,
		ProviderRegistry: registry,
		AccountResolver:  accountResolver,
		Limiter:          NewBatchImageDownloadLimiter(rdb, cfg),
		Config:           cfg,
	}

	// Drive production Process loop: Get → Index(OpenResult) → Settle.
	var terminal bool
	for i := 0; i < 8; i++ {
		result, processErr := processor.Process(ctx, batchID)
		require.NoError(t, processErr)
		if result.Terminal {
			terminal = true
			break
		}
		if result.RequeueAfter > 0 && result.RequeueAfter < time.Second {
			time.Sleep(result.RequeueAfter)
		}
	}
	require.True(t, terminal, "recovery processor should reach a terminal state")

	submit, get, openResult := provider.Counts()
	require.Equal(t, 0, submit, "recovery must never Submit/create upstream")
	require.GreaterOrEqual(t, get, 1, "recovery must Get existing upstream job")
	require.GreaterOrEqual(t, openResult, 1, "recovery must OpenResult for JSONL index")
	require.Equal(t, 0, fixtureClient.UploadCalls(), "fixture client UploadJSONL must stay 0")
	require.Equal(t, 0, fixtureClient.CreateCalls(), "fixture client CreateBatch must stay 0")
	require.GreaterOrEqual(t, fixtureClient.GetCalls(), 1)
	require.GreaterOrEqual(t, fixtureClient.DownloadCalls(), 1)
	t.Logf("zero-create evidence: Submit=%d Get=%d OpenResult=%d upload=%d create=%d download=%d",
		submit, get, openResult, fixtureClient.UploadCalls(), fixtureClient.CreateCalls(), fixtureClient.DownloadCalls())

	settled, err := repo.GetBatchImageJobByBatchID(ctx, batchID)
	require.NoError(t, err)
	require.Equal(t, service.BatchImageJobStatusCompleted, settled.Status)
	require.Equal(t, 2, settled.SuccessCount)
	require.Equal(t, 0, settled.FailCount)
	require.NotNil(t, settled.ActualCost)
	require.InDelta(t, expectedActual, *settled.ActualCost, 1e-9)
	require.NotNil(t, settled.SettledAt)
	require.NotNil(t, settled.OutputExpiresAt)

	assertUserBalances(t, ctx, user.ID, startBalance-expectedActual, 0)

	var usageCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM usage_logs
		WHERE user_id = $1 AND api_key_id = $2 AND request_id = $3
	`, user.ID, apiKey.ID, service.BatchImageCaptureRequestID(batchID)).Scan(&usageCount))
	require.Equal(t, 1, usageCount)

	items, err := repo.ListBatchImageItems(ctx, batchID, service.BatchImageItemFilter{})
	require.NoError(t, err)
	require.Len(t, items, 2)
	for _, item := range items {
		require.Equal(t, service.BatchImageItemStatusSuccess, item.Status)
		require.Equal(t, 1, item.ImageCount)
	}

	// Decode at least one image via production download service; keep bytes in memory only.
	owner := service.BatchImageOwner{UserID: user.ID, APIKeyID: apiKey.ID}
	stream, err := downloadSvc.OpenItemContent(ctx, owner, batchID, "cover_001", 0)
	require.NoError(t, err)
	require.NotNil(t, stream)
	imageBytes, err := io.ReadAll(stream.Reader)
	require.NoError(t, err)
	require.NoError(t, stream.Reader.Close())
	require.Equal(t, "image/png", stream.ContentType)
	require.True(t, bytes.HasPrefix(imageBytes, []byte{0x89, 0x50, 0x4e, 0x47}), "decoded content must be PNG")
	img, decodeErr := png.Decode(bytes.NewReader(imageBytes))
	require.NoError(t, decodeErr)
	require.Equal(t, 1, img.Bounds().Dx())
	require.Equal(t, 1, img.Bounds().Dy())

	// Terminal irreversibility: completed cannot return to running.
	err = repo.TransitionBatchImageJobStatus(ctx, batchID, service.BatchImageJobStatusRunning, service.BatchImageTransitionOptions{})
	require.ErrorIs(t, err, service.ErrBatchImageInvalidTransition)

	// Idempotent re-recovery: Process again must stay terminal without new create.
	beforeSubmit, beforeGet, beforeOpen := provider.Counts()
	again, err := processor.Process(ctx, batchID)
	require.NoError(t, err)
	require.True(t, again.Terminal)
	afterSubmit, afterGet, afterOpen := provider.Counts()
	require.Equal(t, beforeSubmit, afterSubmit)
	require.Equal(t, beforeGet, afterGet, "completed re-process must not poll provider again")
	require.Equal(t, beforeOpen, afterOpen, "completed re-process must not open result again")

	settleAgain, err := processor.SettlementService.Settle(ctx, batchID)
	require.NoError(t, err)
	require.True(t, settleAgain.AlreadySettled)
	assertUserBalances(t, ctx, user.ID, startBalance-expectedActual, 0)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM usage_logs
		WHERE user_id = $1 AND api_key_id = $2 AND request_id = $3
	`, user.ID, apiKey.ID, service.BatchImageCaptureRequestID(batchID)).Scan(&usageCount))
	require.Equal(t, 1, usageCount, "usage log must remain idempotent")

	finalSubmit, finalGet, finalOpen := provider.Counts()
	require.Equal(t, 0, finalSubmit)
	require.GreaterOrEqual(t, finalGet, 1)
	require.GreaterOrEqual(t, finalOpen, 1)
}

func assertUserBalances(t *testing.T, ctx context.Context, userID int64, wantBalance, wantFrozen float64) {
	t.Helper()
	var balance, frozen float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT balance, COALESCE(frozen_balance, 0) FROM users WHERE id = $1
	`, userID).Scan(&balance, &frozen))
	require.InDelta(t, wantBalance, balance, 1e-9, "balance")
	require.InDelta(t, wantFrozen, frozen, 1e-9, "frozen_balance")
}

func mustLoadDesensitizedBatchImageResultJSONL(t *testing.T) string {
	t.Helper()
	// Prefer the checked-in mock fixture shape; rewrite image payloads with a
	// checksum-valid 1x1 PNG so production download decode can succeed.
	// The stock mock base64 is intentionally tiny but fails Go's png.Decode CRC.
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(thisFile), "..", "service", "testdata", "mock_batch_image_result.jsonl")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"custom_id":"cover_001"`)
	require.NotContains(t, string(raw), "AIza")
	require.NotContains(t, string(raw), "ya29.")

	var pngBuf bytes.Buffer
	require.NoError(t, png.Encode(&pngBuf, image.NewNRGBA(image.Rect(0, 0, 1, 1))))
	encoded := base64.StdEncoding.EncodeToString(pngBuf.Bytes())
	_, decodeErr := png.Decode(bytes.NewReader(pngBuf.Bytes()))
	require.NoError(t, decodeErr, "synthetic fixture PNG must be decodable")

	var out strings.Builder
	for i, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]any
		require.NoErrorf(t, json.Unmarshal([]byte(line), &obj), "fixture line %d", i+1)
		customID := fmt.Sprint(obj["custom_id"])
		require.NotEmpty(t, customID)
		out.WriteString(`{"custom_id":"` + customID + `","response":{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"` + encoded + `"}}]}}]}}`)
		out.WriteByte('\n')
	}
	require.NotZero(t, out.Len())
	return out.String()
}

type fixedBatchImagePricing struct {
	unitPrice float64
}

func (p *fixedBatchImagePricing) BatchImageUnitPrice(context.Context, *service.BatchImageJob) (float64, error) {
	return p.unitPrice, nil
}

type countingBatchImageProvider struct {
	inner      service.BatchImageProvider
	mu         sync.Mutex
	submit     int
	get        int
	openResult int
}

func (p *countingBatchImageProvider) Name() string { return p.inner.Name() }

func (p *countingBatchImageProvider) SupportsAccount(account *service.Account) bool {
	return p.inner.SupportsAccount(account)
}

func (p *countingBatchImageProvider) Submit(ctx context.Context, job *service.BatchImageJob, account *service.Account, input service.BatchImageInput) (*service.BatchProviderJob, error) {
	p.mu.Lock()
	p.submit++
	p.mu.Unlock()
	return p.inner.Submit(ctx, job, account, input)
}

func (p *countingBatchImageProvider) Get(ctx context.Context, job *service.BatchImageJob, account *service.Account) (*service.BatchProviderStatus, error) {
	p.mu.Lock()
	p.get++
	p.mu.Unlock()
	return p.inner.Get(ctx, job, account)
}

func (p *countingBatchImageProvider) Cancel(ctx context.Context, job *service.BatchImageJob, account *service.Account) error {
	return p.inner.Cancel(ctx, job, account)
}

func (p *countingBatchImageProvider) OpenResult(ctx context.Context, job *service.BatchImageJob, account *service.Account) (io.ReadCloser, string, error) {
	p.mu.Lock()
	p.openResult++
	p.mu.Unlock()
	return p.inner.OpenResult(ctx, job, account)
}

func (p *countingBatchImageProvider) Cleanup(ctx context.Context, job *service.BatchImageJob, account *service.Account, target service.CleanupTarget) error {
	return p.inner.Cleanup(ctx, job, account, target)
}

func (p *countingBatchImageProvider) Counts() (submit, get, openResult int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.submit, p.get, p.openResult
}

// countingGeminiFixtureClient is a desensitized recorded Gemini client.
// Upload/Create intentionally fail hard if recovery accidentally creates.
type countingGeminiFixtureClient struct {
	mu         sync.Mutex
	jobName    string
	outputRef  string
	resultBody string
	upload     int
	create     int
	get        int
	download   int
}

func (c *countingGeminiFixtureClient) UploadJSONL(context.Context, string, string, io.Reader) (*service.GeminiUploadedFile, error) {
	c.mu.Lock()
	c.upload++
	c.mu.Unlock()
	return nil, fmt.Errorf("fixture recovery forbids UploadJSONL/create")
}

func (c *countingGeminiFixtureClient) CreateBatch(context.Context, string, string, string, string) (*service.GeminiBatchJob, error) {
	c.mu.Lock()
	c.create++
	c.mu.Unlock()
	return nil, fmt.Errorf("fixture recovery forbids CreateBatch")
}

func (c *countingGeminiFixtureClient) GetBatch(_ context.Context, _ string, batchName string) (*service.GeminiBatchJob, error) {
	c.mu.Lock()
	c.get++
	c.mu.Unlock()
	if strings.TrimSpace(batchName) == "" || batchName != c.jobName {
		return nil, fmt.Errorf("unknown fixture batch")
	}
	return &service.GeminiBatchJob{
		Name:  c.jobName,
		State: "JOB_STATE_SUCCEEDED",
		Dest:  &service.GeminiBatchDest{FileName: c.outputRef},
	}, nil
}

func (c *countingGeminiFixtureClient) CancelBatch(context.Context, string, string) error {
	return fmt.Errorf("fixture recovery forbids CancelBatch")
}

func (c *countingGeminiFixtureClient) DownloadFile(_ context.Context, _ string, fileName string) (io.ReadCloser, string, error) {
	c.mu.Lock()
	c.download++
	c.mu.Unlock()
	if strings.TrimSpace(fileName) == "" || fileName != c.outputRef {
		return nil, "", fmt.Errorf("unknown fixture result file")
	}
	return io.NopCloser(strings.NewReader(c.resultBody)), "application/jsonl", nil
}

func (c *countingGeminiFixtureClient) DeleteFile(context.Context, string, string) error {
	return nil
}

func (c *countingGeminiFixtureClient) UploadCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.upload
}

func (c *countingGeminiFixtureClient) CreateCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.create
}

func (c *countingGeminiFixtureClient) GetCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.get
}

func (c *countingGeminiFixtureClient) DownloadCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.download
}
