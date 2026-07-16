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
)

type workerRepoStub struct {
	task      *VideoTask
	provider  VideoProviderAccount
	begin     bool
	finalized []VideoTaskFinalization
	progress  []string
}

func (r *workerRepoStub) CreateTask(context.Context, *VideoTask) error       { return nil }
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

type videoBudgetStub struct{ err error }

func (b videoBudgetStub) AuthorizeVideo(context.Context, VideoTaskScope) (float64, error) {
	return 0.2, b.err
}

func TestVideoGatewayCreateDefaultsDeniedAndDoesNotTrustCallerCost(t *testing.T) {
	repo := &workerRepoStub{provider: VideoProviderAccount{ID: 10, GroupID: 9, Provider: "seedance", Enabled: true}}
	cmd := VideoTaskCreateCommand{Scope: VideoTaskScope{UserID: 1, APIKeyID: 2, GroupID: 9}, ProviderAccountID: 10, Prompt: "x", Duration: 4, Resolution: "720p", SingleSmokeAuthorized: true}
	_, err := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(false), videoBudgetStub{}).CreateTask(context.Background(), cmd)
	if !errors.Is(err, ErrVideoRealDispatchDenied) {
		t.Fatalf("err=%v", err)
	}
	task, err := NewVideoGatewayService(repo, NewSingleSmokeAuthorization(true), videoBudgetStub{}).CreateTask(context.Background(), cmd)
	if err != nil || task == nil || task.DurationSeconds != 4 || task.Resolution != "720p" {
		t.Fatalf("task=%#v err=%v", task, err)
	}
}

func TestVideoGatewayRuntimeStartsOnlyWithExplicitWorkerAndRealGate(t *testing.T) {
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{WorkerEnabled: true, WorkerIntervalSeconds: 1}}
	w := NewVideoGatewayWorker(&workerRepoStub{}, keyDecryptStub{}, func(string, string) *SeedanceAdapter { return nil }, &billingDedupStub{}, cfg)
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

type billingDedupStub struct {
	effects int
	seen    map[string]string
	last    *UsageBillingCommand
}

func (b *billingDedupStub) Apply(_ context.Context, cmd *UsageBillingCommand) (*UsageBillingApplyResult, error) {
	b.last = cmd
	if b.seen == nil {
		b.seen = map[string]string{}
	}
	if fp, ok := b.seen[cmd.RequestID]; ok {
		if fp != cmd.RequestPayloadHash {
			return nil, ErrUsageBillingRequestConflict
		}
		return &UsageBillingApplyResult{}, nil
	}
	b.seen[cmd.RequestID] = cmd.RequestPayloadHash
	b.effects++
	return &UsageBillingApplyResult{Applied: true}, nil
}
func (*billingDedupStub) ReserveBatchImageBalance(context.Context, *BatchImageBalanceHoldCommand) (*BatchImageBalanceHoldResult, error) {
	return nil, nil
}
func (*billingDedupStub) CaptureBatchImageBalance(context.Context, *BatchImageBalanceHoldCommand) (*BatchImageBalanceHoldResult, error) {
	return nil, nil
}
func (*billingDedupStub) ReleaseBatchImageBalance(context.Context, *BatchImageBalanceHoldCommand) (*BatchImageBalanceHoldResult, error) {
	return nil, nil
}

func TestVideoWorkerBillsSuccessfulTerminalExactlyOnceWithStableFacts(t *testing.T) {
	tokens := int64(245025)
	repo := &workerRepoStub{task: &VideoTask{ID: 7, APIKeyID: 8, GroupID: 9, ProviderAccountID: 10, CreatedBy: 11, Model: SeedanceModel, Status: VideoStatusSubmitted, UpstreamTaskID: "up-7", Version: 2, DurationSeconds: 4, Resolution: "720p"}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Enabled: true, BaseURL: "https://ark.cn-beijing.volces.com", EncryptedAPIKey: "cipher"}}
	billing := &billingDedupStub{}
	responseBody := `{"id":"up-7","status":"succeeded","content":{"video_url":"https://cdn.example.test/v.mp4"},"usage":{"completion_tokens":245025}}`
	factory := func(base, key string) *SeedanceAdapter {
		return NewSeedanceAdapter(&http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(responseBody)), Header: make(http.Header)}, nil
		})}, base, key)
	}
	cfg := &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7, TinyRealMaximumCNY: 0.1}}
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, factory, billing, cfg)
	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if billing.effects != 1 || billing.last.AccountID != 0 || billing.last.AccountQuotaCost != 0 || billing.last.OutputTokens != int(tokens) {
		t.Fatalf("billing=%#v effects=%d", billing.last, billing.effects)
	}
	if len(repo.finalized) != 2 || repo.finalized[0].CostAmount <= 0 || repo.finalized[0].Status != VideoStatusSucceeded {
		t.Fatalf("finalized=%#v", repo.finalized)
	}
}

func TestVideoWorkerPreservesAssetsAndFailsWithoutCompletionTokens(t *testing.T) {
	repo := &workerRepoStub{task: &VideoTask{ID: 7, APIKeyID: 8, GroupID: 9, ProviderAccountID: 10, CreatedBy: 11, Model: SeedanceModel, Status: VideoStatusSubmitted, UpstreamTaskID: "up-7", Version: 2, DurationSeconds: 4, Resolution: "720p"}, provider: VideoProviderAccount{ID: 10, GroupID: 9, Enabled: true, BaseURL: "https://ark.cn-beijing.volces.com", EncryptedAPIKey: "cipher"}}
	billing := &billingDedupStub{}
	body := `{"id":"up-7","status":"succeeded","content":{"video_url":"https://cdn.example.test/v.mp4"}}`
	factory := func(base, key string) *SeedanceAdapter {
		return NewSeedanceAdapter(&http.Client{Transport: videoRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}, base, key)
	}
	w := NewVideoGatewayWorker(repo, keyDecryptStub{}, factory, billing, &config.Config{VideoGateway: config.VideoGatewayConfig{SeedanceCNYPerMillionTokens: 2, USDCNYExchangeRate: 7}})
	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if billing.effects != 0 || len(repo.finalized) != 1 || repo.finalized[0].ProviderErrorCode != "billing_usage_missing" || repo.finalized[0].ResultURL == "" {
		t.Fatalf("billing=%d finalized=%#v", billing.effects, repo.finalized)
	}
}
