package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type qcanvasKeyContextResponse struct {
	Object     string                     `json:"object"`
	SubjectID  string                     `json:"subject_id"`
	GroupID    int64                      `json:"group_id"`
	ModelKinds []service.QCanvasModelKind `json:"model_kinds"`
}

// KeyContext returns the authenticated key's non-sensitive QCanvas capability
// identity. It never returns the key, group name, balance, pricing, or provider
// configuration.
func (h *GatewayHandler) KeyContext(c *gin.Context) {
	apiKey, ok := servermiddleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.User == nil || apiKey.Group == nil || apiKey.GroupID == nil ||
		apiKey.UserID <= 0 || apiKey.User.ID != apiKey.UserID ||
		*apiKey.GroupID <= 0 || apiKey.Group.ID != *apiKey.GroupID {
		servermiddleware.AbortWithError(c, http.StatusForbidden, "QC_KEY_CONTEXT_INCOMPLETE", "authenticated API key context is incomplete")
		return
	}

	subjectID := qcanvasUserSubjectID(apiKey.UserID)
	modelKinds := service.QCanvasModelKindsForGroup(apiKey.Group)
	if len(modelKinds) == 0 {
		servermiddleware.AbortWithError(c, http.StatusForbidden, "QC_KEY_CONTEXT_UNSUPPORTED", "authenticated API key is not a supported QCanvas credential")
		return
	}

	c.JSON(http.StatusOK, qcanvasKeyContextResponse{
		Object:     "api_key_context",
		SubjectID:  subjectID,
		GroupID:    *apiKey.GroupID,
		ModelKinds: modelKinds,
	})
}

func qcanvasUserSubjectID(userID int64) string {
	digest := sha256.Sum256([]byte("sub2api:qcanvas:subject:v1\x00" + strconv.FormatInt(userID, 10)))
	return "qcs_v1_" + hex.EncodeToString(digest[:16])
}
