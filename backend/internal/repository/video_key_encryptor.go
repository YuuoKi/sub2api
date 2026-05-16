package repository

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// NewVideoKeyEncryptor creates the dedicated reversible encryptor for video
// upstream API keys. Production deployments should set video_gateway.encryption_key.
func NewVideoKeyEncryptor(cfg *config.Config) (service.VideoKeyEncryptor, error) {
	keyHex := strings.TrimSpace(cfg.VideoGateway.EncryptionKey)
	if keyHex == "" {
		keyHex = strings.TrimSpace(cfg.Totp.EncryptionKey)
		slog.Warn("video_gateway.encryption_key is empty; using dev-only fallback from totp.encryption_key")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid video gateway encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("video gateway encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	return &AESEncryptor{key: key}, nil
}
