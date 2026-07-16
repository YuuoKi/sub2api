package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type workerRepoStub struct {
	task       *VideoTask
	provider   VideoProviderAccount
	begin      bool
	finalized  []VideoTaskFinalization
	progress   []string
	reserved   float64
	beginCalls int
}

type videoAuthInvalidatorStub struct {
	users []int64
	err   error
}

func (s *videoAuthInvalidatorStub) InvalidateAuthCacheByUserID(_ context.Context, userID int64) error {
	s.users = append(s.users, userID)
	return s.err
}

type videoBillingInvalidatorStub struct {
	users   []int64
	keys    []int64
	userErr error
	keyErr  error
}

func (s *videoBillingInvalidatorStub) InvalidateUserBalance(_ context.Context, userID int64) error {
	s.users = append(s.users, userID)
	return s.userErr
}
func (s *videoBillingInvalidatorStub) InvalidateAPIKeyRateLimit(_ context.Context, keyID int64) error {
	s.keys = append(s.keys, keyID)
	return s.keyErr
}

func TestInvalidateVideoCachesAggregatesEveryFailure(t *testing.T) {
	authErr := errors.New("auth delete failed")
	balanceErr := errors.New("balance delete failed")
	rateErr := errors.New("rate delete failed")
	err := invalidateVideoCaches(context.Background(), &videoAuthInvalidatorStub{err: authErr},
		&videoBillingInvalidatorStub{userErr: balanceErr, keyErr: rateErr}, 11, 12)
	require.ErrorIs(t, err, authErr)
	require.ErrorIs(t, err, balanceErr)
	require.ErrorIs(t, err, rateErr)
}

func TestVideoWorkerRequiresCacheInvalidators(t *testing.T) {
	w := NewVideoGatewayWorker(&workerRepoStub{}, keyDecryptStub{}, func(string, string) *SeedanceAdapter { return nil }, nil, nil, &config.Config{}, NewSingleSmokeAuthorization(false))
	require.ErrorContains(t, w.RunOnce(context.Background()), "cache invalidators")
}

func TestVideoActualUSDRejectsCostRoundedToZero(t *testing.T) {
	_, err := videoActualUSD(1, &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 0.000001, USDCNYExchangeRate: 100}})
	require.ErrorContains(t, err, "rounds to zero")
}

func (r *workerRepoStub) CreateTask(context.Context, *VideoTask) error { return nil }
func (r *workerRepoStub) ReserveAndCreateTask(_ context.Context, _ *VideoTask, maximumUSD float64) error {
	r.reserved = maximumUSD
	return nil
}
func (r *workerRepoStub) GetTask(context.Context, int64) (*VideoTask, error) { return r.task, nil }
func (r *workerRepoStub) GetTaskForScope(context.Context, int64, VideoTaskScope) (*VideoTask, error) {
	return r.task, nil
}
func (r *workerRepoStub) CancelTaskForScope(context.Context, int64, VideoTaskScope) (*VideoTask, error) {
	return r.task, nil
}
func (r *workerRepoStub) ClaimRunnableTasks(context.Context, int, time.Duration) ([]*VideoTask, error) {
	if r.task == nil {
		return nil, nil
	}
	return []*VideoTask{r.task}, nil
}
func (r *workerRepoStub) FinalizeTask(_ context.Context, f VideoTaskFinalization) (VideoTaskFinalizationResult, error) {
	r.finalized = append(r.finalized, f)
	return VideoTaskFinalizationResult{Applied: true}, nil
}
func (r *workerRepoStub) ListEnabledVideoProviders(context.Context, int64) ([]VideoProviderAccount, error) {
	return []VideoProviderAccount{r.provider}, nil
}
func (r *workerRepoStub) GetVideoProvider(context.Context, int64, int64) (*VideoProviderAccount, error) {
	return &r.provider, nil
}

func TestVideoGatewayCreateRequiresWorkerAndReservesServerMaximum(t *testing.T) {
	repo := &workerRepoStub{provider: VideoProviderAccount{ID: 10, GroupID: 9, Provider: "seedance", Enabled: true}}
	cmd := VideoTaskCreateCommand{Scope: VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, ProviderAccountID: 10, Prompt: "x", Duration: 4, Resolution: "720p"}
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7, TinyRealEstimateCNY: 0.7, TinyRealMaximumCNY: 1.4}}
	authCache := &videoAuthInvalidatorStub{}
	billingCache := &videoBillingInvalidatorStub{}
	_, err := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(false), cfg, authCache, billingCache).CreateTask(context.Background(), cmd)
	if !errors.Is(err, ErrVideoRealDispatchDenied) {
		t.Fatalf("err=%v", err)
	}
	_, err = NewVideoGatewayService(repo, NewSingleSmokeAuthorization(true), cfg, authCache, billingCache).CreateTask(context.Background(), cmd)
	if !errors.Is(err, ErrVideoRealDispatchDenied) {
		t.Fatalf("worker-disabled err=%v", err)
	}
	cfg.VideoGateway.WorkerEnabled = true
	task, err := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(true), cfg, authCache, billingCache).CreateTask(context.Background(), cmd)
	if err != nil || task == nil || task.DurationSeconds != 4 || task.Resolution != "720p" {
		t.Fatalf("task=%#v err=%v", task, err)
	}
	if repo.reserved != 0.2 || task.ReservedCostUSD != 0.2 || task.ReservationState != VideoReservationReserved {
		t.Fatalf("reserved=%v task=%#v", repo.reserved, task)
	}
	if len(authCache.users) != 1 || len(billingCache.users) != 1 || len(billingCache.keys) != 1 {
		t.Fatalf("auth=%v billing_users=%v billing_keys=%v", authCache.users, billingCache.users, billingCache.keys)
	}
}

