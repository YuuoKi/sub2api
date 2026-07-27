package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	hcAtomBatchOrigin       = "https://api-aigc.fzyinghe.com"
	hcAtomBatchCreatePath   = "/image/generation/tasks"
	hcAtomBatchTaskPath     = "/image/generation/tasks/"
	hcAtomBatchAPIKeyField  = "hc_atom_api_key"
	hcAtomBatchRequeueAfter = 30 * time.Second
)

var hcAtomBatchEnabledModels = map[string]struct{}{
	"seedream-5.0":            {},
	"doubao-seedream-5.0-pro": {},
}

// HCAtomBatchClient has no fallback behaviour: every operation is tied to the
// HC fixed-origin protocol and callers only receive normalized task data.
type HCAtomBatchClient interface {
	Create(ctx context.Context, apiKey string, idempotencyKey string, req HCAtomBatchCreateRequest) (*HCAtomBatchTask, error)
	Get(ctx context.Context, apiKey string, taskID string) (*HCAtomBatchTask, error)
	Delete(ctx context.Context, apiKey string, taskID string) error
}

type HCAtomBatchCreateRequest struct {
	Model      string         `json:"model"`
	Input      map[string]any `json:"input"`
	Parameters map[string]any `json:"parameters"`
}

type HCAtomBatchTask struct {
	TaskID     string
	Status     string
	ResultURLs []string
	ResultURL  string
	ImageCount int
	ErrorCode  string
	ErrorMsg   string
}

type HCAtomBatchImageProvider struct{ client HCAtomBatchClient }

func NewHCAtomBatchImageProvider(client HCAtomBatchClient) *HCAtomBatchImageProvider {
	if client == nil {
		client = NewHCAtomBatchHTTPClient(nil)
	}
	return &HCAtomBatchImageProvider{client: client}
}

func (p *HCAtomBatchImageProvider) Name() string { return BatchImageProviderHCAtom }

func (p *HCAtomBatchImageProvider) SupportsAccount(account *Account) bool {
	return account != nil && account.Platform == PlatformHCAtom && account.Type == AccountTypeAPIKey && hcAtomBatchAPIKey(account) != ""
}

func (p *HCAtomBatchImageProvider) Submit(ctx context.Context, job *BatchImageJob, account *Account, input BatchImageInput) (*BatchProviderJob, error) {
	if !p.SupportsAccount(account) {
		if account != nil && account.Platform == PlatformHCAtom && account.Type == AccountTypeAPIKey {
			return nil, ErrBatchImageProviderMissingAPIKey
		}
		return nil, ErrBatchImageProviderUnsupportedAccount
	}
	if input.BatchID == "" && job != nil {
		input.BatchID = job.BatchID
	}
	if input.Model == "" && job != nil {
		input.Model = job.Model
	}
	if !isHCAtomBatchEnabledModel(input.Model) {
		return nil, hcAtomBatchError("HC_ATOM_MODEL_UNSUPPORTED", "HC-ATOM batch image model is not enabled", nil)
	}
	if len(input.Items) != 1 {
		return nil, hcAtomBatchError("HC_ATOM_SINGLE_ITEM_REQUIRED", "HC-ATOM batch image requests require exactly one item", nil)
	}
	item := input.Items[0]
	if strings.TrimSpace(item.Prompt) == "" {
		return nil, ErrBatchImageProviderInvalidInput
	}
	if len(item.ReferenceImages) != 0 {
		return nil, hcAtomBatchError("HC_ATOM_REFERENCE_IMAGES_UNSUPPORTED", "HC-ATOM batch image reference images are not enabled", nil)
	}
	created, err := p.client.Create(ctx, hcAtomBatchAPIKey(account), hcAtomBatchIdempotencyKey(input.BatchID), HCAtomBatchCreateRequest{
		Model:      input.Model,
		Input:      map[string]any{"prompt": item.Prompt},
		Parameters: map[string]any{"response_mime_type": input.ResponseMimeType, "aspect_ratio": input.AspectRatio, "image_size": input.ImageSize},
	})
	if err != nil {
		return nil, mapHCAtomBatchError(err)
	}
	if created == nil || strings.TrimSpace(created.TaskID) == "" {
		return nil, hcAtomBatchError("HC_ATOM_INVALID_RESPONSE", "HC-ATOM create response is missing task id", nil)
	}
	return &BatchProviderJob{ProviderJobName: strings.TrimSpace(created.TaskID), RawState: strings.TrimSpace(created.Status)}, nil
}

func (p *HCAtomBatchImageProvider) Get(ctx context.Context, job *BatchImageJob, account *Account) (*BatchProviderStatus, error) {
	if !p.SupportsAccount(account) {
		return nil, ErrBatchImageProviderUnsupportedAccount
	}
	taskID := batchImageProviderJobName(job)
	if taskID == "" {
		return nil, ErrBatchImageProviderMissingJobName
	}
	task, err := p.client.Get(ctx, hcAtomBatchAPIKey(account), taskID)
	if err != nil {
		return nil, mapHCAtomBatchError(err)
	}
	if task == nil {
		return nil, hcAtomBatchError("HC_ATOM_INVALID_RESPONSE", "HC-ATOM task response is empty", nil)
	}
	return mapHCAtomBatchState(task)
}

