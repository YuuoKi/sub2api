package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeVideoAdminRepo struct {
	created   VideoProviderAccount
	updated   VideoProviderAdminUpdate
	task      *VideoTask
	deletedID int64
	deleteErr error
	providers []VideoProviderAccount
}

func (f *fakeVideoAdminRepo) ListVideoProviders(context.Context) ([]VideoProviderAccount, error) {
	if f.providers != nil {
		return f.providers, nil
	}
	return []VideoProviderAccount{{ID: 1, Provider: "seedance"}}, nil
}
func (f *fakeVideoAdminRepo) CreateVideoProvider(_ context.Context, provider VideoProviderAccount) (*VideoProviderAccount, error) {
	f.created = provider
	return &provider, nil
}
func (f *fakeVideoAdminRepo) UpdateVideoProvider(_ context.Context, _ int64, in VideoProviderAdminUpdate) (*VideoProviderAccount, error) {
	f.updated = in
	return &VideoProviderAccount{ID: 1, GroupID: 9, Provider: "seedance", DisplayName: "Seedance", BaseURL: derefString(in.BaseURL), DefaultModel: derefString(in.DefaultModel)}, nil
}
func (f *fakeVideoAdminRepo) DeleteVideoProvider(_ context.Context, id int64) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedID = id
	return nil
}
func (f *fakeVideoAdminRepo) AuthorizeTinyReal(context.Context, int64, int64) (*VideoProviderAccount, error) {
	return nil, nil
}
func (f *fakeVideoAdminRepo) ListVideoTasks(context.Context, VideoAdminTaskFilter) ([]VideoTask, int64, error) {
	return nil, 0, nil
}
func (f *fakeVideoAdminRepo) GetVideoTaskAdmin(context.Context, int64) (*VideoTask, error) {
	if f.task == nil {
		return nil, ErrVideoTaskNotFound
	}
	return f.task, nil
}
func (f *fakeVideoAdminRepo) VideoSystemCheck(context.Context) (VideoSystemCheck, error) {
	return VideoSystemCheck{}, nil
}

type fakeVideoEncryptor struct{}

func (fakeVideoEncryptor) Encrypt(value string) (string, error) { return "encrypted:" + value, nil }
func (fakeVideoEncryptor) Decrypt(value string) (string, error) { return value, nil }

func TestVideoAdminServiceCreateMasksAndEncryptsSecret(t *testing.T) {
	repo := &fakeVideoAdminRepo{}
	svc := NewVideoAdminService(repo, fakeVideoEncryptor{})
	item, err := svc.CreateProvider(context.Background(), VideoProviderAdminCreate{
		GroupID: 9, Provider: "seedance", DisplayName: "Seedance 2.0", APIKey: "secret-value", Enabled: true,
	})
	require.NoError(t, err)
	require.Equal(t, "encrypted:secret-value", repo.created.EncryptedAPIKey)
	require.Equal(t, "secr…alue", item.MaskedKey)
	require.Empty(t, item.EncryptedAPIKey)
}

func TestLookupVideoProvider(t *testing.T) {
	spec, ok := lookupVideoProvider(" Seedance ")
	require.True(t, ok)
	require.Equal(t, "seedance", spec.Provider)
	require.Equal(t, SeedanceBaseURL, spec.DefaultBaseURL)
	require.Equal(t, SeedanceModel, spec.DefaultModel)
	require.True(t, spec.AdapterReady)

	_, ok = lookupVideoProvider("unknown")
	require.False(t, ok)

	registry := VideoProviderRegistry()
	require.Len(t, registry, 6)
	for _, provider := range []string{"seedance", HCAtomVideoV1Provider, HCAtomSeedanceV3Provider, "jimeng", "veo", "kling"} {
		_, found := lookupVideoProvider(provider)
		require.True(t, found, provider)
	}
}

func TestVideoAdminServiceRejectsUnsupportedProvider(t *testing.T) {
	svc := NewVideoAdminService(&fakeVideoAdminRepo{}, fakeVideoEncryptor{})
	_, err := svc.CreateProvider(context.Background(), VideoProviderAdminCreate{GroupID: 1, Provider: "mock", DisplayName: "mock", APIKey: "x"})
	require.ErrorIs(t, err, ErrVideoAdminInvalidRequest)
	require.ErrorContains(t, err, "不支持的视频平台")
}

func TestVideoAdminServiceRejectsNotReadyProvider(t *testing.T) {
	repo := &fakeVideoAdminRepo{}
	svc := NewVideoAdminService(repo, fakeVideoEncryptor{})
	for _, provider := range []string{"jimeng", "veo", "kling"} {
		_, err := svc.CreateProvider(context.Background(), VideoProviderAdminCreate{GroupID: 1, Provider: provider, DisplayName: provider, APIKey: "x"})
		require.ErrorIs(t, err, ErrVideoAdminInvalidRequest, provider)
		require.ErrorContains(t, err, "该平台即将接入，暂不能创建通道", provider)
	}
	require.Empty(t, repo.created.Provider)
}

