package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	_ "golang.org/x/image/webp"
)

const (
	hcAtomBatchOrigin            = "https://api-aigc.fzyinghe.com"
	hcAtomBatchCreatePath        = "/image/generation/tasks"
	hcAtomBatchTaskPath          = "/image/generation/tasks/"
	hcAtomBatchRequeueAfter      = 30 * time.Second
	hcAtomBatchMaxResponseBytes  = 1 << 20
	hcAtomBatchOverallTimeout    = 30 * time.Second
	hcAtomResultMaxRawImageBytes = 11 << 20
)

// HCAtomBatchClient has no fallback behaviour: every operation is tied to the
// HC fixed-origin protocol and callers only receive normalized task data.
type HCAtomBatchClient interface {
	Create(ctx context.Context, apiKey string, idempotencyKey string, req HCAtomBatchCreateRequest) (*HCAtomBatchTask, error)
	Get(ctx context.Context, apiKey string, taskID string, authScheme ...string) (*HCAtomBatchTask, error)
	Delete(ctx context.Context, apiKey string, taskID string, authScheme ...string) error
}

type HCAtomBatchCreateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Image   any    `json:"image,omitempty"`
	Size    string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
	N       int    `json:"n,omitempty"`
}

type HCAtomBatchTask struct {
	TaskID     string
	Status     string
	ResultURLs []string
	ResultURL  string
	ImageCount int
	ErrorCode  string
	ErrorMsg   string
	FailReason string
}

type HCAtomBatchImageProvider struct {
	client           HCAtomBatchClient
	resultClient     *http.Client
	credentialCipher HCAtomCredentialCipher
	ownedResultStore *HCAtomOwnedResultStore
}

func NewHCAtomBatchImageProvider(client HCAtomBatchClient) *HCAtomBatchImageProvider {
	return newHCAtomBatchImageProvider(client, nil, nil)
}

func NewHCAtomBatchImageProviderWithCredentialCipher(client HCAtomBatchClient, credentialCipher HCAtomCredentialCipher) *HCAtomBatchImageProvider {
	return newHCAtomBatchImageProvider(client, nil, credentialCipher)
}

func NewHCAtomBatchImageProviderWithResultClient(client HCAtomBatchClient, resultClient *http.Client) *HCAtomBatchImageProvider {
	return newHCAtomBatchImageProvider(client, resultClient, nil)
}

func NewHCAtomBatchImageProviderWithResultClientAndCredentialCipher(client HCAtomBatchClient, resultClient *http.Client, credentialCipher HCAtomCredentialCipher) *HCAtomBatchImageProvider {
	return newHCAtomBatchImageProvider(client, resultClient, credentialCipher)
}

func NewHCAtomBatchImageProviderWithOwnedResultStore(client HCAtomBatchClient, resultClient *http.Client, credentialCipher HCAtomCredentialCipher, ownedResultDir string) *HCAtomBatchImageProvider {
	p := newHCAtomBatchImageProvider(client, resultClient, credentialCipher)
	p.ownedResultStore = NewHCAtomOwnedResultStore(ownedResultDir)
	return p
}

func (p *HCAtomBatchImageProvider) OwnedResultDir() string {
	if p == nil || p.ownedResultStore == nil {
		return ""
	}
	return p.ownedResultStore.RootDir()
}

func newHCAtomBatchImageProvider(client HCAtomBatchClient, resultClient *http.Client, credentialCipher HCAtomCredentialCipher) *HCAtomBatchImageProvider {
	if client == nil {
		client = NewHCAtomBatchHTTPClient(nil)
	}
	if resultClient == nil {
		resultClient = newPublicAssetHTTPClient(2 * time.Minute)
	}
	resultClient = newHCAtomResultHTTPClient(resultClient)
	return &HCAtomBatchImageProvider{client: client, resultClient: resultClient, credentialCipher: credentialCipher}
}

