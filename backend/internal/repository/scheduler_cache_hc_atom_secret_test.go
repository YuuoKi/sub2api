package repository

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSchedulerCacheHCAtomSnapshotNeverSerializesPlaintextSentinel(t *testing.T) {
	const sentinel = "hc-sentinel-DO-NOT-CACHE-4p8Z"
	account := service.Account{
		ID:       77,
		Platform: service.PlatformHCAtom,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":                           sentinel,
			"hc_atom_api_key":                   sentinel,
			service.HCAtomAPIKeyCiphertextField: "hc1:synthetic-ciphertext",
			service.HCAtomAPIKeyMaskedField:     "********4p8Z",
			service.HCAtomAPIKeyConfiguredField: true,
			"model_mapping":                     map[string]any{"seedream-5.0": "seedream-5.0"},
		},
	}

	for _, snapshot := range []service.Account{
		buildSchedulerCacheAccount(account),
		buildSchedulerMetadataAccount(account),
	} {
		raw, err := json.Marshal(snapshot)
		require.NoError(t, err)
		require.NotContains(t, string(raw), sentinel)
		require.NotContains(t, snapshot.Credentials, "api_key")
		require.NotContains(t, snapshot.Credentials, "hc_atom_api_key")
		require.Equal(t, "hc1:synthetic-ciphertext", snapshot.Credentials[service.HCAtomAPIKeyCiphertextField])
		require.Equal(t, "********4p8Z", snapshot.Credentials[service.HCAtomAPIKeyMaskedField])
		require.Equal(t, true, snapshot.Credentials[service.HCAtomAPIKeyConfiguredField])
	}
}