func TestVideoAdminServiceRejectsCustomEndpointUntilRelayIsWired(t *testing.T) {
	repo := &fakeVideoAdminRepo{}
	svc := NewVideoAdminService(repo, fakeVideoEncryptor{})
	_, err := svc.CreateProvider(context.Background(), VideoProviderAdminCreate{
		GroupID: 9, Provider: "seedance", DisplayName: "Seedance 中转", APIKey: "secret", BaseURL: "https://relay.test/v1", DefaultModel: "relay-model",
	})
	require.ErrorIs(t, err, ErrVideoAdminInvalidRequest)
	require.ErrorContains(t, err, "自定义接口地址尚未打通")
	require.Empty(t, repo.created.Provider)

	_, err = svc.UpdateProvider(context.Background(), 1, VideoProviderAdminUpdate{BaseURL: stringPointer("https://relay.test/v1")})
	require.ErrorIs(t, err, ErrVideoAdminInvalidRequest)
	require.ErrorContains(t, err, "自定义接口地址尚未打通")
}

func TestVideoAdminServiceFallsBackToRegistryDefaults(t *testing.T) {
	repo := &fakeVideoAdminRepo{}
	svc := NewVideoAdminService(repo, fakeVideoEncryptor{})
	_, err := svc.CreateProvider(context.Background(), VideoProviderAdminCreate{
		GroupID: 9, Provider: "seedance", DisplayName: "Seedance", APIKey: "secret",
	})
	require.NoError(t, err)
	require.Equal(t, SeedanceBaseURL, repo.created.BaseURL)
	require.Equal(t, SeedanceModel, repo.created.DefaultModel)

	// 显式提供空值时回落注册表默认
	_, err = svc.UpdateProvider(context.Background(), 1, VideoProviderAdminUpdate{BaseURL: stringPointer("  "), DefaultModel: stringPointer("")})
	require.NoError(t, err)
	require.Equal(t, SeedanceBaseURL, derefString(repo.updated.BaseURL))
	require.Equal(t, SeedanceModel, derefString(repo.updated.DefaultModel))
}

func TestVideoAdminServicePartialUpdateKeepsCustomEndpointAndModel(t *testing.T) {
	repo := &fakeVideoAdminRepo{}
	svc := NewVideoAdminService(repo, fakeVideoEncryptor{})
	enabled := false
	_, err := svc.UpdateProvider(context.Background(), 1, VideoProviderAdminUpdate{Enabled: &enabled})
	require.NoError(t, err)
	// 字段缺席必须保持 nil，由 repo 层 CASE WHEN 保留原值，不能强制重置自定义中转配置
	require.Nil(t, repo.updated.BaseURL)
	require.Nil(t, repo.updated.DefaultModel)
}

func TestVideoAdminServiceUpdateUsesHCProviderDefaults(t *testing.T) {
	repo := &fakeVideoAdminRepo{providers: []VideoProviderAccount{{ID: 1, Provider: HCAtomSeedanceV3Provider}}}
	svc := NewVideoAdminService(repo, fakeVideoEncryptor{})
	empty := ""
	_, err := svc.UpdateProvider(context.Background(), 1, VideoProviderAdminUpdate{BaseURL: &empty, DefaultModel: &empty})
	require.NoError(t, err)
	require.Equal(t, HCAtomSeedanceV3BaseURL, derefString(repo.updated.BaseURL))
	require.Equal(t, HCAtomSeedanceV3PublicModel, derefString(repo.updated.DefaultModel))
}

func TestVideoAdminServiceDeleteProvider(t *testing.T) {
	repo := &fakeVideoAdminRepo{}
	svc := NewVideoAdminService(repo, fakeVideoEncryptor{})

	require.NoError(t, svc.DeleteProvider(context.Background(), 5))
	require.Equal(t, int64(5), repo.deletedID)

	repo.deleteErr = ErrVideoProviderNotFound
	err := svc.DeleteProvider(context.Background(), 99)
	require.ErrorIs(t, err, ErrVideoProviderNotFound)

	repo.deleteErr = nil
	repo.deletedID = 0
	err = svc.DeleteProvider(context.Background(), 0)
	require.ErrorIs(t, err, ErrVideoProviderNotFound)
	require.Zero(t, repo.deletedID)
}

func TestVideoAdminServiceRejectsInvalidTinyRealAuthorizationIdentity(t *testing.T) {
	svc := NewVideoAdminService(&fakeVideoAdminRepo{}, fakeVideoEncryptor{})
	_, err := svc.AuthorizeTinyReal(context.Background(), 0, 7)
	require.ErrorIs(t, err, ErrVideoAdminInvalidRequest)
	_, err = svc.AuthorizeTinyReal(context.Background(), 7, 0)
	require.ErrorIs(t, err, ErrVideoAdminInvalidRequest)
}

func stringPointer(value string) *string { return &value }
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