func TestVideoGatewayRuntimeStartsOnlyWithExplicitWorkerAndRealGate(t *testing.T) {
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{WorkerEnabled: true, WorkerIntervalSeconds: 1}}
	w := NewVideoGatewayWorker(&workerRepoStub{}, keyDecryptStub{}, func(string, string) *SeedanceAdapter { return nil }, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}, cfg, NewSingleSmokeAuthorization(false))
	denied := ProvideVideoGatewayRuntime(w, cfg, NewSingleSmokeAuthorization(false))
	if denied.Running() {
		t.Fatal("runtime started without real gate")
	}
	allowed := ProvideVideoGatewayRuntime(w, cfg, NewSingleSmokeAuthorization(true))
	if !allowed.Running() {
		t.Fatal("runtime did not start")
	}
	allowed.Stop()
	if allowed.Running() {
		t.Fatal("runtime did not stop")
	}
}
func (r *workerRepoStub) BeginRealDispatch(context.Context, int64, int64) (bool, error) {
	r.beginCalls++
	return r.begin, nil
}
func (r *workerRepoStub) MarkVideoSubmitted(context.Context, int64, int64, string) error { return nil }
func (r *workerRepoStub) UpdateVideoProgress(_ context.Context, _ int64, _ int64, status string) error {
	r.progress = append(r.progress, status)
	return nil
}

type keyDecryptStub struct{}

func (keyDecryptStub) Encrypt(v string) (string, error) { return v, nil }
func (keyDecryptStub) Decrypt(string) (string, error)   { return "synthetic-provider-key", nil }

func TestVideoWorkerRequiresProcessGateBeforeDispatchClaim(t *testing.T) {
	repo := &workerRepoStub{begin: true, task: &VideoTask{ID: 7, GroupID: 9, ProviderAccountID: 10, Status: VideoStatusQueued, Version: 1, ReservedCostUSD: 0.2, ReservationState: VideoReservationReserved}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Enabled: true, BaseURL: SeedanceBaseURL, EncryptedAPIKey: "cipher"}}
	clientCalls := 0
	factory := func(string, string) *SeedanceAdapter { clientCalls++; return nil }
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, factory, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}, &config.Config{}, NewSingleSmokeAuthorization(false))
	require.ErrorIs(t, w.RunOnce(context.Background()), ErrVideoRealDispatchDenied)
	require.Zero(t, repo.beginCalls)
	require.Zero(t, clientCalls)
}

func TestVideoWorkerConsumesProcessGateAfterAtomicDBClaim(t *testing.T) {
	repo := &workerRepoStub{begin: false, task: &VideoTask{ID: 7, GroupID: 9, ProviderAccountID: 10, Status: VideoStatusQueued, Version: 1, ReservedCostUSD: 0.2, ReservationState: VideoReservationReserved}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Enabled: true, BaseURL: SeedanceBaseURL, EncryptedAPIKey: "cipher"}}
	gate := NewSingleSmokeAuthorization(true)
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, func(string, string) *SeedanceAdapter { return nil }, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}, &config.Config{}, gate)
	require.NoError(t, w.RunOnce(context.Background()))
	require.NoError(t, gate.Consume(), "DB denial must not consume process gate")
}

