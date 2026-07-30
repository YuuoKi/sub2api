package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HCAtomImages keeps the normal API-key, group, billing, concurrency, account
// selection, and usage-recording chain, changing only the selected upstream
// transport after an HC account has been chosen.
func (h *GatewayHandler) HCAtomImages(c *gin.Context) {
	started := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	body, err := readLenientJSONRequestBodyWithPrealloc(c.Request, h.cfg)
	if err != nil || len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Invalid request body")
		return
	}
	var request struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	if json.Unmarshal(body, &request) != nil || strings.TrimSpace(request.Prompt) == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model and prompt are required")
		return
	}
	if h.cfg == nil || !h.cfg.HCAtom.SyncImageEnabled {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "HC-ATOM sync image is disabled")
		return
	}
	if _, enabled := service.LookupHCAtomModel(service.HCAtomCapabilityImageSync, request.Model); !enabled {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is not enabled for sync image")
		return
	}
	if !service.GroupAllowsImageGeneration(apiKey.Group) {
		h.errorResponse(c, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage())
		return
	}
	setOpsRequestContext(c, request.Model, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.errorResponse(c, status, code, message)
		return
	}
	streamStarted := false
	releaseUser, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, false, &streamStarted)
	if err != nil {
		h.handleConcurrencyError(c, err, "user", false)
		return
	}
	if releaseUser != nil {
		defer releaseUser()
	}
	selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, "", request.Model, nil, "", 0)
	if err != nil || selection == nil || selection.Account == nil || selection.Account.Platform != service.PlatformHCAtom {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available HC-ATOM account")
		return
	}
	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)
	releaseAccount := selection.ReleaseFunc
	if !selection.Acquired {
		if selection.WaitPlan == nil {
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available HC-ATOM account")
			return
		}
		releaseAccount, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, false, &streamStarted)
		if err != nil {
			h.handleConcurrencyError(c, err, "account", false)
			return
		}
	}
	if releaseAccount != nil {
		defer releaseAccount()
	}
	result, err := h.gatewayService.ForwardHCAtomSyncImage(c.Request.Context(), c, account, body)
	if err != nil {
		var failover *service.UpstreamFailoverError
		if errors.As(err, &failover) {
			h.errorResponse(c, http.StatusBadGateway, "upstream_error", "HC-ATOM upstream failed")
		} else {
			h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		}
		return
	}
	requestHash := service.HashUsageRequestPayload(body)
	h.submitUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
			Result: result, QuotaPlatform: service.QuotaPlatform(c.Request.Context(), apiKey),
			APIKey: apiKey, User: apiKey.User, Account: account, Subscription: subscription,
			InboundEndpoint: GetInboundEndpoint(c), UpstreamEndpoint: GetUpstreamEndpoint(c, account.Platform),
			UserAgent: c.GetHeader("User-Agent"), IPAddress: ip.GetClientIP(c), RequestPayloadHash: requestHash,
			APIKeyService: h.apiKeyService,
		}); err != nil {
			requestLogger(c, "handler.gateway.hc_atom_images", zap.Int64("account_id", account.ID)).Error("record_usage_failed", zap.Error(err))
		}
	})
	service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, time.Since(started).Milliseconds())
}
