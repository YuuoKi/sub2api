package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminLocalAssetOpenerStub struct {
	asset  *service.VideoLocalAsset
	err    error
	taskID int64
}

func (s *adminLocalAssetOpenerStub) OpenLocalAsset(_ context.Context, taskID int64) (*service.VideoLocalAsset, error) {
	s.taskID = taskID
	return s.asset, s.err
}

func TestVideoAdminLocalAssetServesAttachment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := []byte("admin-video-bytes")
	file, err := os.CreateTemp(t.TempDir(), "asset-*.mp4")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file.Name(), payload, 0o640))
	require.NoError(t, file.Close())
	file, err = os.Open(file.Name())
	require.NoError(t, err)
	opener := &adminLocalAssetOpenerStub{asset: &service.VideoLocalAsset{File: file, SizeBytes: int64(len(payload)), ModTime: time.Now().UTC(), ContentType: "video/mp4"}}
	h := &VideoHandler{localAssets: opener}
	router := gin.New()
	router.GET("/tasks/:id/local-asset", h.LocalAsset)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tasks/51/local-asset", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, payload, recorder.Body.Bytes())
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "video-task-51.mp4")
	require.EqualValues(t, 51, opener.taskID)
}

func TestVideoAdminLocalAssetMissingIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &VideoHandler{localAssets: &adminLocalAssetOpenerStub{err: service.ErrVideoLocalAssetNotFound}}
	router := gin.New()
	router.GET("/tasks/:id/local-asset", h.LocalAsset)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tasks/51/local-asset", nil))
	require.Equal(t, http.StatusNotFound, recorder.Code)
}