func TestVideoWorkerDelegatesSuccessfulAtomicCapture(t *testing.T) {
	tokens := int64(245025)
	repo := &workerRepoStub{task: &VideoTask{ID: 7, APIKeyID: 8, GroupID: 9, ProviderAccountID: 10, CreatedBy: 11, Model: SeedanceModel, Status: VideoStatusSubmitted, UpstreamTaskID: "up-7", Version: 2, DurationSeconds: 4, Resolution: "720p", ReservedCostUSD: 0.2, ReservationState: VideoReservationReserved}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Enabled: true, BaseURL: "https://ark.cn-beijing.volces.com", EncryptedAPIKey: "cipher"}}
	responseBody := `{"id":"up-7","status":"succeeded","content":{"video_url":"https://cdn.example.test/v.mp4"},"usage":{"completion_tokens":245025}}`
	factory := func(base, key string) *SeedanceAdapter {
		return NewSeedanceAdapter(&http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(responseBody)), Header: make(http.Header)}, nil
		})}, base, key)
	}
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7, TinyRealMaximumCNY: 0.1}}
	authCache := &videoAuthInvalidatorStub{}
	billingCache := &videoBillingInvalidatorStub{}
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, factory, authCache, billingCache, cfg, NewSingleSmokeAuthorization(true))
	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.finalized) != 1 || repo.finalized[0].CostAmount <= 0 || repo.finalized[0].Status != VideoStatusSucceeded || repo.finalized[0].Settlement != VideoSettlementCaptureActual || *repo.finalized[0].UsageTotalTokens != tokens {
		t.Fatalf("finalized=%#v", repo.finalized)
	}
	if len(authCache.users) != 1 || len(billingCache.users) != 1 || len(billingCache.keys) != 1 {
		t.Fatalf("caches were not invalidated")
	}
}

func TestVideoWorkerPreservesAssetsAndFailsWithoutCompletionTokens(t *testing.T) {
	repo := &workerRepoStub{task: &VideoTask{ID: 7, APIKeyID: 8, GroupID: 9, ProviderAccountID: 10, CreatedBy: 11, Model: SeedanceModel, Status: VideoStatusSubmitted, UpstreamTaskID: "up-7", Version: 2, DurationSeconds: 4, Resolution: "720p", ReservedCostUSD: 0.2, ReservationState: VideoReservationReserved}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Enabled: true, BaseURL: "https://ark.cn-beijing.volces.com", EncryptedAPIKey: "cipher"}}
	body := `{"id":"up-7","status":"succeeded","content":{"video_url":"https://cdn.example.test/v.mp4"}}`
	factory := func(base, key string) *SeedanceAdapter {
		return NewSeedanceAdapter(&http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}, base, key)
	}
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, factory, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}, &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7}}, NewSingleSmokeAuthorization(false))
	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.finalized) != 1 || repo.finalized[0].ProviderErrorCode != "billing_usage_missing" || repo.finalized[0].ResultURL == "" || repo.finalized[0].Settlement != VideoSettlementRelease {
		t.Fatalf("finalized=%#v", repo.finalized)
	}
}

func TestVideoWorkerFailsAndReleasesWhenSuccessOmitsVideoURL(t *testing.T) {
	tokens := int64(10)
	repo := &workerRepoStub{task: &VideoTask{ID: 7, APIKeyID: 8, GroupID: 9, ProviderAccountID: 10, CreatedBy: 11, Model: SeedanceModel, Status: VideoStatusSubmitted, UpstreamTaskID: "up-7", Version: 2, ReservedCostUSD: 0.2, ReservationState: VideoReservationReserved}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Enabled: true, BaseURL: "https://ark.cn-beijing.volces.com", EncryptedAPIKey: "cipher"}}
	body := `{"id":"up-7","status":"succeeded","usage":{"completion_tokens":10}}`
	factory := func(base, key string) *SeedanceAdapter {
		return NewSeedanceAdapter(&http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}, base, key)
	}
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, factory, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}, &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7}}, NewSingleSmokeAuthorization(false))
	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := repo.finalized[0]
	if got.Status != VideoStatusFailed || got.ProviderErrorCode != "result_asset_missing" || got.Settlement != VideoSettlementRelease || got.ResultURL != "" || got.UsageTotalTokens == nil || *got.UsageTotalTokens != tokens {
		t.Fatalf("finalized=%#v", got)
	}
}

func TestVideoWorkerCapturesReservationAndFailsWhenActualExceedsMaximum(t *testing.T) {
	repo := &workerRepoStub{task: &VideoTask{ID: 7, APIKeyID: 8, GroupID: 9, ProviderAccountID: 10, CreatedBy: 11, Model: SeedanceModel, Status: VideoStatusSubmitted, UpstreamTaskID: "up-7", Version: 2, ReservedCostUSD: 0.1, ReservationState: VideoReservationReserved}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Enabled: true, BaseURL: "https://ark.cn-beijing.volces.com", EncryptedAPIKey: "cipher"}}
	body := `{"id":"up-7","status":"succeeded","content":{"video_url":"https://cdn.example.test/v.mp4"},"usage":{"completion_tokens":1000000}}`
	factory := func(base, key string) *SeedanceAdapter {
		return NewSeedanceAdapter(&http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}, base, key)
	}
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, factory, &videoAuthInvalidatorStub{}, &videoBillingInvalidatorStub{}, &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7}}, NewSingleSmokeAuthorization(false))
	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := repo.finalized[0]
	if got.Status != VideoStatusFailed || got.ProviderErrorCode != "budget_actual_exceeded" || got.Settlement != VideoSettlementCaptureReserved || got.CostAmount != 0.1 || got.ProviderActualCostUSD <= got.CostAmount || got.ResultURL == "" {
		t.Fatalf("finalized=%#v", got)
	}
}
