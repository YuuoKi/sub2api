//go:build realsmoke

package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	geminiNB2RealReviewModel = "gemini-3.1-flash-image-preview"
	// As of 2026-07-12 Google's published Batch price for a highest-cost 4K image is
	// USD 0.076. CNY 5 includes conservative input-token, FX, and price-drift buffer.
	// It is a fixed reservation, not an operator estimate; actual bills still require reconciliation.
	geminiNB2FixedReserveCNY = 5.0
	geminiNB2MaxResultBytes  = 16 << 20
)

type geminiNB2RealReviewOptions struct {
	Guard      *realReviewSessionGuard
	PerCallCap float64
	Prompt     string
	ReadAPIKey func() string
	NewClient  func() GeminiBatchClient
}

type geminiNB2PreparedReview struct {
	Client  GeminiBatchClient
	Account *Account
	Job     *BatchImageJob
	Input   BatchImageInput
}

func prepareGeminiNB2RealReview(opts geminiNB2RealReviewOptions) (*geminiNB2PreparedReview, error) {
	if opts.Guard == nil {
		return nil, errors.New("GEMINI_NB2_REVIEW_GUARD_REQUIRED")
	}
	if !(opts.PerCallCap >= geminiNB2FixedReserveCNY) {
		return nil, fmt.Errorf("GEMINI_NB2_PER_CALL_CAP must be at least %.2f CNY", geminiNB2FixedReserveCNY)
	}
	if err := opts.Guard.Reserve(realReviewImage, geminiNB2FixedReserveCNY); err != nil {
		return nil, err
	}
	if opts.ReadAPIKey == nil || opts.NewClient == nil {
		return nil, errors.New("GEMINI_NB2_HARNESS_INVALID")
	}
	apiKey := strings.TrimSpace(opts.ReadAPIKey())
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is empty (value intentionally not logged)")
	}
	client := opts.NewClient()
	if client == nil {
		return nil, errors.New("GEMINI_NB2_CLIENT_MISSING")
	}
	prompt := strings.TrimSpace(opts.Prompt)
	if prompt == "" {
		prompt = "A single neutral product-review scene, no text, one image only."
	}
	batchID := fmt.Sprintf("nb2-review-%d", time.Now().UTC().UnixNano())
	input := BatchImageInput{BatchID: batchID, Model: geminiNB2RealReviewModel, DisplayName: batchID, Items: []BatchImageInputItem{{CustomID: "scene-01", Prompt: prompt}}}
	return &geminiNB2PreparedReview{
		Client:  client,
		Account: &Account{Platform: PlatformGemini, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": apiKey}},
		Job:     &BatchImageJob{BatchID: batchID, Provider: BatchImageProviderGeminiAPI, Model: geminiNB2RealReviewModel, ItemCount: 1, EstimatedCost: geminiNB2FixedReserveCNY, Currency: "CNY"},
		Input:   input,
	}, nil
}

type geminiNB2FlowProvider interface {
	Submit(context.Context, *BatchImageJob, *Account, BatchImageInput) (*BatchProviderJob, error)
	Get(context.Context, *BatchImageJob, *Account) (*BatchProviderStatus, error)
	OpenResult(context.Context, *BatchImageJob, *Account) (io.ReadCloser, string, error)
	Cancel(context.Context, *BatchImageJob, *Account) error
}

func runGeminiNB2ProductFlow(ctx context.Context, provider geminiNB2FlowProvider, prepared *geminiNB2PreparedReview, wait func(context.Context) error) (retErr error) {
	submitted, err := provider.Submit(ctx, prepared.Job, prepared.Account, prepared.Input)
	if err != nil {
		return fmt.Errorf("Submit: %w", err)
	}
	prepared.Job.ProviderJobName = &submitted.ProviderJobName
	prepared.Job.ProviderInputRef = &submitted.ProviderInputRef
	completed := false
	defer func() {
		if !completed {
			cancelCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if cancelErr := provider.Cancel(cancelCtx, prepared.Job, prepared.Account); cancelErr != nil {
				retErr = fmt.Errorf("%w; GEMINI_NB2_CANCEL_UNCONFIRMED: %v", retErr, cancelErr)
			}
		}
	}()
	for {
		status, err := provider.Get(ctx, prepared.Job, prepared.Account)
		if err != nil {
			return fmt.Errorf("Get: %w", err)
		}
		if status.Done {
			if status.InternalState != BatchProviderStateSucceeded || status.ProviderOutputRef == "" {
				return fmt.Errorf("terminal state=%q error=%q", status.InternalState, status.ErrorCode)
			}
			prepared.Job.ProviderOutputRef = &status.ProviderOutputRef
			break
		}
		if err := wait(ctx); err != nil {
			return fmt.Errorf("poll timeout: %w", err)
		}
	}
	result, _, err := provider.OpenResult(ctx, prepared.Job, prepared.Account)
	if err != nil {
		return fmt.Errorf("OpenResult: %w", err)
	}
	defer result.Close()
	if err := validateGeminiNB2ResultJSONL(result); err != nil {
		return err
	}
	completed = true
	return nil
}

