package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	CredentialConnectivityOK      = "ok"
	CredentialConnectivityWarning = "warning"
	CredentialConnectivityError   = "error"

	CredentialAuthAccepted   = "accepted"
	CredentialAuthRejected   = "rejected"
	CredentialAuthUnverified = "unverified"

	credentialConnectivityMaxBodyBytes = int64(8 * 1024)
)

// CredentialConnectivityResult is deliberately narrower than a real model
// smoke test. It proves that the configured endpoint can be reached and reports
// whether the upstream rejected the credential, without creating a paid
// generation task.
type CredentialConnectivityResult struct {
	Status            string `json:"status"`
	Message           string `json:"message"`
	Supported         bool   `json:"supported"`
	Reachable         bool   `json:"reachable"`
	Authentication    string `json:"authentication"`
	LatencyMS         int64  `json:"latency_ms"`
	HTTPStatus        int    `json:"http_status,omitempty"`
	GenerationStarted bool   `json:"generation_started"`
	CheckedAt         string `json:"checked_at"`
}

type credentialConnectivityTarget struct {
	URL     string
	Headers http.Header
}

func newCredentialConnectivityHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.TLSHandshakeTimeout = 8 * time.Second
	transport.ResponseHeaderTimeout = 10 * time.Second
	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func unsupportedCredentialConnectivity(message string) *CredentialConnectivityResult {
	return &CredentialConnectivityResult{
		Status:            CredentialConnectivityWarning,
		Message:           message,
		Supported:         false,
		Authentication:    CredentialAuthUnverified,
		GenerationStarted: false,
		CheckedAt:         time.Now().UTC().Format(time.RFC3339),
	}
}

func checkCredentialConnectivity(
	ctx context.Context,
	client *http.Client,
	target credentialConnectivityTarget,
	secret string,
) *CredentialConnectivityResult {
	result := &CredentialConnectivityResult{
		Status:            CredentialConnectivityError,
		Message:           "连接检查失败",
		Supported:         true,
		Authentication:    CredentialAuthUnverified,
		GenerationStarted: false,
		CheckedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	if client == nil {
		client = newCredentialConnectivityHTTPClient()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		result.Message = "连通性检查配置无效"
		return result
	}
	for key, values := range target.Headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-store")
	request.Header.Set("User-Agent", "Sub2API-Credential-Connectivity/1.0")

	started := time.Now()
	response, err := client.Do(request)
	result.LatencyMS = time.Since(started).Milliseconds()
	if err != nil {
		result.Message = "无法连接上游接口，请检查网络、DNS 或接口地址"
		return result
	}
	defer response.Body.Close()
	result.Reachable = true
	result.HTTPStatus = response.StatusCode

	body, readErr := io.ReadAll(io.LimitReader(response.Body, credentialConnectivityMaxBodyBytes+1))
	if readErr != nil {
		result.Status = CredentialConnectivityWarning
		result.Message = "上游接口可达，但响应读取失败，鉴权状态未确认"
		return result
	}
	if int64(len(body)) > credentialConnectivityMaxBodyBytes {
		result.Status = CredentialConnectivityWarning
		result.Message = "上游接口可达，但响应过大，鉴权状态未确认"
		return result
	}
	if secret != "" && bytes.Contains(body, []byte(secret)) {
		result.Status = CredentialConnectivityError
		result.Message = "上游响应异常，已阻止凭证回显"
		return result
	}

	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden ||
		businessEnvelopeRejectsCredential(body) {
		result.Status = CredentialConnectivityError
		result.Authentication = CredentialAuthRejected
		result.Message = "接口可达，但上游拒绝了当前凭证"
		return result
	}

	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300,
		response.StatusCode == http.StatusBadRequest,
		response.StatusCode == http.StatusNotFound,
		response.StatusCode == http.StatusMethodNotAllowed,
		response.StatusCode == http.StatusUnprocessableEntity:
		result.Status = CredentialConnectivityOK
		result.Authentication = CredentialAuthAccepted
		result.Message = "接口可达，凭证未被上游拒绝；未创建生成任务"
	case response.StatusCode == http.StatusTooManyRequests:
		result.Status = CredentialConnectivityWarning
		result.Message = "接口可达，但上游限流，暂时无法确认凭证"
	default:
		result.Status = CredentialConnectivityWarning
		result.Message = fmt.Sprintf("接口可达，但上游返回 HTTP %d，鉴权状态未确认", response.StatusCode)
	}
	return result
}