func newHCAtomResultHTTPClient(base *http.Client) *http.Client {
	if base == nil {
		base = newPublicAssetHTTPClient(2 * time.Minute)
	}
	client := *base
	previousRedirectCheck := base.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many HC-ATOM result redirects")
		}
		if request == nil || validateHCAtomResultURL(request.URL) != nil {
			return errors.New("unsafe HC-ATOM result redirect")
		}
		if previousRedirectCheck != nil {
			return previousRedirectCheck(request, via)
		}
		return nil
	}
	return &client
}

func (p *HCAtomBatchImageProvider) Name() string { return BatchImageProviderHCAtom }

func (p *HCAtomBatchImageProvider) SupportsAccount(account *Account) bool {
	return p != nil && p.credentialCipher != nil && account != nil && account.Platform == PlatformHCAtom &&
		account.Type == AccountTypeAPIKey && credentialString(account.Credentials, HCAtomAPIKeyCiphertextField) != ""
}

func (p *HCAtomBatchImageProvider) Submit(ctx context.Context, job *BatchImageJob, account *Account, input BatchImageInput) (*BatchProviderJob, error) {
	if input.Model == "" && job != nil {
		input.Model = job.Model
	}
	spec, ok := LookupHCAtomModel(HCAtomCapabilityImageAsync, input.Model)
	if !ok || !validHCAtomFixedRoute(spec) {
		return nil, hcAtomBatchError("HC_ATOM_MODEL_UNSUPPORTED", "HC-ATOM batch image model is not enabled", nil)
	}
	if !p.SupportsAccount(account) {
		if account != nil && account.Platform == PlatformHCAtom && account.Type == AccountTypeAPIKey {
			return nil, ErrBatchImageProviderMissingAPIKey
		}
		return nil, ErrBatchImageProviderUnsupportedAccount
	}
	apiKey, err := p.resolveAPIKey(account)
	if err != nil {
		return nil, err
	}
	if input.BatchID == "" && job != nil {
		input.BatchID = job.BatchID
	}
	if len(input.Items) != 1 {
		return nil, hcAtomBatchError("HC_ATOM_SINGLE_ITEM_REQUIRED", "HC-ATOM batch image requests require exactly one item", nil)
	}
	item := input.Items[0]
	if strings.TrimSpace(item.Prompt) == "" {
		return nil, ErrBatchImageProviderInvalidInput
	}
	request, err := buildHCAtomBatchCreateRequest(spec, input, item)
	if err != nil {
		return nil, err
	}
	created, err := p.client.Create(ctx, apiKey, hcAtomBatchIdempotencyKey(input.BatchID), HCAtomBatchCreateRequest{
		Model:   request.Model,
		Prompt:  request.Prompt,
		Image:   request.Image,
		Size:    request.Size,
		Quality: request.Quality,
		N:       request.N,
	})
	if err != nil {
		return nil, mapHCAtomBatchError(err)
	}
	if created == nil || strings.TrimSpace(created.TaskID) == "" {
		return nil, hcAtomBatchError("HC_ATOM_INVALID_RESPONSE", "HC-ATOM create response is missing task id", nil)
	}
	return &BatchProviderJob{ProviderJobName: strings.TrimSpace(created.TaskID), ProviderInputRef: strings.TrimSpace(item.CustomID), RawState: strings.TrimSpace(created.Status)}, nil
}

