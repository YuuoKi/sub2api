package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	videoAuthModeBearer    = "bearer"
	videoAuthModeKlingAKSK = "kling_aksk"

	klingCredentialBlobVersion = 1
	klingJWTTTLSeconds         = 1800
	klingJWTNBFSkewSeconds     = 5
	klingJWTCacheSkewSeconds   = 60
)

type klingCredentialBlob struct {
	V         int    `json:"v"`
	Auth      string `json:"auth"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type klingJWTCacheEntry struct {
	token     string
	expiresAt time.Time
}

var (
	klingJWTCacheMu sync.Mutex
	klingJWTCache   = map[string]klingJWTCacheEntry{}
)

func packKlingCredentialBlob(accessKey, secretKey string) (string, error) {
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	if accessKey == "" || secretKey == "" {
		return "", fmt.Errorf("kling access_key and secret_key are required")
	}
	raw, err := json.Marshal(klingCredentialBlob{
		V:         klingCredentialBlobVersion,
		Auth:      videoAuthModeKlingAKSK,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		return "", fmt.Errorf("marshal kling credential blob: %w", err)
	}
	return string(raw), nil
}

func unpackKlingCredentialBlob(plain string) (accessKey, secretKey string, ok bool) {
	plain = strings.TrimSpace(plain)
	if plain == "" || plain[0] != '{' {
		return "", "", false
	}
	var blob klingCredentialBlob
	if err := json.Unmarshal([]byte(plain), &blob); err != nil {
		return "", "", false
	}
	if blob.V != klingCredentialBlobVersion || blob.Auth != videoAuthModeKlingAKSK {
		return "", "", false
	}
	accessKey = strings.TrimSpace(blob.AccessKey)
	secretKey = strings.TrimSpace(blob.SecretKey)
	if accessKey == "" || secretKey == "" {
		return "", "", false
	}
	return accessKey, secretKey, true
}

// klingMintJWT builds an HS256 JWT for Kling (iss=AK, exp=now+1800, nbf=now-5).
// Tokens are cached in-process until exp-60s and are never persisted.
func klingMintJWT(accessKey, secretKey string) (string, error) {
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	if accessKey == "" || secretKey == "" {
		return "", fmt.Errorf("kling access_key and secret_key are required")
	}
	cacheKey := accessKey + "\x00" + secretKey
	now := time.Now()

	klingJWTCacheMu.Lock()
	defer klingJWTCacheMu.Unlock()
	if entry, ok := klingJWTCache[cacheKey]; ok {
		// Usable until (exp - 60s).
		if now.Before(entry.expiresAt.Add(-klingJWTCacheSkewSeconds * time.Second)) {
			return entry.token, nil
		}
	}

	exp := now.Add(klingJWTTTLSeconds * time.Second)
	nbf := now.Add(-klingJWTNBFSkewSeconds * time.Second)
	claims := jwt.MapClaims{
		"iss": accessKey,
		"exp": exp.Unix(),
		"nbf": nbf.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("sign kling jwt: %w", err)
	}
	klingJWTCache[cacheKey] = klingJWTCacheEntry{token: signed, expiresAt: exp}
	return signed, nil
}

func videoProviderAuthMode(provider string, configured bool) string {
	if !configured {
		return ""
	}
	if strings.TrimSpace(provider) == VideoProviderKling {
		return videoAuthModeKlingAKSK
	}
	return videoAuthModeBearer
}

func videoProviderHasPlainCredentials(account *VideoProviderAccount) bool {
	if account == nil {
		return false
	}
	if account.Provider == VideoProviderKling {
		return strings.TrimSpace(account.PlainAccessKey) != "" && strings.TrimSpace(account.PlainSecretKey) != ""
	}
	return strings.TrimSpace(account.PlainAPIKey) != ""
}

// videoProviderKnownSecrets returns every transient plaintext credential that must
// be stripped from upstream bodies / checked for hostile echoes: Seedance API key,
// Kling AK/SK, and a freshly minted Kling JWT when AK+SK are both present.
func videoProviderKnownSecrets(account *VideoProviderAccount) []string {
	if account == nil {
		return nil
	}
	keys := make([]string, 0, 4)
	for _, raw := range []string{account.PlainAPIKey, account.PlainAccessKey, account.PlainSecretKey} {
		if k := strings.TrimSpace(raw); k != "" {
			keys = append(keys, k)
		}
	}
	ak := strings.TrimSpace(account.PlainAccessKey)
	sk := strings.TrimSpace(account.PlainSecretKey)
	if ak != "" && sk != "" {
		if token, err := klingMintJWT(ak, sk); err == nil && token != "" {
			keys = append(keys, token)
		}
	}
	return keys
}

// videoProviderUpstreamEchoedCredential reports whether rawBody contains any
// known account secret (API key / AK / SK / derived JWT) verbatim.
func videoProviderUpstreamEchoedCredential(account *VideoProviderAccount, rawBody string) bool {
	if account == nil || rawBody == "" {
		return false
	}
	for _, key := range videoProviderKnownSecrets(account) {
		if len(key) < videoKnownSecretMinLen {
			continue
		}
		if strings.Contains(rawBody, key) {
			return true
		}
	}
	return false
}
