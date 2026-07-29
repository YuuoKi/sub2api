//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

const hcAtomSecretSentinel = "hc-sentinel-DO-NOT-PERSIST-7x9Q"

type hcAtomSecretBoundaryRepo struct {
	mockAccountRepoForGemini
	account     *Account
	createCalls int
	updateCalls int
}

type hcAtomSecretCaptureClient struct {
	apiKey string
}

type hcAtomGroupValidationRepo struct {
	groupRepoStub
	groups map[int64]*Group
}

func validHCAtomImageGroupForTest(id int64) *Group {
	price1K, price2K, price4K := 0.134, 0.201, 0.268
	return &Group{
		ID:                        id,
		Platform:                  PlatformHCAtom,
		Status:                    StatusActive,
		AllowImageGeneration:      true,
		AllowBatchImageGeneration: true,
		ImagePrice1K:              &price1K,
		ImagePrice2K:              &price2K,
		ImagePrice4K:              &price4K,
		ModelsListConfig: GroupModelsListConfig{
			Enabled: true,
			Models: []string{
				HCAtomImageSeedreamModel,
				HCAtomImageDoubaoSeedreamModel,
				HCAtomImageGeminiModel,
				HCAtomImageGPTModel,
				HCAtomImageSGPTModel,
			},
		},
	}
}

func (r *hcAtomGroupValidationRepo) GetByID(_ context.Context, id int64) (*Group, error) {
	if group := r.groups[id]; group != nil {
		return group, nil
	}
	return nil, ErrGroupNotFound
}

func (c *hcAtomSecretCaptureClient) Create(_ context.Context, apiKey, _ string, _ HCAtomBatchCreateRequest) (*HCAtomBatchTask, error) {
	c.apiKey = apiKey
	return &HCAtomBatchTask{TaskID: "hc-secret-task", Status: "PENDING"}, nil
}

func (c *hcAtomSecretCaptureClient) Get(_ context.Context, apiKey, _ string, _ ...string) (*HCAtomBatchTask, error) {
	c.apiKey = apiKey
	return &HCAtomBatchTask{TaskID: "hc-secret-task", Status: "RUNNING"}, nil
}

func (c *hcAtomSecretCaptureClient) Delete(_ context.Context, apiKey, _ string, _ ...string) error {
	c.apiKey = apiKey
	return nil
}

func (r *hcAtomSecretBoundaryRepo) Create(_ context.Context, account *Account) error {
	r.createCalls++
	account.ID = 901
	r.account = account
	return nil
}

func (r *hcAtomSecretBoundaryRepo) GetByID(_ context.Context, _ int64) (*Account, error) {
	return r.account, nil
}

func (r *hcAtomSecretBoundaryRepo) GetByIDs(_ context.Context, _ []int64) ([]*Account, error) {
	if r.account == nil {
		return nil, nil
	}
	return []*Account{r.account}, nil
}

func (r *hcAtomSecretBoundaryRepo) Update(_ context.Context, account *Account) error {
	r.updateCalls++
	r.account = account
	return nil
}

func TestAdminService_HCAtomAccountAcceptsOnlyAuthorizedImageGroup(t *testing.T) {
	valid := validHCAtomImageGroupForTest(11)
	video := &Group{
		ID:       12,
		Platform: PlatformHCAtom,
		Status:   StatusActive,
		ModelsListConfig: GroupModelsListConfig{
			Enabled: true,
			Models:  []string{HCAtomVideoV1PublicModel},
		},
	}
	svc := &adminServiceImpl{groupRepo: &hcAtomGroupValidationRepo{groups: map[int64]*Group{
		valid.ID: valid,
		video.ID: video,
	}}}

	got, err := svc.validateAccountGroupIDs(context.Background(), PlatformHCAtom, []int64{valid.ID, valid.ID})
	require.NoError(t, err)
	require.Equal(t, []int64{valid.ID}, got)

	_, err = svc.validateAccountGroupIDs(context.Background(), PlatformHCAtom, []int64{video.ID})
	require.Equal(t, "HC_ATOM_IMAGE_GROUP_INVALID", infraerrors.Reason(err))

	withExtra := *valid
	withExtra.ID = 13
	withExtra.ModelsListConfig.Models = append(
		append([]string{}, valid.ModelsListConfig.Models...),
		"dola-seedream-5.0-pro",
	)
	svc.groupRepo = &hcAtomGroupValidationRepo{groups: map[int64]*Group{withExtra.ID: &withExtra}}
	_, err = svc.validateAccountGroupIDs(context.Background(), PlatformHCAtom, []int64{withExtra.ID})
	require.Equal(t, "HC_ATOM_IMAGE_GROUP_INVALID", infraerrors.Reason(err))
}

