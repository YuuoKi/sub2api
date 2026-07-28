package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestProvideHCAtomCredentialCipherConfiguredWhileDispatchDisabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.BatchImage.HCAtomEncryptionKey = strings.Repeat("71", 32)

	cipher, err := ProvideHCAtomCredentialCipher(cfg)
	require.NoError(t, err)

	ciphertext, err := cipher.Encrypt("synthetic-hc-atom-api-key")
	require.NoError(t, err)
	require.NotContains(t, ciphertext, "synthetic-hc-atom-api-key")

	plaintext, err := cipher.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Equal(t, "synthetic-hc-atom-api-key", plaintext)
}

func TestProvideHCAtomCredentialCipherEmptyWhileDispatchDisabled(t *testing.T) {
	cipher, err := ProvideHCAtomCredentialCipher(&config.Config{})
	require.NoError(t, err)

	_, err = cipher.Encrypt("synthetic-hc-atom-api-key")
	require.ErrorIs(t, err, ErrHCAtomCredentialKeyUnavailable)
}

func TestProvideHCAtomCredentialCipherValidatesConfiguredKey(t *testing.T) {
	cfg := &config.Config{}
	cfg.BatchImage.HCAtomEncryptionKey = "not-a-valid-key"

	_, err := ProvideHCAtomCredentialCipher(cfg)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrHCAtomCredentialKeyUnavailable))
}
