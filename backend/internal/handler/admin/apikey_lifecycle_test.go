package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type lifecycleAPIKeyManagerStub struct {
	keys             map[int64]*service.APIKey
	nextID           int64
	createErr        error
	updateErr        error
	deleteErr        error
	lastCreateUserID int64
	lastCreateReq    service.CreateAPIKeyRequest
	lastUpdateReq    service.UpdateAPIKeyRequest
	deletedID        int64
	deletedOwnerID   int64
	createCalls      int
	createPairCalls  int
	updateCalls      int
}

func newLifecycleAPIKeyManager(keys ...*service.APIKey) *lifecycleAPIKeyManagerStub {
	manager := &lifecycleAPIKeyManagerStub{keys: make(map[int64]*service.APIKey), nextID: 100}
	for _, key := range keys {
		clone := *key
		manager.keys[key.ID] = &clone
	}
	return manager
}

func (m *lifecycleAPIKeyManagerStub) Create(_ context.Context, userID int64, req service.CreateAPIKeyRequest) (*service.APIKey, error) {
	m.createCalls++
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.lastCreateUserID, m.lastCreateReq = userID, req
	m.nextID++
	key := &service.APIKey{ID: m.nextID, UserID: userID, Key: "sk-new-one-time-secret", Name: req.Name, Status: service.StatusActive}
	m.keys[key.ID] = key
	return key, nil
}

func (m *lifecycleAPIKeyManagerStub) CreateQCanvasKeyPair(_ context.Context, userID int64, req service.CreateQCanvasKeyPairRequest) (*service.QCanvasKeyPair, error) {
	m.createPairCalls++
	videoGroupID, mediaGroupID := req.VideoGroupID, req.MediaGroupID
	m.nextID++
	video := &service.APIKey{ID: m.nextID, UserID: userID, Key: "sk-video-one-time-secret", Name: "QCanvas · video", GroupID: &videoGroupID, Status: service.StatusActive}
	m.nextID++
	media := &service.APIKey{ID: m.nextID, UserID: userID, Key: "sk-media-one-time-secret", Name: "QCanvas · media", GroupID: &mediaGroupID, Status: service.StatusActive}
	m.keys[video.ID], m.keys[media.ID] = video, media
	return &service.QCanvasKeyPair{Video: video, Media: media}, nil
}

func (m *lifecycleAPIKeyManagerStub) GetByID(_ context.Context, id int64) (*service.APIKey, error) {
	key, ok := m.keys[id]
	if !ok {
		return nil, service.ErrAPIKeyNotFound
	}
	clone := *key
	clone.IPWhitelist = append([]string(nil), key.IPWhitelist...)
	clone.IPBlacklist = append([]string(nil), key.IPBlacklist...)
	return &clone, nil
}

func (m *lifecycleAPIKeyManagerStub) Update(_ context.Context, id, userID int64, req service.UpdateAPIKeyRequest) (*service.APIKey, error) {
	m.updateCalls++
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	key, ok := m.keys[id]
	if !ok {
		return nil, service.ErrAPIKeyNotFound
	}
	if key.UserID != userID {
		return nil, service.ErrInsufficientPerms
	}
	m.lastUpdateReq = req
	if req.Name != nil {
		key.Name = *req.Name
	}
	if req.Status != nil {
		key.Status = *req.Status
	}
	if req.Quota != nil {
		key.Quota = *req.Quota
	}
	if req.ResetQuota != nil && *req.ResetQuota {
		key.QuotaUsed = 0
	}
	if req.ClearExpiration {
		key.ExpiresAt = nil
	} else if req.ExpiresAt != nil {
		key.ExpiresAt = req.ExpiresAt
	}
	if req.RateLimit5h != nil {
		key.RateLimit5h = *req.RateLimit5h
	}
	if req.RateLimit1d != nil {
		key.RateLimit1d = *req.RateLimit1d
	}
	if req.RateLimit7d != nil {
		key.RateLimit7d = *req.RateLimit7d
	}
	key.IPWhitelist = append([]string(nil), req.IPWhitelist...)
	key.IPBlacklist = append([]string(nil), req.IPBlacklist...)
	clone := *key
	return &clone, nil
}

func (m *lifecycleAPIKeyManagerStub) Delete(_ context.Context, id, userID int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	key, ok := m.keys[id]
	if !ok {
		return service.ErrAPIKeyNotFound
	}
	if key.UserID != userID {
		return service.ErrInsufficientPerms
	}
	m.deletedID, m.deletedOwnerID = id, userID
	delete(m.keys, id)
	return nil
}