func TestHCAtomSecretBoundary_CreateEncryptsSentinelBeforeRepository(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("11", 32))
	require.NoError(t, err)
	nestedSentinel := strings.Join([]string{"hc-review", "repo", "secret"}, "-")
	repo := &hcAtomSecretBoundaryRepo{}
	group := validHCAtomImageGroupForTest(11)
	svc := &adminServiceImpl{
		accountRepo:            repo,
		groupRepo:              &hcAtomGroupValidationRepo{groups: map[int64]*Group{group.ID: group}},
		hcAtomCredentialCipher: cipher,
	}

	created, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "hc-secret-boundary",
		Platform: PlatformHCAtom,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":       hcAtomSecretSentinel,
			"protocol":      "hc_atom",
			"Authorization": "Bearer " + nestedSentinel,
			"metadata":      []any{map[string]any{"token": nestedSentinel}},
			"model_mapping": map[string]any{"seedream-5.0": "seedream-5.0"},
		},
		GroupIDs: []int64{group.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.NotNil(t, repo.account)

	persisted, err := json.Marshal(repo.account.Credentials)
	require.NoError(t, err)
	require.NotContains(t, string(persisted), hcAtomSecretSentinel)
	require.NotContains(t, string(persisted), nestedSentinel)
	require.NotContains(t, repo.account.Credentials, "api_key")
	require.NotContains(t, repo.account.Credentials, "hc_atom_api_key")
	require.NotContains(t, repo.account.Credentials, "Authorization")
	require.NotContains(t, repo.account.Credentials, "metadata")
	require.Equal(t, "hc_atom", repo.account.Credentials["protocol"])
	require.Equal(t, map[string]any{"seedream-5.0": "seedream-5.0"}, repo.account.Credentials["model_mapping"])
	require.NotEmpty(t, repo.account.Credentials[HCAtomAPIKeyCiphertextField])
	require.Equal(t, "********7x9Q", repo.account.Credentials[HCAtomAPIKeyMaskedField])
	require.Equal(t, true, repo.account.Credentials[HCAtomAPIKeyConfiguredField])
}

func TestHCAtomSecretBoundary_CreateRequiresAuthorizedImageGroup(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("31", 32))
	require.NoError(t, err)
	repo := &hcAtomSecretBoundaryRepo{}
	svc := &adminServiceImpl{
		accountRepo:            repo,
		hcAtomCredentialCipher: cipher,
	}

	_, err = svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "hc-group-required",
		Platform: PlatformHCAtom,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  hcAtomSecretSentinel,
			"protocol": "hc_atom",
		},
		SkipDefaultGroupBind: true,
	})
	require.Equal(t, "HC_ATOM_IMAGE_GROUP_REQUIRED", infraerrors.Reason(err))
	require.Zero(t, repo.createCalls)
	require.Nil(t, repo.account)
}

func TestHCAtomSecretBoundary_UpdateWithoutNewKeyRetainsCiphertext(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("12", 32))
	require.NoError(t, err)
	existing, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": hcAtomSecretSentinel}, cipher)
	require.NoError(t, err)
	oldCiphertext := existing[HCAtomAPIKeyCiphertextField]
	oldMasked := existing[HCAtomAPIKeyMaskedField]
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
	require.Equal(t, oldMasked, repo.account.Credentials[HCAtomAPIKeyMaskedField])
	require.NotContains(t, repo.account.Credentials, "api_key")
	raw, err := json.Marshal(repo.account.Credentials)
	require.NoError(t, err)
	require.NotContains(t, string(raw), hcAtomSecretSentinel)
}

