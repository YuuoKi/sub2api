package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandlerKeyContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	price := func(value float64) *float64 { return &value }
	groupID := int64(31)
	userID := int64(21)
	validGroup := &service.Group{
		ID:              groupID,
		Platform:        service.PlatformHCAtom,
		Status:          service.StatusActive,
		VideoPrice480P:  price(0.05),
		VideoPrice720P:  price(0.07),
		VideoPrice1080P: price(0.25),
		ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{
			service.HCAtomVideoV1PublicModel,
		}},
	}
	validKey := &service.APIKey{
		ID:      41,
		UserID:  userID,
		Key:     "secret-must-not-be-returned",
		GroupID: &groupID,
		Status:  service.StatusActive,
		User:    &service.User{ID: userID, Status: service.StatusActive},
		Group:   validGroup,
	}

	t.Run("returns only structured non-sensitive identity", func(t *testing.T) {
		w := serveKeyContext(validKey)
		require.Equal(t, http.StatusOK, w.Code)
		var response qcanvasKeyContextResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		require.Equal(t, "api_key_context", response.Object)
		require.Equal(t, groupID, response.GroupID)
		require.Equal(t, []service.QCanvasModelKind{service.QCanvasModelKindVideo}, response.ModelKinds)
		require.Regexp(t, `^qcs_v1_[0-9a-f]{32}$`, response.SubjectID)
		require.JSONEq(t, `{
			"object":"api_key_context",
			"subject_id":"`+response.SubjectID+`",
			"group_id":31,
			"model_kinds":["video"]
		}`, w.Body.String())
		require.NotContains(t, w.Body.String(), validKey.Key)
		require.NotContains(t, w.Body.String(), "api_key_id")
		require.NotContains(t, w.Body.String(), "user_id")
		require.NotContains(t, w.Body.String(), "group_platform")
		require.NotContains(t, w.Body.String(), "api_key_status")

		repeated := serveKeyContext(validKey)
		var repeatedResponse qcanvasKeyContextResponse
		require.NoError(t, json.Unmarshal(repeated.Body.Bytes(), &repeatedResponse))
		require.Equal(t, response.SubjectID, repeatedResponse.SubjectID)

		siblingKey := *validKey
		siblingKey.ID++
		siblingKey.Key = "a-different-key-for-the-same-user"
		siblingGroupID := groupID + 1
		siblingKey.GroupID = &siblingGroupID
		siblingKey.Group = &service.Group{
			ID:                        siblingGroupID,
			Platform:                  service.PlatformHCAtom,
			Status:                    service.StatusActive,
			AllowImageGeneration:      true,
			AllowBatchImageGeneration: true,
			ImagePrice1K:              price(0.134),
			ImagePrice2K:              price(0.201),
			ImagePrice4K:              price(0.268),
			ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{
				service.HCAtomImageSeedreamModel,
				service.HCAtomImageDoubaoSeedreamModel,
				service.HCAtomImageGeminiModel,
				service.HCAtomImageGPTModel,
				service.HCAtomImageSGPTModel,
			}},
		}
		sibling := serveKeyContext(&siblingKey)
		var siblingResponse qcanvasKeyContextResponse
		require.NoError(t, json.Unmarshal(sibling.Body.Bytes(), &siblingResponse))
		require.Equal(t, response.SubjectID, siblingResponse.SubjectID)
		require.Equal(t, []service.QCanvasModelKind{service.QCanvasModelKindImage}, siblingResponse.ModelKinds)

		otherUserKey := siblingKey
		otherUserKey.UserID++
		otherUserKey.User = &service.User{ID: otherUserKey.UserID, Status: service.StatusActive}
		otherUser := serveKeyContext(&otherUserKey)
		var otherUserResponse qcanvasKeyContextResponse
		require.NoError(t, json.Unmarshal(otherUser.Body.Bytes(), &otherUserResponse))
		require.NotEqual(t, response.SubjectID, otherUserResponse.SubjectID)
	})

	t.Run("rejects incomplete identity", func(t *testing.T) {
		incomplete := *validKey
		incomplete.UserID = userID + 1
		w := serveKeyContext(&incomplete)
		require.Equal(t, http.StatusForbidden, w.Code)
		require.JSONEq(t, `{"code":"QC_KEY_CONTEXT_INCOMPLETE","message":"authenticated API key context is incomplete"}`, w.Body.String())
	})

	t.Run("rejects unsupported group", func(t *testing.T) {
		unsupported := *validKey
		unsupported.Group = &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive}
		w := serveKeyContext(&unsupported)
		require.Equal(t, http.StatusForbidden, w.Code)
		require.JSONEq(t, `{"code":"QC_KEY_CONTEXT_UNSUPPORTED","message":"authenticated API key is not a supported QCanvas credential"}`, w.Body.String())
	})
}

func serveKeyContext(apiKey *service.APIKey) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/v1/key-context", func(c *gin.Context) {
		if apiKey != nil {
			c.Set(string(servermiddleware.ContextKeyAPIKey), apiKey)
		}
		(&GatewayHandler{}).KeyContext(c)
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/key-context", nil))
	return w
}
