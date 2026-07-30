//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/buildinfo"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type buildIdentitySettingRepoStub struct{}

func (buildIdentitySettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (buildIdentitySettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	return "", service.ErrSettingNotFound
}
func (buildIdentitySettingRepoStub) Set(ctx context.Context, key, value string) error { return nil }
func (buildIdentitySettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (buildIdentitySettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	return nil
}
func (buildIdentitySettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (buildIdentitySettingRepoStub) Delete(ctx context.Context, key string) error { return nil }

func TestFourVersionIdentitySourcesConsistent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	info := buildinfo.New("0.1.151", "deadbeefcafebabe0123456789abcdef01234567", "2026-07-25T08:30:00Z", "source")

	cli := info.CLIString()
	require.Contains(t, cli, info.Version)
	require.Contains(t, cli, info.BuildCommit)
	require.Contains(t, cli, info.BuildDate)
	require.NotContains(t, cli, "v"+info.BuildCommit)

	settingSvc := service.NewSettingService(buildIdentitySettingRepoStub{}, &config.Config{DeploymentProfile: config.DeploymentProfileLANAdmin})
	settingSvc.SetBuildInfo(info)
	injectionAny, err := settingSvc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	injection := injectionAny.(*service.PublicSettingsInjectionPayload)
	require.Equal(t, info.Version, injection.Version)
	require.Equal(t, info.BuildCommit, injection.BuildCommit)
	require.Equal(t, info.BuildDate, injection.BuildDate)

	publicHandler := NewSettingHandler(settingSvc, info)
	pubRec := httptest.NewRecorder()
	pubCtx, _ := gin.CreateTestContext(pubRec)
	pubCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/settings/public", nil)
	publicHandler.GetPublicSettings(pubCtx)
	require.Equal(t, http.StatusOK, pubRec.Code)
	var pubResp struct {
		Data dto.PublicSettings `json:"data"`
	}
	require.NoError(t, json.Unmarshal(pubRec.Body.Bytes(), &pubResp))
	require.Equal(t, info.Version, pubResp.Data.Version)
	require.Equal(t, info.BuildCommit, pubResp.Data.BuildCommit)
	require.Equal(t, info.BuildDate, pubResp.Data.BuildDate)

	sysHandler := admin.NewSystemHandler(info, nil, nil)
	sysRec := httptest.NewRecorder()
	sysCtx, _ := gin.CreateTestContext(sysRec)
	sysCtx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/system/version", nil)
	sysHandler.GetVersion(sysCtx)
	require.Equal(t, http.StatusOK, sysRec.Code)
	var sysResp struct {
		Data map[string]string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(sysRec.Body.Bytes(), &sysResp))
	require.Equal(t, info.Version, sysResp.Data["version"])
	require.Equal(t, info.BuildCommit, sysResp.Data["build_commit"])
	require.Equal(t, info.BuildDate, sysResp.Data["build_date"])
	require.NotContains(t, sysRec.Body.String(), "v"+info.BuildCommit)
}