func TestHCAtomSecretBoundary_BulkCredentialUpdateFailsClosed(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("13", 32))
	require.NoError(t, err)
	existing, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": "existing-synthetic-key"}, cipher)
	require.NoError(t, err)
	repo := &hcAtomSecretBoundaryRepo{account: &Account{
		ID: 903, Platform: PlatformHCAtom, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: existing,
	}}
	svc := &adminServiceImpl{accountRepo: repo, hcAtomCredentialCipher: cipher}

	_, err = svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{903},
		Credentials: map[string]any{
			"api_key": hcAtomSecretSentinel,
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "HC-ATOM")
	require.Equal(t, 0, repo.updateCalls)
	raw, marshalErr := json.Marshal(repo.account.Credentials)
	require.NoError(t, marshalErr)
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

func TestProtectHCAtomAccountCredentials_StrictAllowlistDropsAliasesAndNestedSecrets(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("23", 32))
	require.NoError(t, err)
	nestedSentinel := strings.Join([]string{"hc-review", "nested", "secret"}, "-")

	protected, err := ProtectHCAtomAccountCredentials(nil, map[string]any{
		"api_key":       hcAtomSecretSentinel,
		"protocol":      "hc_atom",
		"Authorization": "Bearer " + nestedSentinel,
		"TOKEN":         nestedSentinel,
		"metadata": map[string]any{
			"token": nestedSentinel,
			"items": []any{map[string]any{"api_key": nestedSentinel}},
		},
		"arbitrary": []any{nestedSentinel},
		"model_mapping": map[string]any{
			"seedream-5.0": "seedream-5.0",
		},
	}, cipher)
	require.NoError(t, err)

	raw, err := json.Marshal(protected)
	require.NoError(t, err)
	require.NotContains(t, string(raw), nestedSentinel)
	require.Equal(t, "hc_atom", protected["protocol"])
	require.Equal(t, map[string]any{"seedream-5.0": "seedream-5.0"}, protected["model_mapping"])
	require.ElementsMatch(t, []string{
		"protocol",
		"model_mapping",
		HCAtomAPIKeyCiphertextField,
		HCAtomAPIKeyMaskedField,
		HCAtomAPIKeyConfiguredField,
	}, mapKeys(protected))
}

func TestProtectHCAtomAccountCredentials_RejectsSecretValuesInProtocolAndModelMapping(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("25", 32))
	require.NoError(t, err)
	sentinel := hcAtomSecretSentinel
	liveLike := "sk-live-abc123"

	tests := []struct {
		name        string
		credentials map[string]any
	}{
		{name: "protocol sentinel", credentials: map[string]any{"api_key": "synthetic", "protocol": sentinel}},
		{name: "protocol live-like", credentials: map[string]any{"api_key": "synthetic", "protocol": liveLike}},
		{name: "protocol near miss", credentials: map[string]any{"api_key": "synthetic", "protocol": "hc-atom"}},
		{name: "mapping key sentinel", credentials: map[string]any{"api_key": "synthetic", "model_mapping": map[string]any{sentinel: "seedream-5.0"}}},
		{name: "mapping value sentinel", credentials: map[string]any{"api_key": "synthetic", "model_mapping": map[string]any{"seedream-5.0": sentinel}}},
		{name: "mapping key live-like", credentials: map[string]any{"api_key": "synthetic", "model_mapping": map[string]any{liveLike: "seedream-5.0"}}},
		{name: "mapping value live-like", credentials: map[string]any{"api_key": "synthetic", "model_mapping": map[string]any{"seedream-5.0": liveLike}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ProtectHCAtomAccountCredentials(nil, tt.credentials, cipher)
			require.ErrorIs(t, err, ErrHCAtomCredentialInvalid)
			require.NotContains(t, err.Error(), sentinel)
			require.NotContains(t, err.Error(), liveLike)
		})
	}
}

func TestHCAtomSecretBoundary_InvalidAllowedFieldValueFailsBeforeRepository(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("26", 32))
	require.NoError(t, err)
	for _, credentials := range []map[string]any{
		{"api_key": "synthetic", "protocol": hcAtomSecretSentinel},
		{"api_key": "synthetic", "model_mapping": map[string]any{"seedream-5.0": "sk-live-abc123"}},
	} {
		repo := &hcAtomSecretBoundaryRepo{}
		svc := &adminServiceImpl{accountRepo: repo, hcAtomCredentialCipher: cipher}
		_, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
			Name: "hc-invalid-secret", Platform: PlatformHCAtom, Type: AccountTypeAPIKey,
			Credentials: credentials, SkipDefaultGroupBind: true,
		})
		require.Error(t, err)
		require.Zero(t, repo.createCalls)
		require.Nil(t, repo.account)
	}
}

