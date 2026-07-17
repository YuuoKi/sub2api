package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type simulationResultOpenerStub struct {
	result *service.VideoSimulationResult
	err    error
	taskID int64
	userID int64
	admin  bool
}

func (s *simulationResultOpenerStub) OpenSimulationResult(_ context.Context, taskID, userID int64) (*service.VideoSimulationResult, error) {
	s.taskID, s.userID, s.admin = taskID, userID, false
	return s.result, s.err
}

func (s *simulationResultOpenerStub) OpenSimulationResultAsAdmin(_ context.Context, taskID int64) (*service.VideoSimulationResult, error) {
	s.taskID, s.admin = taskID, true
	return s.result, s.err
}

func TestSimulationResultOwnerAndAdminCanPreviewAndDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>模拟视频结果</text></svg>`)
	opener := &simulationResultOpenerStub{result: &service.VideoSimulationResult{
		TaskID: 42, MediaKind: "image", ContentType: "image/svg+xml", Filename: "simulation-task-42.svg",
		Label: "模拟视频结果", Body: payload,
	}}
	h := NewVideoSimulationHandler(opener)

	t.Run("owner_preview", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/user/video/simulation/tasks/:id/result", func(c *gin.Context) {
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7})
			c.Next()
		}, h.Result)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/user/video/simulation/tasks/42/result", nil))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "image/svg+xml", recorder.Header().Get("Content-Type"))
		require.Equal(t, "image", recorder.Header().Get("X-Media-Kind"))
		require.Contains(t, recorder.Body.String(), "模拟视频结果")
		require.EqualValues(t, 7, opener.userID)
		require.False(t, opener.admin)
	})

	t.Run("owner_download", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/user/video/simulation/tasks/:id/result", func(c *gin.Context) {
			c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7})
			c.Next()
		}, h.Result)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user/video/simulation/tasks/42/result?download=1", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Contains(t, recorder.Header().Get("Content-Disposition"), "attachment")
		require.Contains(t, recorder.Header().Get("Content-Disposition"), "simulation-task-42.svg")
		body, err := io.ReadAll(recorder.Body)
		require.NoError(t, err)
		require.Equal(t, payload, body)
	})

	t.Run("admin_preview", func(t *testing.T) {
		admin := NewVideoSimulationAdminHandler(opener)
		router := gin.New()
		router.GET("/api/v1/admin/video/simulation/tasks/:id/result", admin.Result)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/admin/video/simulation/tasks/42/result", nil))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.True(t, opener.admin)
		require.Equal(t, "image/svg+xml", recorder.Header().Get("Content-Type"))
	})
}

func TestSimulationResultForeignUserIsForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opener := &simulationResultOpenerStub{err: service.ErrVideoTaskForbidden}
	h := NewVideoSimulationHandler(opener)
	router := gin.New()
	router.GET("/tasks/:id/result", func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 9})
		c.Next()
	}, h.Result)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tasks/42/result", nil))
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.NotContains(t, recorder.Body.String(), `D:\`)
	require.NotContains(t, strings.ToLower(recorder.Body.String()), "script")
}

func TestSimulationResultMapsMissingWithoutPathLeak(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opener := &simulationResultOpenerStub{err: errors.New(`open D:\secret\result.svg failed`)}
	h := NewVideoSimulationHandler(opener)
	router := gin.New()
	router.GET("/tasks/:id/result", func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7})
		c.Next()
	}, h.Result)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tasks/42/result", nil))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.NotContains(t, recorder.Body.String(), `D:\secret`)
}