func buildHCAtomBatchCreateRequest(spec HCAtomModelSpec, input BatchImageInput, item BatchImageInputItem) (HCAtomBatchCreateRequest, error) {
	request := HCAtomBatchCreateRequest{
		Model:  spec.UpstreamModel,
		Prompt: strings.TrimSpace(item.Prompt),
	}
	if request.Prompt == "" {
		return HCAtomBatchCreateRequest{}, ErrBatchImageProviderInvalidInput
	}
	if len(item.ReferenceImages) > 0 {
		images := make([]string, 0, len(item.ReferenceImages))
		for _, reference := range item.ReferenceImages {
			if len(reference.Data) != 0 || !isHCAtomHTTPSReferenceURL(reference.FileURI) {
				return HCAtomBatchCreateRequest{}, hcAtomBatchError("HC_ATOM_REFERENCE_IMAGE_INVALID", "HC-ATOM reference images must be public HTTPS URLs", nil)
			}
			images = append(images, strings.TrimSpace(reference.FileURI))
		}
		if len(images) == 1 {
			request.Image = images[0]
		} else {
			request.Image = images
		}
	}

	aspectRatio := strings.TrimSpace(input.AspectRatio)
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	switch spec.PublicModel {
	case HCAtomImageGeminiModel:
		if !hcAtomGeminiAspectRatioSupported(aspectRatio) {
			return HCAtomBatchCreateRequest{}, hcAtomBatchError("HC_ATOM_INVALID_ASPECT_RATIO", "HC-ATOM Gemini image aspect ratio is unsupported", nil)
		}
		request.Size = aspectRatio
		request.Quality = normalizeHCAtomGeminiQuality(input.ImageSize)
	case HCAtomImageGPTModel, HCAtomImageSGPTModel:
		size, ok := hcAtomGPTImageSize(aspectRatio, input.ImageSize)
		if !ok {
			return HCAtomBatchCreateRequest{}, hcAtomBatchError("HC_ATOM_INVALID_ASPECT_RATIO", "HC-ATOM GPT image aspect ratio is unsupported", nil)
		}
		request.Size = size
		request.Quality = "auto"
		request.N = 1
	default:
		return HCAtomBatchCreateRequest{}, hcAtomBatchError("HC_ATOM_MODEL_UNSUPPORTED", "HC-ATOM batch image model is not enabled", nil)
	}
	return request, nil
}

func hcAtomGeminiAspectRatioSupported(value string) bool {
	switch value {
	case "1:8", "1:4", "1:1", "2:3", "3:2", "3:4", "4:3", "4:5", "5:4", "9:16", "16:9", "21:9", "4:1", "8:1":
		return true
	default:
		return false
	}
}

func normalizeHCAtomGeminiQuality(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "0.5K":
		return "0.5K"
	case "2K":
		return "2K"
	case "4K":
		return "4K"
	default:
		return "1K"
	}
}

func hcAtomGPTImageSize(aspectRatio, imageSize string) (string, bool) {
	ratios := map[string]float64{
		"1:8": 1.0 / 8.0, "1:4": 1.0 / 4.0, "1:1": 1,
		"2:3": 2.0 / 3.0, "3:2": 3.0 / 2.0, "3:4": 3.0 / 4.0,
		"4:3": 4.0 / 3.0, "4:5": 4.0 / 5.0, "5:4": 5.0 / 4.0,
		"9:16": 9.0 / 16.0, "16:9": 16.0 / 9.0, "21:9": 21.0 / 9.0,
		"4:1": 4, "8:1": 8,
	}
	ratio, ok := ratios[aspectRatio]
	if !ok {
		return "", false
	}
	targetPixels := 1_048_576
	switch strings.ToUpper(strings.TrimSpace(imageSize)) {
	case "", "1K":
	case "2K":
		targetPixels = 4_194_304
	case "4K":
		// HC documents a hard limit below 8,294,400 pixels, so 4K is
		// represented by the largest ratio-preserving dimensions at that cap.
		targetPixels = 8_294_399
	default:
		return "", false
	}
	width := int(math.Floor(math.Sqrt(float64(targetPixels) * ratio)))
	height := int(math.Floor(float64(width) / ratio))
	for width > 0 && height > 0 && width*height > targetPixels {
		if width >= height {
			width--
		} else {
			height--
		}
	}
	if width <= 0 || height <= 0 {
		return "", false
	}
	return fmt.Sprintf("%dx%d", width, height), true
}

func hcAtomAsyncImageSpecForJob(job *BatchImageJob) (HCAtomModelSpec, bool) {
	if job == nil {
		return HCAtomModelSpec{}, false
	}
	spec, ok := LookupHCAtomModel(HCAtomCapabilityImageAsync, job.Model)
	return spec, ok && validHCAtomFixedRoute(spec)
}