func validateGeminiNB2ResultJSONL(r io.Reader) error {
	s := bufio.NewScanner(io.LimitReader(r, geminiNB2MaxResultBytes+1))
	s.Buffer(make([]byte, 64<<10), geminiNB2MaxResultBytes+1)
	var items []map[string]any
	for s.Scan() {
		if strings.TrimSpace(s.Text()) == "" {
			continue
		}
		var item map[string]any
		if err := json.Unmarshal(s.Bytes(), &item); err != nil {
			return fmt.Errorf("invalid result JSONL: %w", err)
		}
		items = append(items, item)
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("result exceeds %d bytes or is unreadable: %w", geminiNB2MaxResultBytes, err)
	}
	if len(items) != 1 {
		return fmt.Errorf("expected exactly one result item, got %d", len(items))
	}
	if containsGeminiResultError(items[0]) {
		return errors.New("result item contains an error")
	}
	images := collectGeminiInlineImages(items[0])
	if len(images) == 0 {
		return errors.New("result item contains no inline image")
	}
	for _, inlineImage := range images {
		mime, _ := firstString(inlineImage, "mimeType", "mime_type")
		data, _ := firstString(inlineImage, "data")
		if !strings.HasPrefix(strings.ToLower(mime), "image/") {
			return fmt.Errorf("invalid image mime %q", mime)
		}
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return fmt.Errorf("invalid image base64: %w", err)
		}
		if len(decoded) == 0 || len(decoded) > geminiNB2MaxResultBytes {
			return fmt.Errorf("decoded image size %d outside allowed range", len(decoded))
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(decoded))
		if err != nil {
			return fmt.Errorf("decoded payload is not a supported image: %w", err)
		}
		if config.Width <= 0 || config.Height <= 0 {
			return fmt.Errorf("decoded image dimensions %dx%d are invalid", config.Width, config.Height)
		}
	}
	return nil
}

func containsGeminiResultError(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if strings.EqualFold(k, "error") && child != nil && fmt.Sprint(child) != "map[]" && fmt.Sprint(child) != "" {
				return true
			}
			if containsGeminiResultError(child) {
				return true
			}
		}
	case []any:
		for _, child := range x {
			if containsGeminiResultError(child) {
				return true
			}
		}
	}
	return false
}

func collectGeminiInlineImages(v any) []map[string]any {
	var out []map[string]any
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if k == "inlineData" || k == "inline_data" {
				if m, ok := child.(map[string]any); ok {
					out = append(out, m)
				}
			}
			out = append(out, collectGeminiInlineImages(child)...)
		}
	case []any:
		for _, child := range x {
			out = append(out, collectGeminiInlineImages(child)...)
		}
	}
	return out
}

func firstString(m map[string]any, keys ...string) (string, bool) {
	for _, k := range keys {
		if s, ok := m[k].(string); ok {
			return strings.TrimSpace(s), true
		}
	}
	return "", false
}

type scriptedGeminiNB2Provider struct {
	statuses                   []*BatchProviderStatus
	getErr, openErr, cancelErr error
	result                     string
	calls                      []string
	cancelContextErr           error
}

func (p *scriptedGeminiNB2Provider) Submit(context.Context, *BatchImageJob, *Account, BatchImageInput) (*BatchProviderJob, error) {
	p.calls = append(p.calls, "Submit")
	return &BatchProviderJob{ProviderJobName: "batches/1", ProviderInputRef: "files/in"}, nil
}
func (p *scriptedGeminiNB2Provider) Get(context.Context, *BatchImageJob, *Account) (*BatchProviderStatus, error) {
	p.calls = append(p.calls, "Get")
	if p.getErr != nil {
		return nil, p.getErr
	}
	s := p.statuses[0]
	p.statuses = p.statuses[1:]
	return s, nil
}
func (p *scriptedGeminiNB2Provider) OpenResult(context.Context, *BatchImageJob, *Account) (io.ReadCloser, string, error) {
	p.calls = append(p.calls, "OpenResult")
	if p.openErr != nil {
		return nil, "", p.openErr
	}
	return io.NopCloser(strings.NewReader(p.result)), "application/jsonl", nil
}
func (p *scriptedGeminiNB2Provider) Cancel(ctx context.Context, _ *BatchImageJob, _ *Account) error {
	p.calls = append(p.calls, "Cancel")
	p.cancelContextErr = ctx.Err()
	return p.cancelErr
}

