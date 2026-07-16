package repository

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// NewVideoKeyEncryptor builds the video credential encryptor from its dedicated
// key domain. It intentionally cannot read or fall back to the TOTP key.
func NewVideoKeyEncryptor(keyHex string) (service.VideoKeyEncryptor, error) {
	keyHex = strings.TrimSpace(keyHex)
	if keyHex == "" {
		return nil, fmt.Errorf("VIDEO_GATEWAY_ENCRYPTION_KEY is required for the dedicated video credential key domain")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid video gateway encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("video gateway encryption key must be 32 bytes, got %d", len(key))
	}
	return &AESEncryptor{key: key}, nil
}
