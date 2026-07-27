package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	HCAtomAPIKeyCiphertextField = "hc_atom_api_key_ciphertext"
	HCAtomAPIKeyMaskedField     = "hc_atom_api_key_masked"
	HCAtomAPIKeyConfiguredField = "hc_atom_api_key_configured"

	hcAtomAPIKeyInputField       = "api_key"
	hcAtomLegacyAPIKeyInputField = "hc_atom_api_key"
	hcAtomCiphertextPrefix       = "hc1:"
	hcAtomCredentialAAD          = "sub2api:hc_atom:image-credential:v1"
)

var (
	ErrHCAtomCredentialKeyUnavailable = errors.New("HC-ATOM credential encryption is not configured")
	ErrHCAtomCredentialInvalid        = errors.New("HC-ATOM account credential is invalid")
	ErrHCAtomCredentialMissing        = errors.New("HC-ATOM account API key is required")
)

// HCAtomCredentialCipher is a dedicated key domain for HC image account
// credentials. It must never be backed by the video, TOTP, or JWT keys.
type HCAtomCredentialCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type hcAtomAESGCMCredentialCipher struct {
	key []byte
}

type disabledHCAtomCredentialCipher struct{}

func (disabledHCAtomCredentialCipher) Encrypt(string) (string, error) {
	return "", ErrHCAtomCredentialKeyUnavailable
}

func (disabledHCAtomCredentialCipher) Decrypt(string) (string, error) {
	return "", ErrHCAtomCredentialKeyUnavailable
}

func ProvideHCAtomCredentialCipher(cfg *config.Config) (HCAtomCredentialCipher, error) {
	if cfg == nil || !cfg.BatchImage.HCAtomEnabled {
		return disabledHCAtomCredentialCipher{}, nil
	}
	return NewHCAtomCredentialCipher(cfg.BatchImage.HCAtomEncryptionKey)
}

func NewHCAtomCredentialCipher(keyHex string) (HCAtomCredentialCipher, error) {
	keyHex = strings.TrimSpace(keyHex)
	if keyHex == "" {
		return nil, ErrHCAtomCredentialKeyUnavailable
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid HC-ATOM credential encryption key encoding")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("HC-ATOM credential encryption key must be 32 bytes")
	}
	return &hcAtomAESGCMCredentialCipher{key: key}, nil
}

func (c *hcAtomAESGCMCredentialCipher) Encrypt(plaintext string) (string, error) {
	if c == nil || len(c.key) != 32 {
		return "", ErrHCAtomCredentialKeyUnavailable
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", ErrHCAtomCredentialInvalid
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrHCAtomCredentialInvalid
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrHCAtomCredentialInvalid
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), []byte(hcAtomCredentialAAD))
	return hcAtomCiphertextPrefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (c *hcAtomAESGCMCredentialCipher) Decrypt(ciphertext string) (string, error) {
	if c == nil || len(c.key) != 32 {
		return "", ErrHCAtomCredentialKeyUnavailable
	}
	if !strings.HasPrefix(ciphertext, hcAtomCiphertextPrefix) {
		return "", ErrHCAtomCredentialInvalid
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(ciphertext, hcAtomCiphertextPrefix))
	if err != nil {
		return "", ErrHCAtomCredentialInvalid
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", ErrHCAtomCredentialInvalid
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", ErrHCAtomCredentialInvalid
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], []byte(hcAtomCredentialAAD))
	if err != nil {
		return "", ErrHCAtomCredentialInvalid
	}
	return string(plaintext), nil
}

// ProtectHCAtomAccountCredentials accepts api_key only as transient input. The
// returned map is safe to pass to repository persistence.
func ProtectHCAtomAccountCredentials(existing, incoming map[string]any, cipher HCAtomCredentialCipher) (map[string]any, error) {
	out := make(map[string]any, len(incoming)+3)
	for key, value := range incoming {
		switch key {
		case hcAtomAPIKeyInputField, hcAtomLegacyAPIKeyInputField,
			HCAtomAPIKeyCiphertextField, HCAtomAPIKeyMaskedField, HCAtomAPIKeyConfiguredField:
			continue
		default:
			out[key] = value
		}
	}

	existingCiphertext := credentialString(existing, HCAtomAPIKeyCiphertextField)
	existingMasked := credentialString(existing, HCAtomAPIKeyMaskedField)
	rawAPIKey, provided := incoming[hcAtomAPIKeyInputField]
	apiKey := ""
	if provided {
		var ok bool
		apiKey, ok = rawAPIKey.(string)
		if !ok {
			return nil, ErrHCAtomCredentialInvalid
		}
		apiKey = strings.TrimSpace(apiKey)
	}

	if apiKey != "" {
		if cipher == nil {
			return nil, ErrHCAtomCredentialKeyUnavailable
		}
		encrypted, err := cipher.Encrypt(apiKey)
		if err != nil {
			return nil, ErrHCAtomCredentialInvalid
		}
		out[HCAtomAPIKeyCiphertextField] = encrypted
		out[HCAtomAPIKeyMaskedField] = maskHCAtomAPIKey(apiKey)
		out[HCAtomAPIKeyConfiguredField] = true
		return out, nil
	}

	if existingCiphertext == "" {
		return nil, ErrHCAtomCredentialMissing
	}
	out[HCAtomAPIKeyCiphertextField] = existingCiphertext
	if existingMasked == "" {
		existingMasked = "********"
	}
	out[HCAtomAPIKeyMaskedField] = existingMasked
	out[HCAtomAPIKeyConfiguredField] = true
	return out, nil
}

// ResolveHCAtomAPIKey decrypts only after the dedicated HC account was chosen.
// Errors intentionally omit ciphertext, plaintext, and wrapped crypto details.
func ResolveHCAtomAPIKey(account *Account, cipher HCAtomCredentialCipher) (string, error) {
	if account == nil || account.Platform != PlatformHCAtom || account.Type != AccountTypeAPIKey {
		return "", ErrHCAtomCredentialInvalid
	}
	if cipher == nil {
		return "", ErrHCAtomCredentialKeyUnavailable
	}
	ciphertext := credentialString(account.Credentials, HCAtomAPIKeyCiphertextField)
	if ciphertext == "" {
		return "", ErrHCAtomCredentialMissing
	}
	plaintext, err := cipher.Decrypt(ciphertext)
	if err != nil {
		return "", ErrHCAtomCredentialInvalid
	}
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return "", ErrHCAtomCredentialInvalid
	}
	return plaintext, nil
}

func credentialString(credentials map[string]any, key string) string {
	if credentials == nil {
		return ""
	}
	value, _ := credentials[key].(string)
	return strings.TrimSpace(value)
}

func maskHCAtomAPIKey(apiKey string) string {
	runes := []rune(strings.TrimSpace(apiKey))
	if len(runes) <= 4 {
		return "********"
	}
	return "********" + string(runes[len(runes)-4:])
}