func newLifecycleAPIKeyRouter(manager apiKeyManager) *gin.Engine {
	return newLifecycleAPIKeyRouterWithAdmin(newStubAdminService(), manager)
}

func newLifecycleAPIKeyRouterWithAdmin(adminService service.AdminService, manager apiKeyManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &AdminAPIKeyHandler{adminService: adminService, apiKeys: manager}
	router := gin.New()
	router.POST("/api/v1/admin/users/:id/api-keys", h.CreateForUser)
	router.PUT("/api/v1/admin/api-keys/:id", h.UpdateGroup)
	router.DELETE("/api/v1/admin/api-keys/:id", h.Delete)
	return router
}

type countingAPIKeyMutationAdminService struct {
	*stubAdminService
	groupMutationCalls int
	resetMutationCalls int
}

func (s *countingAPIKeyMutationAdminService) AdminUpdateAPIKeyGroupID(ctx context.Context, keyID int64, groupID *int64) (*service.AdminUpdateAPIKeyGroupIDResult, error) {
	s.groupMutationCalls++
	return s.stubAdminService.AdminUpdateAPIKeyGroupID(ctx, keyID, groupID)
}

func (s *countingAPIKeyMutationAdminService) AdminResetAPIKeyRateLimitUsage(ctx context.Context, keyID int64) (*service.APIKey, error) {
	s.resetMutationCalls++
	return s.stubAdminService.AdminResetAPIKeyRateLimitUsage(ctx, keyID)
}

func performAPIKeyLifecycleRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAdminAPIKeyCreateReturnsOnlyNewSecret(t *testing.T) {
	manager := newLifecycleAPIKeyManager(&service.APIKey{ID: 7, UserID: 42, Key: "sk-existing-must-not-leak", Status: service.StatusActive})
	router := newLifecycleAPIKeyRouter(manager)
	recorder := performAPIKeyLifecycleRequest(router, http.MethodPost, "/api/v1/admin/users/42/api-keys", `{"name":"员工卡","quota":10,"rate_limit_5h":2}`)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(42), manager.lastCreateUserID)
	require.Equal(t, "员工卡", manager.lastCreateReq.Name)
	require.Equal(t, 10.0, manager.lastCreateReq.Quota)
	require.Equal(t, 2.0, manager.lastCreateReq.RateLimit5h)
	require.Contains(t, recorder.Body.String(), "sk-new-one-time-secret")
	require.NotContains(t, recorder.Body.String(), "sk-existing-must-not-leak")
}

func TestAdminAPIKeyCreateIdempotencyStoresAndReplaysSanitizedResponse(t *testing.T) {
	repo := newMemoryIdempotencyRepoStub()
	cfg := service.DefaultIdempotencyConfig()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repo, cfg))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })

	manager := newLifecycleAPIKeyManager()
	router := newLifecycleAPIKeyRouter(manager)
	call := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/42/api-keys", bytes.NewBufferString(`{"name":"员工卡"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "create-key-once")
		router.ServeHTTP(recorder, request)
		return recorder
	}

	first := call()
	require.Equal(t, http.StatusOK, first.Code)
	var firstBody struct {
		Data struct {
			Key string `json:"key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstBody))
	require.Equal(t, "sk-new-one-time-secret", firstBody.Data.Key)
	require.Empty(t, first.Header().Get("X-Idempotency-Replayed"))

	stored, err := repo.GetByScopeAndKeyHash(context.Background(), "admin.users.api_keys.create", service.HashIdempotencyKey("create-key-once"))
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.NotNil(t, stored.ResponseBody)
	var storedBody struct {
		Key string `json:"key"`
	}
	require.NoError(t, json.Unmarshal([]byte(*stored.ResponseBody), &storedBody))
	require.Empty(t, storedBody.Key)

	replay := call()
	require.Equal(t, http.StatusOK, replay.Code)
	require.Equal(t, "true", replay.Header().Get("X-Idempotency-Replayed"))
	var replayBody struct {
		Data struct {
			Key string `json:"key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(replay.Body.Bytes(), &replayBody))
	require.Empty(t, replayBody.Data.Key)
	require.Equal(t, 1, manager.createCalls)
}

func TestAdminQCanvasKeyPairCreateIdempotencyReturnsTwoSecretsOnlyOnce(t *testing.T) {
	repo := newMemoryIdempotencyRepoStub()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repo, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })

	manager := newLifecycleAPIKeyManager()
	h := &AdminAPIKeyHandler{adminService: newStubAdminService(), apiKeys: manager}
	router := gin.New()
	router.POST("/api/v1/admin/users/:id/qcanvas-key-pair", h.CreateQCanvasKeyPair)

	call := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/42/qcanvas-key-pair", bytes.NewBufferString(`{"video_group_id":11,"media_group_id":22}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "qcanvas-pair-once")
		router.ServeHTTP(recorder, request)
		return recorder
	}

	first := call()
	require.Equal(t, http.StatusOK, first.Code)
	var firstBody struct {
		Data struct {
			Video struct {
				Key string `json:"key"`
			} `json:"video"`
			Media struct {
				Key string `json:"key"`
			} `json:"media"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstBody))
	require.NotEmpty(t, firstBody.Data.Video.Key)
	require.NotEmpty(t, firstBody.Data.Media.Key)
	require.NotEqual(t, firstBody.Data.Video.Key, firstBody.Data.Media.Key)

	replay := call()
	require.Equal(t, http.StatusOK, replay.Code)
	require.Equal(t, "true", replay.Header().Get("X-Idempotency-Replayed"))
	var replayBody struct {
		Data struct {
			Video struct {
				Key string `json:"key"`
			} `json:"video"`
			Media struct {
				Key string `json:"key"`
			} `json:"media"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(replay.Body.Bytes(), &replayBody))
	require.Empty(t, replayBody.Data.Video.Key)
	require.Empty(t, replayBody.Data.Media.Key)
}