func (p *HCAtomBatchImageProvider) Get(ctx context.Context, job *BatchImageJob, account *Account) (*BatchProviderStatus, error) {
	spec, ok := hcAtomAsyncImageSpecForJob(job)
	if !ok {
		return nil, hcAtomBatchError("HC_ATOM_MODEL_UNSUPPORTED", "HC-ATOM batch image model is not enabled", nil)
	}
	if !p.SupportsAccount(account) {
		return nil, ErrBatchImageProviderUnsupportedAccount
	}
	taskID := batchImageProviderJobName(job)
	if taskID == "" {
		return nil, ErrBatchImageProviderMissingJobName
	}
	apiKey, err := p.resolveAPIKey(account)
	if err != nil {
		return nil, err
	}
	task, err := p.client.Get(ctx, apiKey, taskID, spec.AuthScheme)
	if err != nil {
		return nil, mapHCAtomBatchError(err)
	}
	if task == nil {
		return nil, hcAtomBatchError("HC_ATOM_INVALID_RESPONSE", "HC-ATOM task response is empty", nil)
	}
	return mapHCAtomBatchState(task, apiKey)
}

func (p *HCAtomBatchImageProvider) Cancel(ctx context.Context, job *BatchImageJob, account *Account) error {
	spec, ok := hcAtomAsyncImageSpecForJob(job)
	if !ok {
		return hcAtomBatchError("HC_ATOM_MODEL_UNSUPPORTED", "HC-ATOM batch image model is not enabled", nil)
	}
	if !p.SupportsAccount(account) {
		return ErrBatchImageProviderUnsupportedAccount
	}
	taskID := batchImageProviderJobName(job)
	if taskID == "" {
		return ErrBatchImageProviderMissingJobName
	}
	apiKey, err := p.resolveAPIKey(account)
	if err != nil {
		return err
	}
	return mapHCAtomBatchError(p.client.Delete(ctx, apiKey, taskID, spec.AuthScheme))
}

func (p *HCAtomBatchImageProvider) OpenResult(ctx context.Context, job *BatchImageJob, account *Account) (io.ReadCloser, string, error) {
	if r, ok, err := p.recoverOwnedResult(job); err != nil {
		return nil, "", err
	} else if ok {
		return r, "application/jsonl", nil
	}
	if !p.SupportsAccount(account) {
		return nil, "", ErrBatchImageProviderUnsupportedAccount
	}
	spec, ok := hcAtomAsyncImageSpecForJob(job)
	if !ok {
		return nil, "", hcAtomBatchError("HC_ATOM_MODEL_UNSUPPORTED", "HC-ATOM batch image model is not enabled", nil)
	}
	taskID := batchImageProviderJobName(job)
	customID := batchImageProviderInputRef(job)
	if taskID == "" || customID == "" || p.resultClient == nil {
		return nil, "", ErrBatchImageProviderMissingResultRef
	}
	apiKey, err := p.resolveAPIKey(account)
	if err != nil {
		return nil, "", err
	}
	task, err := p.client.Get(ctx, apiKey, taskID, spec.AuthScheme)
	if err != nil {
		return nil, "", mapHCAtomBatchError(err)
	}
	status, err := mapHCAtomBatchState(task, apiKey)
	if err != nil || status.InternalState != BatchProviderStateSucceeded {
		if err != nil {
			return nil, "", err
		}
		return nil, "", hcAtomBatchError("HC_ATOM_RESULT_NOT_READY", "HC-ATOM result is not ready for archival", nil)
	}
	urls := hcAtomBatchResultURLs(task)
	if len(urls) == 0 {
		return nil, "", hcAtomBatchError("HC_ATOM_RESULT_MISSING", "HC-ATOM success response is missing result URLs", nil)
	}
	parts := make([]any, 0, len(urls))
	for _, rawURL := range urls {
		encoded, mimeType, err := p.archiveResultURL(ctx, rawURL)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, map[string]any{"inlineData": map[string]any{"mimeType": mimeType, "data": encoded}})
	}
	line := map[string]any{"custom_id": customID, "response": map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": parts}}}}}
	var lines bytes.Buffer
	if err := json.NewEncoder(&lines).Encode(line); err != nil {
		return nil, "", err
	}
	if lines.Len() >= batchImageJSONLMaxLineBytes {
		return nil, "", hcAtomBatchError("HC_ATOM_RESULT_TOO_LARGE", "HC-ATOM result cannot fit the owned JSONL boundary", nil)
	}
	if err := p.writeOwnedResult(job, lines.Bytes()); err != nil {
		return nil, "", err
	}
	return p.openOwnedResultAfterWrite(job)
}