func businessEnvelopeRejectsCredential(body []byte) bool {
	if len(bytes.TrimSpace(body)) == 0 {
		return false
	}
	var envelope struct {
		Code    any    `json:"code"`
		Msg     string `json:"msg"`
		Message string `json:"message"`
		Error   struct {
			Code    any    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return false
	}
	code := strings.ToLower(strings.TrimSpace(fmt.Sprint(envelope.Code)))
	errorCode := strings.ToLower(strings.TrimSpace(fmt.Sprint(envelope.Error.Code)))
	message := strings.ToLower(strings.Join([]string{envelope.Msg, envelope.Message, envelope.Error.Message}, " "))
	for _, value := range []string{code, errorCode} {
		if value == strconv.Itoa(http.StatusUnauthorized) || value == strconv.Itoa(http.StatusForbidden) ||
			strings.Contains(value, "unauthor") || strings.Contains(value, "forbidden") ||
			strings.Contains(value, "invalid_api_key") || strings.Contains(value, "invalid_token") {
			return true
		}
	}
	for _, marker := range []string{
		"unauthorized", "forbidden", "invalid api key", "invalid key", "invalid token",
		"api key is invalid", "token is invalid", "authentication failed", "authorization failed",
		"鉴权失败", "认证失败", "凭证无效", "密钥无效", "无效的 api key",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func connectivityProbeID() string {
	return "sub2-connectivity-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func (s *adminServiceImpl) CheckAccountConnectivity(ctx context.Context, id int64) (*CredentialConnectivityResult, error) {
	if id <= 0 {
		return nil, ErrAccountNotFound
	}
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if account.Platform != PlatformHCAtom || account.Type != AccountTypeAPIKey {
		return unsupportedCredentialConnectivity(
			"该平台没有无费用鉴权探针；为避免产生真实模型费用，本次未发送生成请求",
		), nil
	}
	apiKey, err := ResolveHCAtomAPIKey(account, s.hcAtomCredentialCipher)
	if err != nil {
		return nil, err
	}
	syncSpec, ok := LookupHCAtomModel(HCAtomCapabilityImageSync, HCAtomImageSeedreamModel)
	if !ok || !validHCAtomFixedRoute(syncSpec) {
		return nil, fmt.Errorf("HC-ATOM sync image connectivity route is unavailable")
	}
	syncHeaders := make(http.Header)
	syncHeaders.Set("Authorization", "Bearer "+apiKey)
	syncResult := checkCredentialConnectivity(ctx, s.connectivityClient, credentialConnectivityTarget{
		URL:     syncSpec.Origin + syncSpec.Path,
		Headers: syncHeaders,
	}, apiKey)

	asyncHeaders := make(http.Header)
	asyncHeaders.Set("Authorization", "Bearer "+apiKey)
	asyncHeaders.Set("x-api-key", apiKey)
	asyncResult := checkCredentialConnectivity(ctx, s.connectivityClient, credentialConnectivityTarget{
		URL:     hcAtomBatchOrigin + hcAtomBatchTaskPath + url.PathEscape(connectivityProbeID()),
		Headers: asyncHeaders,
	}, apiKey)
	return combineImageConnectivityResults(syncResult, asyncResult), nil
}

func combineImageConnectivityResults(syncResult, asyncResult *CredentialConnectivityResult) *CredentialConnectivityResult {
	result := &CredentialConnectivityResult{
		Status:            CredentialConnectivityOK,
		Message:           "同步与异步图片接口均可达，凭证未被上游拒绝；未创建生成任务",
		Supported:         true,
		Reachable:         syncResult.Reachable && asyncResult.Reachable,
		Authentication:    CredentialAuthAccepted,
		LatencyMS:         syncResult.LatencyMS + asyncResult.LatencyMS,
		GenerationStarted: false,
		CheckedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	if syncResult.Authentication == CredentialAuthRejected || asyncResult.Authentication == CredentialAuthRejected {
		result.Status = CredentialConnectivityError
		result.Authentication = CredentialAuthRejected
		result.Message = "图片接口可达，但至少一种图片协议拒绝了当前凭证"
		return result
	}
	if syncResult.Status == CredentialConnectivityError || asyncResult.Status == CredentialConnectivityError {
		result.Status = CredentialConnectivityError
		result.Authentication = CredentialAuthUnverified
		result.Message = "至少一种图片协议无法连接，请检查网络或上游服务"
		return result
	}
	if syncResult.Status == CredentialConnectivityWarning || asyncResult.Status == CredentialConnectivityWarning {
		result.Status = CredentialConnectivityWarning
		result.Authentication = CredentialAuthUnverified
		result.Message = "图片接口已响应，但至少一种图片协议的鉴权状态未确认"
	}
	return result
}

func (s *VideoAdminService) CheckProviderConnectivity(ctx context.Context, id int64) (*CredentialConnectivityResult, error) {
	if id <= 0 {
		return nil, ErrVideoProviderNotFound
	}
	providers, err := s.repo.ListVideoProviders(ctx)
	if err != nil {
		return nil, err
	}
	var provider *VideoProviderAccount
	for index := range providers {
		if providers[index].ID == id {
			provider = &providers[index]
			break
		}
	}
	if provider == nil {
		return nil, ErrVideoProviderNotFound
	}
	spec, ok := lookupVideoProvider(provider.Provider)
	if !ok || !spec.AdapterReady {
		return unsupportedCredentialConnectivity(
			"该视频平台没有安全探针；本次未创建生成任务",
		), nil
	}
	if s.encryptor == nil || strings.TrimSpace(provider.EncryptedAPIKey) == "" {
		return nil, fmt.Errorf("%w: video provider credential is unavailable", ErrVideoAdminInvalidRequest)
	}
	apiKey, err := s.encryptor.Decrypt(provider.EncryptedAPIKey)
	if err != nil || strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("%w: video provider credential is invalid", ErrVideoAdminInvalidRequest)
	}
	apiKey = strings.TrimSpace(apiKey)
	probeID := url.PathEscape(connectivityProbeID())
	var targetURL string
	switch provider.Provider {
	case "seedance":
		baseURL, validateErr := validateArkBaseURL(videoProviderBaseURL(spec, provider.BaseURL))
		if validateErr != nil {
			return nil, fmt.Errorf("%w: video provider endpoint is invalid", ErrVideoAdminInvalidRequest)
		}
		targetURL = arkTaskCollectionURL(baseURL) + "/" + probeID
	case HCAtomVideoV1Provider:
		model, found := LookupHCAtomModel(HCAtomCapabilityVideoV1, HCAtomVideoV1PublicModel)
		if !found || !validHCAtomFixedRoute(model) {
			return nil, fmt.Errorf("%w: HC-ATOM video V1 route is unavailable", ErrVideoAdminInvalidRequest)
		}
		targetURL = model.Origin + model.Path + "/" + probeID
	case HCAtomSeedanceV3Provider:
		targetURL = HCAtomSeedanceV3BaseURL + HCAtomSeedanceV3Path + "/" + probeID
	default:
		return unsupportedCredentialConnectivity(
			"该视频平台没有安全探针；本次未创建生成任务",
		), nil
	}
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+apiKey)
	return checkCredentialConnectivity(ctx, s.connectivityClient, credentialConnectivityTarget{
		URL:     targetURL,
		Headers: headers,
	}, apiKey), nil
}