func TestAdminQCanvasKeyPairAllowsEqualGroups(t *testing.T) {
	repo := newMemoryIdempotencyRepoStub()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repo, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })

	manager := newLifecycleAPIKeyManager()
	h := &AdminAPIKeyHandler{adminService: newStubAdminService(), apiKeys: manager}
	router := gin.New()
	router.POST("/api/v1/admin/users/:id/qcanvas-key-pair", h.CreateQCanvasKeyPair)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/42/qcanvas-key-pair", bytes.NewBufferString(`{"video_group_id":11,"media_group_id":11}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "qcanvas-pair-same-group")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, manager.createPairCalls)
	var body struct {
		Data struct {
			Video struct {
				Key     string `json:"key"`
				GroupID int64  `json:"group_id"`
			} `json:"video"`
			Media struct {
				Key     string `json:"key"`
				GroupID int64  `json:"group_id"`
			} `json:"media"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.NotEmpty(t, body.Data.Video.Key)
	require.NotEmpty(t, body.Data.Media.Key)
	require.Equal(t, int64(11), body.Data.Video.GroupID)
	require.Equal(t, int64(11), body.Data.Media.GroupID)
}

func TestAdminAPIKeyListDoesNotExposeExistingSecret(t *testing.T) {
	router, _ := setupAdminRouter()
	for _, path := range []string{"/api/v1/admin/users/1/api-keys", "/api/v1/admin/groups/2/api-keys"} {
		recorder := performAPIKeyLifecycleRequest(router, http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, recorder.Code, path)
		require.NotContains(t, recorder.Body.String(), "sk-test", path)
	}
}

func TestAdminAPIKeyCreateValidationAndNotFound(t *testing.T) {
	invalidBodies := []string{
		`{}`,
		`{"name":"   "}`,
		`{"name":"k","quota":-1}`,
		`{"name":"k","rate_limit_1d":1e309}`,
		`{"name":"k","expires_in_days":0}`,
		`{"name":"k"} {}`,
	}
	for _, body := range invalidBodies {
		manager := newLifecycleAPIKeyManager()
		recorder := performAPIKeyLifecycleRequest(newLifecycleAPIKeyRouter(manager), http.MethodPost, "/api/v1/admin/users/42/api-keys", body)
		require.Equal(t, http.StatusBadRequest, recorder.Code, body)
		require.Zero(t, manager.lastCreateUserID, body)
	}

	recorder := performAPIKeyLifecycleRequest(newLifecycleAPIKeyRouter(newLifecycleAPIKeyManager()), http.MethodPost, "/api/v1/admin/users/0/api-keys", `{"name":"k"}`)
	require.Equal(t, http.StatusBadRequest, recorder.Code)

	manager := newLifecycleAPIKeyManager()
	manager.createErr = infraerrors.NotFound("USER_NOT_FOUND", "user not found")
	recorder = performAPIKeyLifecycleRequest(newLifecycleAPIKeyRouter(manager), http.MethodPost, "/api/v1/admin/users/999/api-keys", `{"name":"k"}`)
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "sk-")
}

func TestAdminAPIKeyUpdateLifecyclePreservesListsAndRedactsSecret(t *testing.T) {
	manager := newLifecycleAPIKeyManager(&service.APIKey{
		ID: 10, UserID: 42, Key: "sk-existing-secret", Name: "old", Status: service.StatusActive,
		IPWhitelist: []string{"10.0.0.1"}, IPBlacklist: []string{"192.0.2.0/24"}, QuotaUsed: 4,
	})
	router := newLifecycleAPIKeyRouter(manager)
	recorder := performAPIKeyLifecycleRequest(router, http.MethodPut, "/api/v1/admin/api-keys/10", `{"name":"new","status":"disabled","quota":20,"reset_quota":true,"rate_limit_7d":5}`)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "new", manager.keys[10].Name)
	require.Equal(t, service.StatusAPIKeyDisabled, manager.keys[10].Status)
	require.Equal(t, []string{"10.0.0.1"}, manager.keys[10].IPWhitelist)
	require.Equal(t, []string{"192.0.2.0/24"}, manager.keys[10].IPBlacklist)
	require.Zero(t, manager.keys[10].QuotaUsed)
	require.NotContains(t, recorder.Body.String(), "sk-existing-secret")

	var body struct {
		Data struct {
			APIKey struct {
				Key string `json:"key"`
			} `json:"api_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Empty(t, body.Data.APIKey.Key)
}

func TestAdminAPIKeyUpdateValidationAndNotFound(t *testing.T) {
	invalidBodies := []string{
		`{"status":"bogus"}`,
		`{"quota":-1}`,
		`{"rate_limit_5h":-1}`,
		`{"expires_at":"tomorrow"}`,
		`{"name":"x"} {}`,
	}
	for _, body := range invalidBodies {
		manager := newLifecycleAPIKeyManager(&service.APIKey{ID: 10, UserID: 42, Key: "sk-secret", Status: service.StatusActive})
		recorder := performAPIKeyLifecycleRequest(newLifecycleAPIKeyRouter(manager), http.MethodPut, "/api/v1/admin/api-keys/10", body)
		require.Equal(t, http.StatusBadRequest, recorder.Code, body)
		require.NotContains(t, recorder.Body.String(), "sk-secret", body)
	}

	recorder := performAPIKeyLifecycleRequest(newLifecycleAPIKeyRouter(newLifecycleAPIKeyManager()), http.MethodPut, "/api/v1/admin/api-keys/999", `{"name":"x"}`)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAdminAPIKeyUpdateRejectsMixedOrEmptyMutationBeforeSideEffects(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "field and group", body: `{"name":"new","group_id":2}`},
		{name: "field and rate reset", body: `{"name":"new","reset_rate_limit_usage":true}`},
		{name: "group and rate reset", body: `{"group_id":2,"reset_rate_limit_usage":true}`},
		{name: "all categories", body: `{"name":"new","group_id":2,"reset_rate_limit_usage":true}`},
		{name: "empty", body: `{}`},
		{name: "negative group preflight", body: `{"group_id":-1}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := newLifecycleAPIKeyManager(&service.APIKey{ID: 10, UserID: 1, Key: "sk-secret", Status: service.StatusActive})
			adminService := &countingAPIKeyMutationAdminService{stubAdminService: newStubAdminService()}
			router := newLifecycleAPIKeyRouterWithAdmin(adminService, manager)
			recorder := performAPIKeyLifecycleRequest(router, http.MethodPut, "/api/v1/admin/api-keys/10", test.body)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Zero(t, manager.updateCalls, "APIKeyService.Update must not run")
			require.Zero(t, adminService.groupMutationCalls, "group mutation must not run")
			require.Zero(t, adminService.resetMutationCalls, "rate-limit reset must not run")
		})
	}
}

func TestAdminAPIKeyDeleteResolvesOwnerAndHandlesNotFound(t *testing.T) {
	manager := newLifecycleAPIKeyManager(&service.APIKey{ID: 10, UserID: 42, Key: "sk-delete-secret", Status: service.StatusActive})
	router := newLifecycleAPIKeyRouter(manager)
	recorder := performAPIKeyLifecycleRequest(router, http.MethodDelete, "/api/v1/admin/api-keys/10", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(10), manager.deletedID)
	require.Equal(t, int64(42), manager.deletedOwnerID)
	require.NotContains(t, recorder.Body.String(), "sk-delete-secret")

	recorder = performAPIKeyLifecycleRequest(router, http.MethodDelete, "/api/v1/admin/api-keys/10", "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "sk-delete-secret")
}