func (p *HCAtomBatchImageProvider) archiveResultURL(ctx context.Context, rawURL string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || validateHCAtomResultURL(parsed) != nil {
		return "", "", hcAtomBatchError("HC_ATOM_RESULT_URL_UNSAFE", "HC-ATOM result URL is unsafe", nil)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", "", hcAtomBatchError("HC_ATOM_RESULT_DOWNLOAD_FAILED", "HC-ATOM result download failed", nil)
	}
	resp, err := p.resultClient.Do(req)
	if err != nil {
		return "", "", hcAtomBatchError("HC_ATOM_RESULT_DOWNLOAD_FAILED", "HC-ATOM result download failed", nil)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > hcAtomResultMaxRawImageBytes {
		return "", "", hcAtomBatchError("HC_ATOM_RESULT_DOWNLOAD_FAILED", "HC-ATOM result download failed", nil)
	}
	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if mimeType != "image/png" && mimeType != "image/jpeg" && mimeType != "image/webp" {
		return "", "", hcAtomBatchError("HC_ATOM_RESULT_MIME_INVALID", "HC-ATOM result is not an allowed image", nil)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, hcAtomResultMaxRawImageBytes+1))
	if err != nil || len(data) == 0 || len(data) > hcAtomResultMaxRawImageBytes || !hcAtomImageSignatureMatches(mimeType, data) || !hcAtomImageDimensionsAllowed(mimeType, data) {
		return "", "", hcAtomBatchError("HC_ATOM_RESULT_CONTENT_INVALID", "HC-ATOM result image is invalid", nil)
	}
	return base64.StdEncoding.EncodeToString(data), mimeType, nil
}

func validateHCAtomResultURL(parsed *url.URL) error {
	if parsed == nil || parsed.Fragment != "" || (parsed.Port() != "" && parsed.Port() != "443") {
		return errors.New("HC-ATOM result URL is unsafe")
	}
	if err := validatePublicHTTPSAssetURL(parsed.String()); err != nil {
		return err
	}
	return validateAssetSourceURL(parsed)
}

func hcAtomImageDimensionsAllowed(mimeType string, data []byte) bool {
	// PNG dimensions live in the fixed IHDR header, so this check rejects image
	// bombs before invoking a decoder that may allocate based on those values.
	if mimeType == "image/png" {
		if len(data) < 24 || string(data[12:16]) != "IHDR" {
			return false
		}
		width, height := binary.BigEndian.Uint32(data[16:20]), binary.BigEndian.Uint32(data[20:24])
		return hcAtomImageDimensionsWithinLimit(uint64(width), uint64(height))
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (mimeType == "image/jpeg" && format != "jpeg") || (mimeType == "image/webp" && format != "webp") {
		return false
	}
	return hcAtomImageDimensionsWithinLimit(uint64(config.Width), uint64(config.Height))
}

func hcAtomImageDimensionsWithinLimit(width, height uint64) bool {
	return width > 0 && height > 0 && width <= 10000 && height <= 10000 && width*height <= 40_000_000
}

func hcAtomImageSignatureMatches(mimeType string, data []byte) bool {
	switch mimeType {
	case "image/png":
		return len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	case "image/jpeg":
		return len(data) >= 4 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff &&
			data[len(data)-2] == 0xff && data[len(data)-1] == 0xd9
	case "image/webp":
		if len(data) < 20 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" ||
			uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
			return false
		}
		return hcAtomWebPHasCompleteImageChunk(data)
	default:
		return false
	}
}

