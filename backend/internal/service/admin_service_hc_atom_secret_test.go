//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const hcAtomSecretSentinel = "hc-sentinel-DO-NOT-PERSIST-7x9Q"

type hcAtomSecretBoundaryRepo struct {
	mockAccountRepoForGemini
	account *Account
}

type hcAtomSecretCaptureClient struct {
	apiKey string
}

func (c *hcAtomSecretCaptureClient) Create(_ context.Context, apiKey, _ string, _ HCAtomBatchCreateRequest) (*HCAtomBatchTask, error) {
	c.apiKey = apiKey
	return &HCAtomBatchTask{TaskID: "hc-secret-task", Status: "PENDING"}, nil
}

func (c *hcAtomSecretCaptureClient) Get(_ context.Context, apiKey, _ string) (*HCAtomBatchTask, error) {
	c.apiKey = apiKey
	return &HCAtomBatchTask{TaskID: "hc-secret-task", Status: "RUNNING"}, nil
}

func (c *hcAtomSecretCaptureClient) Delete(_ context.Context, apiKey, _ string) error {
	c.apiKey = apiKey
	return nil
}

func (r *hcAtomSecretBoundaryRepo) Create(_ context.Context, account *Account) error {
	account.ID = 901
	r.account = account
	return nil
}

func (r *hcAtomSecretBoundaryRepo) GetByID(_ context.Context, _ int64) (*Account, error) {
	return r.account, nil
}

func (r *hcAtomSecretBoundaryRepo) Update(_ context.Context, account *Account) error {
	r.account = account
	return nil
}

func TestHCAtomSecretBoundary_CreateEncryptsSentinelBeforeRepository(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("11", 32))
	require.NoError(t, err)
	repo := &hcAtomSecretBoundaryRepo{}
	svc := &adminServiceImpl{
		accountRepo:            repo,
		hcAtomCredentialCipher: cipher,
	}

	created, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "hc-secret-boundary",
		Platform:             PlatformHCAtom,
		Type:                 AccountTypeAPIKey,
		Credentials:          map[string]any{"api_key": hcAtomSecretSentinel},
		SkipDefaultGroupBind: true,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotNil(t, repo.account)

	persisted, err := json.Marshal(repo.account.Credentials)
	require.NoError(t, err)
	require.NotContains(t, string(persisted), hcAtomSecretSentinel)
	require.NotContains(t, repo.account.Credentials, "api_key")
	require.NotContains(t, repo.account.Credentials, "hc_atom_api_key")
	require.NotEmpty(t, repo.account.Credentials[HCAtomAPIKeyCiphertextField])
	require.Equal(t, "********7x9Q", repo.account.Credentials[HCAtomAPIKeyMaskedField])
	require.Equal(t, true, repo.account.Credentials[HCAtomAPIKeyConfiguredField])
}

func TestHCAtomSecretBoundary_UpdateWithoutNewKeyRetainsCiphertext(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("12", 32))
	require.NoError(t, err)
	existing, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": hcAtomSecretSentinel}, cipher)
	require.NoError(t, err)
	oldCiphertext := existing[HCAtomAPIKeyCiphertextField]
	repo := &hcAtomSecretBoundaryRepo{account: &Account{
		ID: 902, Platform: PlatformHCAtom, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: existing,
	}}
	svc := &adminServiceImpl{accountRepo: repo, hcAtomCredentialCipher: cipher}

	updated, err := svc.UpdateAccount(context.Background(), 902, &UpdateAccountInput{
		Credentials: map[string]any{"model_mapping": map[string]any{"seedream-5.0": "seedream-5.0"}},
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, oldCiphertext, repo.account.Credentials[HCAtomAPIKeyCiphertextField])
	require.NotContains(t, repo.account.Credentials, "api_key")
	raw, err := json.Marshal(repo.account.Credentials)
	require.NoError(t, err)
	require.NotContains(t, string(raw), hcAtomSecretSentinel)
}