func preparedGeminiNB2ForFlow() *geminiNB2PreparedReview {
	return &geminiNB2PreparedReview{Account: &Account{}, Job: &BatchImageJob{}, Input: BatchImageInput{}}
}
func validGeminiNB2JSONL() string {
	var encoded bytes.Buffer
	_ = png.Encode(&encoded, image.NewNRGBA64(image.Rect(0, 0, 1, 1)))
	return `{"key":"scene-01","response":{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"` + base64.StdEncoding.EncodeToString(encoded.Bytes()) + `"}}]}}]}}` + "\n"
}

func TestRunGeminiNB2ProductFlow(t *testing.T) {
	t.Run("running succeeded open result", func(t *testing.T) {
		p := &scriptedGeminiNB2Provider{statuses: []*BatchProviderStatus{{InternalState: BatchProviderStateRunning}, {Done: true, InternalState: BatchProviderStateSucceeded, ProviderOutputRef: "files/out"}}, result: validGeminiNB2JSONL()}
		err := runGeminiNB2ProductFlow(context.Background(), p, preparedGeminiNB2ForFlow(), func(context.Context) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(p.calls, ","); got != "Submit,Get,Get,OpenResult" {
			t.Fatalf("calls=%s", got)
		}
	})
	for _, tc := range []struct {
		name string
		p    *scriptedGeminiNB2Provider
		wait func(context.Context) error
	}{
		{"terminal failed", &scriptedGeminiNB2Provider{statuses: []*BatchProviderStatus{{Done: true, InternalState: BatchProviderStateFailed}}}, func(context.Context) error { return nil }},
		{"get error", &scriptedGeminiNB2Provider{getErr: errors.New("get")}, func(context.Context) error { return nil }},
		{"timeout", &scriptedGeminiNB2Provider{statuses: []*BatchProviderStatus{{InternalState: BatchProviderStateRunning}}}, func(context.Context) error { return context.DeadlineExceeded }},
		{"open error", &scriptedGeminiNB2Provider{statuses: []*BatchProviderStatus{{Done: true, InternalState: BatchProviderStateSucceeded, ProviderOutputRef: "files/out"}}, openErr: errors.New("open")}, func(context.Context) error { return nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := runGeminiNB2ProductFlow(context.Background(), tc.p, preparedGeminiNB2ForFlow(), tc.wait); err == nil {
				t.Fatal("expected error")
			}
			if len(tc.p.calls) == 0 || tc.p.calls[len(tc.p.calls)-1] != "Cancel" {
				t.Fatalf("calls=%v", tc.p.calls)
			}
		})
	}
	t.Run("cancel failure is combined using independent context", func(t *testing.T) {
		p := &scriptedGeminiNB2Provider{getErr: errors.New("get failed"), cancelErr: errors.New("cancel transport failed")}
		err := runGeminiNB2ProductFlow(context.Background(), p, preparedGeminiNB2ForFlow(), func(context.Context) error { return nil })
		if err == nil || !strings.Contains(err.Error(), "GEMINI_NB2_CANCEL_UNCONFIRMED") || !strings.Contains(err.Error(), "cancel transport failed") || !strings.Contains(err.Error(), "get failed") {
			t.Fatalf("combined error=%v", err)
		}
		if p.cancelContextErr != nil {
			t.Fatalf("Cancel received an already-cancelled context: %v", p.cancelContextErr)
		}
	})
}

func TestValidateGeminiNB2ResultJSONL(t *testing.T) {
	if err := validateGeminiNB2ResultJSONL(strings.NewReader(validGeminiNB2JSONL())); err != nil {
		t.Fatal(err)
	}
	garbageImage := `{"inline_data":{"mime_type":"image/png","data":"` + base64.StdEncoding.EncodeToString([]byte("not an image")) + `"}}` + "\n"
	for name, body := range map[string]string{"two items": validGeminiNB2JSONL() + validGeminiNB2JSONL(), "error": `{"error":{"message":"bad"}}` + "\n", "no image": `{"response":{}}` + "\n", "bad base64": `{"inline_data":{"mime_type":"image/png","data":"%%%"}}` + "\n", "image mime with garbage bytes": garbageImage} {
		t.Run(name, func(t *testing.T) {
			if validateGeminiNB2ResultJSONL(strings.NewReader(body)) == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

type noNetworkGeminiBatchClient struct{}

func (noNetworkGeminiBatchClient) UploadJSONL(context.Context, string, string, io.Reader) (*GeminiUploadedFile, error) {
	return nil, errors.New("network disabled")
}
func (noNetworkGeminiBatchClient) CreateBatch(context.Context, string, string, string, string) (*GeminiBatchJob, error) {
	return nil, errors.New("network disabled")
}
func (noNetworkGeminiBatchClient) GetBatch(context.Context, string, string) (*GeminiBatchJob, error) {
	return nil, errors.New("network disabled")
}
func (noNetworkGeminiBatchClient) CancelBatch(context.Context, string, string) error {
	return errors.New("network disabled")
}
func (noNetworkGeminiBatchClient) DownloadFile(context.Context, string, string) (io.ReadCloser, string, error) {
	return nil, "", errors.New("network disabled")
}
func (noNetworkGeminiBatchClient) DeleteFile(context.Context, string, string) error {
	return errors.New("network disabled")
}

func TestGeminiNB2PreflightFixedReserveBeforeSecret(t *testing.T) {
	guard := newRealReviewSessionGuard(filepath.Join(t.TempDir(), "review.json"), true)
	var reads atomic.Int32
	if _, err := prepareGeminiNB2RealReview(geminiNB2RealReviewOptions{Guard: guard, PerCallCap: 4.99, ReadAPIKey: func() string { reads.Add(1); return "x" }, NewClient: func() GeminiBatchClient { return noNetworkGeminiBatchClient{} }}); err == nil || reads.Load() != 0 {
		t.Fatalf("err=%v reads=%d", err, reads.Load())
	}
	p, err := prepareGeminiNB2RealReview(geminiNB2RealReviewOptions{Guard: guard, PerCallCap: 5, ReadAPIKey: func() string { reads.Add(1); return "test-key" }, NewClient: func() GeminiBatchClient { return noNetworkGeminiBatchClient{} }})
	if err != nil {
		t.Fatal(err)
	}
	if p.Job.Model != geminiNB2RealReviewModel || len(p.Input.Items) != 1 {
		t.Fatal("model/item drift")
	}
	state, _ := guard.loadStateForTest()
	if state.ReservedCNY != 5 || state.ImageAttempts != 1 {
		t.Fatalf("state=%#v", state)
	}
}

func TestGeminiNB2SingleRealReviewFormA(t *testing.T) {
	if strings.TrimSpace(os.Getenv("SUB2API_GEMINI_NB2_FORMA_REAL_SMOKE_RUN")) != "1" {
		t.Skip("disabled; ordinary AI Studio key completion remains BLOCKED until authorized and proven")
	}
	cap := requiredPositiveEnvFloat(t, "SUB2API_GEMINI_NB2_PER_CALL_CAP_CNY")
	prepared, err := prepareGeminiNB2RealReview(geminiNB2RealReviewOptions{Guard: newRealReviewSessionGuard(strings.TrimSpace(os.Getenv("SUB2API_REAL_REVIEW_SESSION_STATE_PATH")), strings.TrimSpace(os.Getenv("SUB2API_REAL_REVIEW_SESSION_ENABLED")) == "1"), PerCallCap: cap, Prompt: strings.TrimSpace(os.Getenv("SUB2API_GEMINI_NB2_PROMPT")), ReadAPIKey: func() string { return strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) }, NewClient: func() GeminiBatchClient { return NewGeminiBatchHTTPClient("", nil) }})
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	provider := NewGeminiAPIBatchImageProvider(prepared.Client)
	if err := runGeminiNB2ProductFlow(ctx, provider, prepared, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(defaultGeminiBatchRequeueAfter):
			return nil
		}
	}); err != nil {
		t.Fatal(err)
	}
}

func requiredPositiveEnvFloat(t *testing.T, name string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(name)), 64)
	if err != nil || v <= 0 {
		t.Fatalf("%s must be positive", name)
	}
	return v
}
