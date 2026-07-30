package repository

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSchedulerCacheHCAtomSnapshotNeverSerializesPlaintextSentinel(t *testing.T) {
	sentinel := strings.Join([]string{"hc-sentinel", "DO-NOT-PERSIST", "7x9Q"}, "-")
	liveLike := strings.Join([]string{"sk", "live", "abc123"}, "-")
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
			"protocol":                          sentinel,
			"model_mapping": map[string]any{
				"seedream-5.0": liveLike,
			},
			"metadata": map[string]any{
				"Authorization": "Bearer " + sentinel,
				"nested":        []any{map[string]any{"api_key": liveLike}},
			},
		},
	}

	for _, snapshot := range []service.Account{
		buildSchedulerCacheAccount(account),
		buildSchedulerMetadataAccount(account),
	} {
		raw, err := json.Marshal(snapshot)
		require.NoError(t, err)
		require.NotContains(t, string(raw), sentinel)
		require.NotContains(t, string(raw), liveLike)
		require.NotContains(t, snapshot.Credentials, "api_key")
		require.NotContains(t, snapshot.Credentials, "hc_atom_api_key")
		require.Equal(t, "hc1:synthetic-ciphertext", snapshot.Credentials[service.HCAtomAPIKeyCiphertextField])
		require.Equal(t, "********4p8Z", snapshot.Credentials[service.HCAtomAPIKeyMaskedField])
		require.Equal(t, true, snapshot.Credentials[service.HCAtomAPIKeyConfiguredField])
		require.NotContains(t, snapshot.Credentials, "protocol")
		require.NotContains(t, snapshot.Credentials, "model_mapping")
	}
}

func TestSchedulerCacheHCAtomSnapshotKeepsDispatchableCatalogMappings(t *testing.T) {
	account := service.Account{
		Platform: service.PlatformHCAtom,
		Credentials: map[string]any{
			"protocol": "hc_atom",
			"model_mapping": map[string]any{
				"seedream-5.0":  "seedream-5.0",
				"s-gpt-image-2": "s-gpt-image-2",
			},
		},
	}
	got := buildSchedulerCacheAccount(account).Credentials
	require.Equal(t, "hc_atom", got["protocol"])
	require.Equal(t, map[string]any{
		"seedream-5.0":  "seedream-5.0",
		"s-gpt-image-2": "s-gpt-image-2",
	}, got["model_mapping"])
}

func TestSchedulerCacheHCAtomSnapshotDropsMappingContainingDisabledDola(t *testing.T) {
	account := service.Account{
		Platform: service.PlatformHCAtom,
		Credentials: map[string]any{
			"protocol": "hc_atom",
			"model_mapping": map[string]any{
				"seedream-5.0":          "seedream-5.0",
				"dola-seedream-5.0-pro": "dola-seedream-5.0-pro",
			},
		},
	}

	got := buildSchedulerCacheAccount(account).Credentials
	require.Equal(t, "hc_atom", got["protocol"])
	require.NotContains(t, got, "model_mapping")
}
