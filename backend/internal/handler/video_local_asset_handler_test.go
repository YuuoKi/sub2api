package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ownedLocalAssetOpenerStub struct {
	asset  *service.VideoLocalAsset
	err    error
	userID int64
	taskID int64
}

func (s *ownedLocalAssetOpenerStub) OpenOwnedLocalAsset(_ context.Context, taskID, userID int64) (*service.VideoLocalAsset, error) {
	s.taskID, s.userID = taskID, userID
	return s.asset, s.err
}

func TestVideoGatewayLocalAssetUsesJWTSubjectAndServesRangeAttachment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := testHandlerMP4Payload()
	file, err := os.CreateTemp(t.TempDir(), "asset-*.mp4")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file.Name(), payload, 0o640))
	require.NoError(t, file.Close())
	file, err = os.Open(file.Name())
	require.NoError(t, err)
	opener := &ownedLocalAssetOpenerStub{asset: &service.VideoLocalAsset{File: file, SizeBytes: int64(len(payload)), ModTime: time.Now().UTC(), ContentType: "video/mp4"}}
	h := &VideoGatewayHandler{localAssets: opener}
	router := gin.New()
	router.GET("/api/v1/video/tasks/:id/local-asset", func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 91})
		c.Next()
	}, h.LocalAsset)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/video/tasks/42/local-asset", nil)
	request.Header.Set("Range", "bytes=0-7")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusPartialContent, recorder.Code)
	require.Equal(t, payload[:8], recorder.Body.Bytes())
	require.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "attachment")
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "video-task-42.mp4")
	require.EqualValues(t, 91, opener.userID)
	require.EqualValues(t, 42, opener.taskID)
}

func TestVideoGatewayLocalAssetMapsForeignAndMissingWithoutPathLeak(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "foreign", err: service.ErrVideoTaskForbidden, status: http.StatusForbidden},
		{name: "missing", err: service.ErrVideoLocalAssetNotFound, status: http.StatusNotFound},
		{name: "internal", err: errors.New(`open D:\\secret\\result.mp4 failed`), status: http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := &VideoGatewayHandler{localAssets: &ownedLocalAssetOpenerStub{err: tt.err}}
			router := gin.New()
			router.GET("/tasks/:id/local-asset", func(c *gin.Context) {
				c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 9})
				c.Next()
			}, h.LocalAsset)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tasks/42/local-asset", nil))
			require.Equal(t, tt.status, recorder.Code)
			require.NotContains(t, recorder.Body.String(), "secret")
			require.NotContains(t, recorder.Body.String(), "result.mp4")
		})
	}
}

func testHandlerMP4Payload() []byte {
	return []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 2, 0, 'i', 's', 'o', 'm', 'i', 's', 'o', '2', 'd', 'a', 't', 'a'}
}