func TestProtectHCAtomAccountCredentials_AllowsOnlyCurrentCatalogMappingsAndOwnsProtectedFields(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("27", 32))
	require.NoError(t, err)
	protected, err := ProtectHCAtomAccountCredentials(nil, map[string]any{
		"api_key":                   hcAtomSecretSentinel,
		"protocol":                  "hc_atom",
		HCAtomAPIKeyCiphertextField: hcAtomSecretSentinel,
		HCAtomAPIKeyMaskedField:     hcAtomSecretSentinel,
		HCAtomAPIKeyConfiguredField: false,
		"model_mapping": map[string]string{
			"seedream-5.0":      "seedream-5.0",
			HCAtomImageGPTModel: "gpt-image-2",
			"gpt-5.6-sol":       "gpt-5.6-sol",
			"claude-opus-4-6":   "claude-opus-4-6",
		},
	}, cipher)
	require.NoError(t, err)
	raw, err := json.Marshal(protected)
	require.NoError(t, err)
	require.NotContains(t, string(raw), hcAtomSecretSentinel)
	require.Equal(t, true, protected[HCAtomAPIKeyConfiguredField])
	require.Equal(t, "********7x9Q", protected[HCAtomAPIKeyMaskedField])
	require.NotEqual(t, hcAtomSecretSentinel, protected[HCAtomAPIKeyCiphertextField])
	require.Equal(t, HCAtomImageGPTModel, protected["model_mapping"].(map[string]any)[HCAtomImageGPTModel])
	require.True(t, isHCAtomBatchEnabledModel(HCAtomImageGPTModel))
}

func TestProtectHCAtomAccountCredentials_RejectsNestedModelMappingMetadata(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("24", 32))
	require.NoError(t, err)
	nestedSentinel := strings.Join([]string{"hc-review", "mapping", "secret"}, "-")

	_, err = ProtectHCAtomAccountCredentials(nil, map[string]any{
		"api_key": hcAtomSecretSentinel,
		"model_mapping": map[string]any{
			"seedream-5.0": map[string]any{"token": nestedSentinel},
		},
	}, cipher)
	require.ErrorIs(t, err, ErrHCAtomCredentialInvalid)
	require.NotContains(t, err.Error(), nestedSentinel)
}

func TestProtectHCAtomAccountCredentials_UpdateWithoutNewKeyRetainsCiphertext(t *testing.T) {
	cipher, err := NewHCAtomCredentialCipher(strings.Repeat("33", 32))
	require.NoError(t, err)
	existing, err := ProtectHCAtomAccountCredentials(nil, map[string]any{"api_key": hcAtomSecretSentinel}, cipher)
	require.NoError(t, err)
	oldCiphertext := existing[HCAtomAPIKeyCiphertextField]
	oldMasked := existing[HCAtomAPIKeyMaskedField]

	protected, err := ProtectHCAtomAccountCredentials(existing, map[string]any{
		"model_mapping":             map[string]any{"seedream-5.0": "seedream-5.0"},
		HCAtomAPIKeyCiphertextField: hcAtomSecretSentinel,
		HCAtomAPIKeyMaskedField:     hcAtomSecretSentinel,
		HCAtomAPIKeyConfiguredField: false,
	}, cipher)
	require.NoError(t, err)
	require.Equal(t, oldCiphertext, protected[HCAtomAPIKeyCiphertextField])
	require.Equal(t, oldMasked, protected[HCAtomAPIKeyMaskedField])
	require.Equal(t, true, protected[HCAtomAPIKeyConfiguredField])
	require.NotContains(t, protected, "api_key")
	raw, err := json.Marshal(protected)
	require.NoError(t, err)
	require.NotContains(t, string(raw), hcAtomSecretSentinel)
}

func mapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
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
	_, err = provider.Submit(context.Background(), &BatchImageJob{BatchID: "imgbatch_secret", Model: HCAtomImageGeminiModel}, account, BatchImageInput{
		BatchID:     "imgbatch_secret",
		Model:       HCAtomImageGeminiModel,
		AspectRatio: "1:1",
		Items:       []BatchImageInputItem{{CustomID: "item_1", Prompt: "safe synthetic prompt"}},
	})
	require.NoError(t, err)
	require.Equal(t, hcAtomSecretSentinel, client.apiKey)
}
