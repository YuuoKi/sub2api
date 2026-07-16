package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewVideoKeyEncryptorRequiresDedicatedKeyAndRoundTrips(t *testing.T) {
	_, err := NewVideoKeyEncryptor("")
	require.ErrorContains(t, err, "VIDEO_GATEWAY_ENCRYPTION_KEY")

	encryptor, err := NewVideoKeyEncryptor(strings.Repeat("11", 32))
	require.NoError(t, err)
	ciphertext, err := encryptor.Encrypt("test-only-video-credential")
	require.NoError(t, err)
	require.NotContains(t, ciphertext, "test-only-video-credential")
	plaintext, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Equal(t, "test-only-video-credential", plaintext)
}