func hcAtomWebPHasCompleteImageChunk(data []byte) bool {
	hasImageChunk := false
	for offset := 12; offset < len(data); {
		if offset+8 > len(data) {
			return false
		}
		kind := string(data[offset : offset+4])
		size := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		end := uint64(offset+8) + size
		if end > uint64(len(data)) {
			return false
		}
		if kind == "VP8 " || kind == "VP8L" {
			hasImageChunk = true
		}
		offset = int(end + size%2)
		if offset > len(data) {
			return false
		}
	}
	return hasImageChunk
}

func (p *HCAtomBatchImageProvider) Cleanup(_ context.Context, job *BatchImageJob, _ *Account, target CleanupTarget) error {
	if target != CleanupTargetOutput || p == nil || p.ownedResultStore == nil {
		return nil
	}
	_, err := p.ownedResultStore.DeleteReferenced(job)
	return err
}

func (p *HCAtomBatchImageProvider) openOwnedResult(job *BatchImageJob) (io.ReadCloser, bool, error) {
	if p == nil || p.ownedResultStore == nil {
		return nil, false, nil
	}
	return p.ownedResultStore.OpenReferenced(job)
}

func (p *HCAtomBatchImageProvider) recoverOwnedResult(job *BatchImageJob) (io.ReadCloser, bool, error) {
	if p == nil || p.ownedResultStore == nil {
		return nil, false, nil
	}
	return p.ownedResultStore.Recover(job)
}

func (p *HCAtomBatchImageProvider) openOwnedResultAfterWrite(job *BatchImageJob) (io.ReadCloser, string, error) {
	r, ok, err := p.openOwnedResult(job)
	if err != nil || !ok {
		if err == nil {
			err = hcAtomBatchError("HC_ATOM_OWNED_RESULT_OPEN_FAILED", "HC-ATOM owned result is unavailable", nil)
		}
		return nil, "", err
	}
	return r, "application/jsonl", nil
}

func (p *HCAtomBatchImageProvider) writeOwnedResult(job *BatchImageJob, data []byte) error {
	if p == nil || p.ownedResultStore == nil {
		return hcAtomBatchError("HC_ATOM_OWNED_RESULT_STORE_UNAVAILABLE", "HC-ATOM owned result store is unavailable", nil)
	}
	return p.ownedResultStore.Write(job, data)
}

func hcAtomOwnedResultRef(batchID string) string { return "hc_atom_owned:" + batchID }

