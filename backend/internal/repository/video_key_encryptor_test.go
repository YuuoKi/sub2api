package repository

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"

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

func TestProvideVideoKeyEncryptorFailsClosedWhenWorkerEnabled(t *testing.T) {
	_, err := ProvideVideoKeyEncryptor(&config.Config{VideoGateway: config.VideoGatewayConfig{WorkerEnabled: true}})
	require.ErrorContains(t, err, "VIDEO_GATEWAY_ENCRYPTION_KEY")
	key := strings.Repeat("11", 32)
	enc, err := ProvideVideoKeyEncryptor(&config.Config{VideoGateway: config.VideoGatewayConfig{WorkerEnabled: true, EncryptionKey: key}})
	require.NoError(t, err)
	ciphertext, err := enc.Encrypt("synthetic-provider-key")
	require.NoError(t, err)
	plaintext, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	require.Equal(t, "synthetic-provider-key", plaintext)
}
