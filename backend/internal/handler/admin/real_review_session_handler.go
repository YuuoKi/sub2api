package admin

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// RealReviewSessionHandler exposes admin-only real-review session capability status.
// It never returns state paths, credentials, or secrets.
type RealReviewSessionHandler struct {
	video *service.VideoGatewayService
}

func NewRealReviewSessionHandler(video *service.VideoGatewayService) *RealReviewSessionHandler {
	return &RealReviewSessionHandler{video: video}
}

type realReviewSessionStatusResponse struct {
	Enabled         bool   `json:"enabled"`
	ImageUsed       int    `json:"image_used"`
	ImageRemaining  int    `json:"image_remaining"`
	VideoUsed       int    `json:"video_used"`
	VideoRemaining  int    `json:"video_remaining"`
	ReservedCNY     string `json:"reserved_cny"`
	RemainingCNY    string `json:"remaining_cny"`
	PricingVersion  string `json:"pricing_version"`
}

func (h *RealReviewSessionHandler) GetStatus(c *gin.Context) {
	if _, ok := middleware.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	enabled := h.video != nil && h.video.RealReviewSessionEnabled()
	out := realReviewSessionStatusResponse{Enabled: enabled}
	if h.video == nil {
		response.Success(c, out)
		return
	}
	snap, err := h.video.RealCreateSnapshot(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out.ImageUsed = snap.ImageUsed
	out.ImageRemaining = snap.ImageRemaining
	out.VideoUsed = snap.VideoUsed
	out.VideoRemaining = snap.VideoRemaining
	out.ReservedCNY = snap.ReservedCNY.String()
	out.RemainingCNY = snap.RemainingCNY.String()
	out.PricingVersion = snap.PricingVersion
	response.Success(c, out)
}

// BootstrapCredentials upserts review_only accounts from env key presence.
// Response never includes secret values.
func (h *RealReviewSessionHandler) BootstrapCredentials(c *gin.Context) {
	if _, ok := middleware.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.video == nil {
		response.ErrorFrom(c, service.ErrReviewRealSessionDisabled)
		return
	}
	result, err := h.video.ReviewCredentialBootstrap(c.Request.Context())
	if result == nil {
		response.ErrorFrom(c, err)
		return
	}
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    "REAL_REVIEW_SESSION_DISABLED",
			"message": err.Error(),
			"data":    result,
		})
		return
	}
	response.Success(c, result)
}

// ClearReviewOnlyAccounts disables review_only bootstrap accounts.
func (h *RealReviewSessionHandler) ClearReviewOnlyAccounts(c *gin.Context) {
	if _, ok := middleware.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.video == nil {
		response.ErrorFrom(c, service.ErrReviewRealAccountUnavailable)
		return
	}
	result, err := h.video.ClearReviewOnlyAccounts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type realAccessPolicyResponse struct {
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
	GlobalKillSwitch bool   `json:"global_kill_switch"`
	AllowMember      bool   `json:"allow_member"`
	AllowGroup       bool   `json:"allow_group"`
	ImageDailyCNY    string `json:"image_daily_cny"`
	VideoDailyCNY    string `json:"video_daily_cny"`
	MonthlyCNY       string `json:"monthly_cny"`
	AuditActorEmail  string `json:"audit_actor_email,omitempty"`
}

type realAccessPolicyRequest struct {
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
	GlobalKillSwitch bool   `json:"global_kill_switch"`
	AllowMember      bool   `json:"allow_member"`
	AllowGroup       bool   `json:"allow_group"`
	ImageDailyCNY    string `json:"image_daily_cny"`
	VideoDailyCNY    string `json:"video_daily_cny"`
	MonthlyCNY       string `json:"monthly_cny"`
}

type killSwitchRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *RealReviewSessionHandler) GetRealAccessPolicy(c *gin.Context) {
	if _, ok := middleware.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.video == nil {
		response.ErrorFrom(c, service.ErrInternalRealPolicyDenied)
		return
	}
	policy, err := h.video.GetRealAccessPolicy(c.Request.Context(), "default")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, realAccessPolicyToResponse(policy))
}

func (h *RealReviewSessionHandler) PutRealAccessPolicy(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.video == nil {
		response.ErrorFrom(c, service.ErrInternalRealPolicyDenied)
		return
	}
	var req realAccessPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid policy payload")
		return
	}
	policy := &service.ProviderRealAccessPolicy{
		Name:             firstNonEmptyPolicyName(req.Name),
		Enabled:          req.Enabled,
		GlobalKillSwitch: req.GlobalKillSwitch,
		AllowMember:      req.AllowMember,
		AllowGroup:       req.AllowGroup,
		ImageDailyCNY:    parsePolicyDecimal(req.ImageDailyCNY),
		VideoDailyCNY:    parsePolicyDecimal(req.VideoDailyCNY),
		MonthlyCNY:       parsePolicyDecimal(req.MonthlyCNY),
	}
	actorID := subject.UserID
	policy.AuditActorID = &actorID
	policy.AuditActorEmail = ""
	if err := h.video.SaveRealAccessPolicy(c.Request.Context(), policy); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, realAccessPolicyToResponse(policy))
}

func (h *RealReviewSessionHandler) PutKillSwitch(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.video == nil {
		response.ErrorFrom(c, service.ErrInternalRealPolicyDenied)
		return
	}
	var req killSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid kill switch payload")
		return
	}
	policy, err := h.video.GetRealAccessPolicy(c.Request.Context(), "default")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	policy.GlobalKillSwitch = req.Enabled
	actorID := subject.UserID
	policy.AuditActorID = &actorID
	policy.AuditActorEmail = ""
	if err := h.video.SaveRealAccessPolicy(c.Request.Context(), policy); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, realAccessPolicyToResponse(policy))
}

func realAccessPolicyToResponse(policy *service.ProviderRealAccessPolicy) realAccessPolicyResponse {
	if policy == nil {
		return realAccessPolicyResponse{Name: "default"}
	}
	return realAccessPolicyResponse{
		Name:             policy.Name,
		Enabled:          policy.Enabled,
		GlobalKillSwitch: policy.GlobalKillSwitch,
		AllowMember:      policy.AllowMember,
		AllowGroup:       policy.AllowGroup,
		ImageDailyCNY:    policy.ImageDailyCNY.String(),
		VideoDailyCNY:    policy.VideoDailyCNY.String(),
		MonthlyCNY:       policy.MonthlyCNY.String(),
		AuditActorEmail:  policy.AuditActorEmail,
	}
}

func firstNonEmptyPolicyName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}

func parsePolicyDecimal(raw string) decimal.Decimal {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero
	}
	v, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero
	}
	return v
}