func (p *HCAtomBatchImageProvider) Cancel(ctx context.Context, job *BatchImageJob, account *Account) error {
	if !p.SupportsAccount(account) {
		return ErrBatchImageProviderUnsupportedAccount
	}
	taskID := batchImageProviderJobName(job)
	if taskID == "" {
		return ErrBatchImageProviderMissingJobName
	}
	return mapHCAtomBatchError(p.client.Delete(ctx, hcAtomBatchAPIKey(account), taskID))
}

func (p *HCAtomBatchImageProvider) OpenResult(context.Context, *BatchImageJob, *Account) (io.ReadCloser, string, error) {
	return nil, "", hcAtomBatchError("HC_ATOM_ARCHIVE_REQUIRED", "HC-ATOM result must be archived before it can be opened", nil)
}

func (p *HCAtomBatchImageProvider) Cleanup(context.Context, *BatchImageJob, *Account, CleanupTarget) error {
	return nil
}

func hcAtomBatchAPIKey(account *Account) string {
	if account == nil {
		return ""
	}
	return strings.TrimSpace(account.GetCredential(hcAtomBatchAPIKeyField))
}

func hcAtomBatchIdempotencyKey(batchID string) string {
	return "hc-image:" + strings.TrimSpace(batchID)
}

func isHCAtomBatchEnabledModel(model string) bool {
	_, ok := hcAtomBatchEnabledModels[strings.TrimSpace(model)]
	return ok
}

func mapHCAtomBatchState(task *HCAtomBatchTask) (*BatchProviderStatus, error) {
	state := strings.ToUpper(strings.TrimSpace(task.Status))
	status := &BatchProviderStatus{RawState: state, SuggestedRequeueAfter: hcAtomBatchRequeueAfter}
	switch state {
	case "PENDING":
		status.InternalState = BatchProviderStateQueued
	case "RUNNING":
		status.InternalState = BatchProviderStateRunning
	case "SUCCESS":
		if len(hcAtomBatchResultURLs(task)) == 0 {
			return nil, hcAtomBatchError("HC_ATOM_RESULT_MISSING", "HC-ATOM success response is missing result URLs", nil)
		}
		status.InternalState, status.Done = BatchProviderStateSucceeded, true
	case "FAILED":
		status.InternalState, status.Done = BatchProviderStateFailed, true
		status.ErrorCode, status.ErrorMessage = hcAtomBatchFailure(task)
	case "CANCELLED":
		status.InternalState, status.Done = BatchProviderStateCancelled, true
		status.ErrorCode, status.ErrorMessage = "HC_ATOM_CANCELLED", "HC-ATOM task was cancelled"
	default:
		return nil, hcAtomBatchError("HC_ATOM_PROTOCOL_ERROR", "HC-ATOM task returned an unknown status", nil)
	}
	return status, nil
}

func hcAtomBatchFailure(task *HCAtomBatchTask) (string, string) {
	code, msg := strings.TrimSpace(task.ErrorCode), strings.TrimSpace(task.ErrorMsg)
	if code == "" {
		code = "HC_ATOM_TASK_FAILED"
	}
	if msg == "" {
		msg = "HC-ATOM task failed"
	}
	return code, msg
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
		client = batchImageDefaultHTTPClient()
	}
	return &HCAtomBatchHTTPClient{client: client}
}

func (c *HCAtomBatchHTTPClient) Create(ctx context.Context, apiKey, idempotencyKey string, body HCAtomBatchCreateRequest) (*HCAtomBatchTask, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, hcAtomBatchCreatePath, apiKey, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	return c.doTask(req)
}

func (c *HCAtomBatchHTTPClient) Get(ctx context.Context, apiKey, taskID string) (*HCAtomBatchTask, error) {
	req, err := c.newRequest(ctx, http.MethodGet, hcAtomBatchTaskPath+url.PathEscape(taskID), apiKey, nil)
	if err != nil {
		return nil, err
	}
	return c.doTask(req)
}
func (c *HCAtomBatchHTTPClient) Delete(ctx context.Context, apiKey, taskID string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, hcAtomBatchTaskPath+url.PathEscape(taskID), apiKey, nil)
	if err != nil {
		return err
	}
	_, err = c.doTask(req)
	return err
}

func (c *HCAtomBatchHTTPClient) newRequest(ctx context.Context, method, path, apiKey string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, hcAtomBatchOrigin+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
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
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
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
		ErrorCode string `json:"errorCode"`
		ErrorMsg  string `json:"errorMsg"`
	}
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, err
	}
	return &HCAtomBatchTask{TaskID: firstHCAtomString(data.TaskID, data.TaskIDSnake), Status: data.Status, ResultURLs: append(data.ResultURLs, data.ResultURLsSnake...), ResultURL: firstHCAtomString(data.ResultURL, data.ResultURLSnake), ImageCount: data.Usage.ImageCount, ErrorCode: data.ErrorCode, ErrorMsg: data.ErrorMsg}, nil
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
	case nil:
		return true
	case float64:
		return value == 0
	case string:
		return strings.TrimSpace(value) == "" || strings.TrimSpace(value) == "0"
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
