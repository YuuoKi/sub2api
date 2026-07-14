package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
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