func hcAtomSafeOwnedResultID(value string) bool {
	if strings.TrimSpace(value) == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func (p *HCAtomBatchImageProvider) resolveAPIKey(account *Account) (string, error) {
	apiKey, err := ResolveHCAtomAPIKey(account, p.credentialCipher)
	if err != nil {
		return "", hcAtomBatchError("HC_ATOM_CREDENTIAL_UNAVAILABLE", "HC-ATOM account credential is unavailable", nil)
	}
	return apiKey, nil
}

func hcAtomBatchIdempotencyKey(batchID string) string {
	return "hc-image:" + strings.TrimSpace(batchID)
}

func mapHCAtomBatchState(task *HCAtomBatchTask, secrets ...string) (*BatchProviderStatus, error) {
	state := strings.ToUpper(strings.TrimSpace(task.Status))
	status := &BatchProviderStatus{RawState: state, SuggestedRequeueAfter: hcAtomBatchRequeueAfter}
	switch state {
	case "PENDING", "QUEUED":
		status.InternalState = BatchProviderStateQueued
	case "RUNNING":
		status.InternalState = BatchProviderStateRunning
	case "SUCCESS":
		resultCount := len(hcAtomBatchResultURLs(task))
		if resultCount == 0 {
			return nil, hcAtomBatchError("HC_ATOM_RESULT_MISSING", "HC-ATOM success response is missing result URLs", nil)
		}
		if task.ImageCount < 0 || (task.ImageCount > 0 && task.ImageCount != resultCount) {
			return nil, hcAtomBatchError("HC_ATOM_USAGE_MISMATCH", "HC-ATOM usage image count does not match result URLs", nil)
		}
		status.InternalState, status.Done = BatchProviderStateSucceeded, true
	case "FAILED":
		status.InternalState, status.Done = BatchProviderStateFailed, true
		status.ErrorCode, status.ErrorMessage = hcAtomBatchFailure(task, secrets...)
	case "CANCELLED":
		status.InternalState, status.Done = BatchProviderStateCancelled, true
		status.ErrorCode, status.ErrorMessage = "HC_ATOM_CANCELLED", "HC-ATOM task was cancelled"
	default:
		return nil, hcAtomBatchError("HC_ATOM_PROTOCOL_ERROR", "HC-ATOM task returned an unknown status", nil)
	}
	return status, nil
}

func hcAtomBatchFailure(task *HCAtomBatchTask, secrets ...string) (string, string) {
	// HC fields are provider-controlled. Keep actionable plain-language failures,
	// but redact the selected credential first and fall back whenever the
	// remainder contains secret-like text or signed URLs.
	return sanitizeBatchImageProviderFailure(
		RedactVideoSecrets(task.ErrorCode, secrets...),
		RedactVideoSecrets(firstHCAtomString(task.ErrorMsg, task.FailReason), secrets...),
		"HC_ATOM_TASK_FAILED",
		"HC-ATOM task failed",
	)
}

func hcAtomBatchResultURLs(task *HCAtomBatchTask) []string {
	if task == nil {
		return nil
	}
	seen, out := make(map[string]struct{}), make([]string, 0, len(task.ResultURLs)+1)
	for _, raw := range append(append([]string{}, task.ResultURLs...), task.ResultURL) {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func hcAtomBatchError(reason, message string, cause error) error {
	e := infraerrors.New(http.StatusBadGateway, reason, message)
	if cause != nil {
		return e.WithCause(cause)
	}
	return e
}

type HCAtomBatchHTTPClient struct{ client *http.Client }

func NewHCAtomBatchHTTPClient(client *http.Client) *HCAtomBatchHTTPClient {
	if client == nil {
		client = newHCAtomBatchProductionHTTPClient()
	}
	return &HCAtomBatchHTTPClient{client: client}
}

func newHCAtomBatchProductionHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ResponseHeaderTimeout = 15 * time.Second
	return &http.Client{
		Transport: transport,
		Timeout:   hcAtomBatchOverallTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (c *HCAtomBatchHTTPClient) Create(ctx context.Context, apiKey, idempotencyKey string, body HCAtomBatchCreateRequest) (*HCAtomBatchTask, error) {
	spec, ok := LookupHCAtomModel(HCAtomCapabilityImageAsync, body.Model)
	if !ok || !validHCAtomFixedRoute(spec) {
		return nil, hcAtomBatchError("HC_ATOM_MODEL_UNSUPPORTED", "HC-ATOM batch image model is not enabled", nil)
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, hcAtomBatchCreatePath, apiKey, bytes.NewReader(payload), spec.AuthScheme)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	return c.doTask(req)
}

func (c *HCAtomBatchHTTPClient) Get(ctx context.Context, apiKey, taskID string, authScheme ...string) (*HCAtomBatchTask, error) {
	req, err := c.newRequest(ctx, http.MethodGet, hcAtomBatchTaskPath+url.PathEscape(taskID), apiKey, nil, authScheme...)
	if err != nil {
		return nil, err
	}
	return c.doTask(req)
}
func (c *HCAtomBatchHTTPClient) Delete(ctx context.Context, apiKey, taskID string, authScheme ...string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, hcAtomBatchTaskPath+url.PathEscape(taskID), apiKey, nil, authScheme...)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &HCAtomBatchAPIError{StatusCode: resp.StatusCode}
	}
	body, err := readHCAtomBatchResponseBody(resp)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var envelope struct {
		Code any `json:"code"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return err
	}
	if !hcAtomBatchBusinessSuccess(envelope.Code) {
		return &HCAtomBatchAPIError{Code: "HC_ATOM_BUSINESS_ERROR"}
	}
	return nil
}

func (c *HCAtomBatchHTTPClient) newRequest(ctx context.Context, method, path, apiKey string, body io.Reader, authSchemes ...string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, hcAtomBatchOrigin+path, body)
	if err != nil {
		return nil, err
	}
	authScheme := HCAtomAuthBearer
	if len(authSchemes) > 0 && strings.TrimSpace(authSchemes[0]) != "" {
		authScheme = authSchemes[0]
	}
	applyHCAtomAuthHeader(req.Header, authScheme, apiKey)
	return req, nil
}

func (c *HCAtomBatchHTTPClient) doTask(req *http.Request) (*HCAtomBatchTask, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HCAtomBatchAPIError{StatusCode: resp.StatusCode}
	}
	var envelope struct {
		Code any             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	body, err := readHCAtomBatchResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if !hcAtomBatchBusinessSuccess(envelope.Code) {
		return nil, &HCAtomBatchAPIError{Code: "HC_ATOM_BUSINESS_ERROR"}
	}
	var data struct {
		TaskID          string   `json:"taskId"`
		TaskIDSnake     string   `json:"task_id"`
		Status          string   `json:"status"`
		ResultURLs      []string `json:"resultUrls"`
		ResultURLsSnake []string `json:"result_urls"`
		ResultURL       string   `json:"resultUrl"`
		ResultURLSnake  string   `json:"result_url"`
		Usage           struct {
			ImageCount int `json:"imageCount"`
		} `json:"usage"`
		ErrorCode  string `json:"errorCode"`
		ErrorMsg   string `json:"errorMsg"`
		FailReason string `json:"failReason"`
	}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, err
	}
	resultURLs := append([]string{}, data.ResultURLs...)
	resultURLs = append(resultURLs, data.ResultURLsSnake...)
	resultURLs = append(resultURLs, data.ResultURL, data.ResultURLSnake)
	return &HCAtomBatchTask{
		TaskID:     firstHCAtomString(data.TaskID, data.TaskIDSnake),
		Status:     data.Status,
		ResultURLs: resultURLs,
		ImageCount: data.Usage.ImageCount,
		ErrorCode:  data.ErrorCode,
		ErrorMsg:   data.ErrorMsg,
		FailReason: data.FailReason,
	}, nil
}

func readHCAtomBatchResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, errors.New("HC-ATOM response body is missing")
	}
	if resp.ContentLength > hcAtomBatchMaxResponseBytes {
		return nil, errors.New("HC-ATOM response body exceeds limit")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, hcAtomBatchMaxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > hcAtomBatchMaxResponseBytes {
		return nil, errors.New("HC-ATOM response body exceeds limit")
	}
	return body, nil
}

func firstHCAtomString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
func hcAtomBatchBusinessSuccess(code any) bool {
	switch value := code.(type) {
	case float64:
		return value == 0 || value == 200
	case string:
		value = strings.TrimSpace(value)
		return value == "0" || value == "200"
	default:
		return false
	}
}

type HCAtomBatchAPIError struct {
	StatusCode int
	Code       string
}

func (e *HCAtomBatchAPIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("HC-ATOM HTTP status %d", e.StatusCode)
	}
	return "HC-ATOM provider error"
}
func mapHCAtomBatchError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *HCAtomBatchAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return hcAtomBatchError("HC_ATOM_AUTH_FAILED", "HC-ATOM authentication failed", nil)
		case http.StatusTooManyRequests:
			return hcAtomBatchError("HC_ATOM_RATE_LIMITED", "HC-ATOM rate limit exceeded", nil)
		case 0:
			return hcAtomBatchError("HC_ATOM_BUSINESS_ERROR", "HC-ATOM rejected the request", nil)
		default:
			return hcAtomBatchError("HC_ATOM_UPSTREAM_ERROR", "HC-ATOM request failed", nil)
		}
	}
	return hcAtomBatchError("HC_ATOM_UPSTREAM_ERROR", "HC-ATOM request failed", nil)
}