func TestProtectHCAtomAccountCredentials_EncryptsTransientAPIKeyAndResolvesOnDemand(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("22", 32))
	require.NoError(t, err)

	protected, err := ProtectHCAtomAccountCredentials(nil, map[string]any{
		"api_key":       hcAtomSecretSentinel,
		"model_mapping": map[string]any{"seedream-5.0": "seedream-5.0"},
	}, cipher)
	require.NoError(t, err)
	require.NotContains(t, protected, "api_key")
	require.NotContains(t, protected, "hc_atom_api_key")
	require.NotEqual(t, hcAtomSecretSentinel, protected[HCAtomAPIKeyCiphertextField])
	require.Equal(t, "********7x9Q", protected[HCAtomAPIKeyMaskedField])
	require.Equal(t, true, protected[HCAtomAPIKeyConfiguredField])

	account := &Account{Platform: PlatformHCAtom, Type: AccountTypeAPIKey, Credentials: protected}
	resolved, err := ResolveHCAtomAPIKey(account, cipher)
	require.NoError(t, err)
	require.Equal(t, hcAtomSecretSentinel, resolved)
}

func TestProtectHCAtomAccountCredentials_UpdateWithoutNewKeyRetainsCiphertext(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("33", 32))
	require.NoError(t, err)
	existing, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": hcAtomSecretSentinel}, cipher)
	require.NoError(t, err)
	oldCiphertext := existing[HCAtomAPIKeyCiphertextField]

	protected, err := ProtectHCAtomAccountCredentials(existing, map[string]any{
		"model_mapping": map[string]any{"seedream-5.0": "seedream-5.0"},
	}, cipher)
	require.NoError(t, err)
	require.Equal(t, oldCiphertext, protected[HCAtomAPIKeyCiphertextField])
	require.Equal(t, true, protected[HCAtomAPIKeyConfiguredField])
	require.NotContains(t, protected, "api_key")
}

func TestResolveHCAtomAPIKey_WrongKeyFailsWithoutSecretOrCiphertext(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("44", 32))
	require.NoError(t, err)
	protected, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": hcAtomSecretSentinel}, cipher)
	require.NoError(t, err)
	ciphertext := protected[HCAtomAPIKeyCiphertextField].(string)
	wrongCipher, err := NewHCAtomCredentialCipher(strings.Repeat("55", 32))
	require.NoError(t, err)

	_, err = ResolveHCAtomAPIKey(&Account{
		Platform:    PlatformHCAtom,
		Type:        AccountTypeAPIKey,
		Credentials: protected,
	}, wrongCipher)
	require.Error(t, err)
	require.NotContains(t, err.Error(), hcAtomSecretSentinel)
	require.NotContains(t, err.Error(), ciphertext)
}

func TestHCAtomBatchProvider_DecryptsOnlyAfterDedicatedAccountSelection(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("66", 32))
	require.NoError(t, err)
	protected, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": hcAtomSecretSentinel}, cipher)
	require.NoError(t, err)
	account := &Account{Platform: PlatformHCAtom, Type: AccountTypeAPIKey, Credentials: protected}
	client := &hcAtomSecretCaptureClient{}
	provider := NewHCAtomBatchImageProviderWithCredentialCipher(client, cipher)

	require.True(t, provider.SupportsAccount(account))
	require.Empty(t, client.apiKey, "account selection must not decrypt the credential")
	_, err = provider.Submit(context.Background(), &BatchImageJob{BatchID: "imgbatch_secret", Model: "seedream-5.0"}, account, BatchImageInput{
		BatchID: "imgbatch_secret",
		Model:   "seedream-5.0",
		Items:   []BatchImageInputItem{{CustomID: "item_1", Prompt: "safe synthetic prompt"}},
	})
	require.NoError(t, err)
	require.Equal(t, hcAtomSecretSentinel, client.apiKey)
}
