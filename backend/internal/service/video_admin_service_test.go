package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeVideoAdminRepo struct {
	created VideoProviderAccount
	updated VideoProviderAdminUpdate
}

func (f *fakeVideoAdminRepo) ListVideoProviders(context.Context) ([]VideoProviderAccount, error) {
	return nil, nil
}
func (f *fakeVideoAdminRepo) CreateVideoProvider(_ context.Context, provider VideoProviderAccount) (*VideoProviderAccount, error) {
	f.created = provider
	return &provider, nil
}
func (f *fakeVideoAdminRepo) UpdateVideoProvider(_ context.Context, _ int64, in VideoProviderAdminUpdate) (*VideoProviderAccount, error) {
	f.updated = in
	return &VideoProviderAccount{ID: 1, GroupID: 9, Provider: "seedance", DisplayName: "Seedance", BaseURL: derefString(in.BaseURL), DefaultModel: derefString(in.DefaultModel)}, nil
}
func (f *fakeVideoAdminRepo) AuthorizeTinyReal(context.Context, int64, int64) (*VideoProviderAccount, error) {
	return nil, nil
}
func (f *fakeVideoAdminRepo) ListVideoTasks(context.Context, VideoAdminTaskFilter) ([]VideoTask, int64, error) {
	return nil, 0, nil
}
func (f *fakeVideoAdminRepo) GetVideoTaskAdmin(context.Context, int64) (*VideoTask, error) {
	return nil, nil
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

func TestVideoAdminServiceRejectsNonSeedanceProvider(t *testing.T) {
	svc := NewVideoAdminService(&fakeVideoAdminRepo{}, fakeVideoEncryptor{})
	_, err := svc.CreateProvider(context.Background(), VideoProviderAdminCreate{GroupID: 1, Provider: "mock", DisplayName: "mock", APIKey: "x"})
	require.ErrorContains(t, err, "seedance")
}

func TestVideoAdminServiceForcesCanonicalEndpointAndModel(t *testing.T) {
	repo := &fakeVideoAdminRepo{}
	svc := NewVideoAdminService(repo, fakeVideoEncryptor{})
	_, err := svc.CreateProvider(context.Background(), VideoProviderAdminCreate{
		GroupID: 9, Provider: "seedance", DisplayName: "Seedance", APIKey: "secret", BaseURL: "https://evil.test", DefaultModel: "wrong-model",
	})
	require.NoError(t, err)
	require.Equal(t, SeedanceBaseURL, repo.created.BaseURL)
	require.Equal(t, SeedanceModel, repo.created.DefaultModel)

	_, err = svc.UpdateProvider(context.Background(), 1, VideoProviderAdminUpdate{BaseURL: stringPointer("https://evil.test"), DefaultModel: stringPointer("wrong-model")})
	require.NoError(t, err)
	require.Equal(t, SeedanceBaseURL, derefString(repo.updated.BaseURL))
	require.Equal(t, SeedanceModel, derefString(repo.updated.DefaultModel))
}

func stringPointer(value string) *string { return &value }
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
