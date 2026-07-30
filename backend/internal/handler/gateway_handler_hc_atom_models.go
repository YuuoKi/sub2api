package handler

import (
	"net/http"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) HCAtomModels(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	models := h.gatewayService.AuthorizedHCAtomPublicModels(c.Request.Context(), apiKey, c.Query("kind"))
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": models})
}
