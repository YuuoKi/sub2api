package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestPackUnpackKlingCredentialBlob(t *testing.T) {
	ak := "kling-access-key-abc12345"
	sk := "kling-secret-key-xyz98765"

	blob, err := packKlingCredentialBlob(ak, sk)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if !strings.Contains(blob, `"v":1`) || !strings.Contains(blob, `"auth":"kling_aksk"`) {
		t.Fatalf("blob missing version/auth markers: %s", blob)
	}
	if strings.Contains(blob, "enc:") {
		t.Fatalf("pack must return plaintext JSON, not ciphertext: %s", blob)
	}

	gotAK, gotSK, ok := unpackKlingCredentialBlob(blob)
	if !ok {
		t.Fatal("expected unpack ok")
	}
	if gotAK != ak || gotSK != sk {
		t.Fatalf("unpack mismatch: ak=%q sk=%q", gotAK, gotSK)
	}
}

func TestUnpackKlingCredentialBlobRejectsRawKey(t *testing.T) {
	_, _, ok := unpackKlingCredentialBlob("seedance-raw-api-key-not-json")
	if ok {
		t.Fatal("raw single key must not parse as kling blob")
	}
	_, _, ok = unpackKlingCredentialBlob(`{"v":1,"auth":"bearer","access_key":"a","secret_key":"b"}`)
	if ok {
		t.Fatal("wrong auth mode must not parse as kling blob")
	}
}

func TestDecryptProviderKeyKlingBlobPopulatesAKSK(t *testing.T) {
	svc := NewVideoGatewayService(newMemoryVideoGatewayRepo(), noopVideoKeyEncryptor{}, nil)
	blob, err := packKlingCredentialBlob("ak-kling-test-00112233", "sk-kling-test-44556677")
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	enc, err := svc.encryptor.Encrypt(blob)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	account := &VideoProviderAccount{
		Provider:        VideoProviderKling,
		EncryptedAPIKey: enc,
	}
	svc.decryptProviderKey(account)
	if account.PlainAPIKey != "" {
		t.Fatalf("kling blob must not populate PlainAPIKey, got %q", account.PlainAPIKey)
	}
	if account.PlainAccessKey != "ak-kling-test-00112233" {
		t.Fatalf("PlainAccessKey=%q", account.PlainAccessKey)
	}
	if account.PlainSecretKey != "sk-kling-test-44556677" {
		t.Fatalf("PlainSecretKey=%q", account.PlainSecretKey)
	}
	if !account.APIKeyConfigured {
		t.Fatal("APIKeyConfigured must stay true when ciphertext non-empty")
	}
}

func TestDecryptProviderKeySeedanceRawStaysPlainAPIKey(t *testing.T) {
	svc := NewVideoGatewayService(newMemoryVideoGatewayRepo(), noopVideoKeyEncryptor{}, nil)
	enc, err := svc.encryptor.Encrypt("seedance-plain-key-aabbccdd")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	account := &VideoProviderAccount{
		Provider:        VideoProviderSeedance,
		EncryptedAPIKey: enc,
	}
	svc.decryptProviderKey(account)
	if account.PlainAPIKey != "seedance-plain-key-aabbccdd" {
		t.Fatalf("PlainAPIKey=%q", account.PlainAPIKey)
	}
	if account.PlainAccessKey != "" || account.PlainSecretKey != "" {
		t.Fatalf("seedance must not set AK/SK: ak=%q sk=%q", account.PlainAccessKey, account.PlainSecretKey)
	}
}

func TestApplyKlingCredentialsStoresVersionedBlob(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	created, err := svc.CreateProviderAccount(t.Context(), VideoProviderCreateParams{
		Provider:    VideoProviderKling,
		DisplayName: "Kling Main",
		Enabled:     true,
		AccessKey:   "ak-create-111122223333",
		SecretKey:   "sk-create-444455556666",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	stored := repo.providers[created.ID]
	if stored == nil || stored.EncryptedAPIKey == "" {
		t.Fatal("expected encrypted ciphertext stored")
	}
	plain, err := svc.encryptor.Decrypt(stored.EncryptedAPIKey)
	if err != nil {
		t.Fatalf("decrypt stored: %v", err)
	}
	ak, sk, ok := unpackKlingCredentialBlob(plain)
	if !ok || ak != "ak-create-111122223333" || sk != "sk-create-444455556666" {
		t.Fatalf("stored blob unpack failed: ok=%v ak=%q sk=%q plain=%s", ok, ak, sk, plain)
	}
	if created.AuthMode != videoAuthModeKlingAKSK {
		t.Fatalf("AuthMode=%q", created.AuthMode)
	}
	if created.MaskedKey == "" || strings.Contains(created.MaskedKey, "sk-create") {
		t.Fatalf("MaskedKey should mask AK only, got %q", created.MaskedKey)
	}
	if created.PlainAccessKey != "" || created.PlainSecretKey != "" || created.PlainAPIKey != "" {
		t.Fatal("response must not leak plaintext credentials")
	}
}

func TestApplySeedanceAPIKeyStillRawString(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	created, err := svc.CreateProviderAccount(t.Context(), VideoProviderCreateParams{
		Provider: VideoProviderSeedance,
		APIKey:   "seedance-raw-key-77889900",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	plain, err := svc.encryptor.Decrypt(repo.providers[created.ID].EncryptedAPIKey)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plain != "seedance-raw-key-77889900" {
		t.Fatalf("seedance must store raw key, got %q", plain)
	}
	if created.AuthMode != videoAuthModeBearer {
		t.Fatalf("AuthMode=%q", created.AuthMode)
	}
}

func TestUpdateKlingCredentialsLeaveEmptyKeepsExisting(t *testing.T) {
	repo := newMemoryVideoGatewayRepo()
	svc := NewVideoGatewayService(repo, noopVideoKeyEncryptor{}, nil)
	created, err := svc.CreateProviderAccount(t.Context(), VideoProviderCreateParams{
		Provider:  VideoProviderKling,
		AccessKey: "ak-keep-aaaaaaaaaaaa",
		SecretKey: "sk-keep-bbbbbbbbbbbb",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Omit both → keep
	emptyAK := ""
	emptySK := ""
	if _, err := svc.UpdateProviderAccount(t.Context(), created.ID, VideoProviderUpdateParams{
		AccessKey: &emptyAK,
		SecretKey: &emptySK,
	}); err != nil {
		t.Fatalf("update omit both: %v", err)
	}
	plain, _ := svc.encryptor.Decrypt(repo.providers[created.ID].EncryptedAPIKey)
	ak, sk, ok := unpackKlingCredentialBlob(plain)
	if !ok || ak != "ak-keep-aaaaaaaaaaaa" || sk != "sk-keep-bbbbbbbbbbbb" {
		t.Fatalf("leave-empty both changed blob: ak=%q sk=%q ok=%v", ak, sk, ok)
	}

	// Update only AK
	newAK := "ak-new-cccccccccccc"
	if _, err := svc.UpdateProviderAccount(t.Context(), created.ID, VideoProviderUpdateParams{
		AccessKey: &newAK,
	}); err != nil {
		t.Fatalf("update ak: %v", err)
	}
	plain, _ = svc.encryptor.Decrypt(repo.providers[created.ID].EncryptedAPIKey)
	ak, sk, ok = unpackKlingCredentialBlob(plain)
	if !ok || ak != "ak-new-cccccccccccc" || sk != "sk-keep-bbbbbbbbbbbb" {
		t.Fatalf("partial AK update failed: ak=%q sk=%q ok=%v", ak, sk, ok)
	}

	// Update only SK
	newSK := "sk-new-dddddddddddd"
	if _, err := svc.UpdateProviderAccount(t.Context(), created.ID, VideoProviderUpdateParams{
		SecretKey: &newSK,
	}); err != nil {
		t.Fatalf("update sk: %v", err)
	}
	plain, _ = svc.encryptor.Decrypt(repo.providers[created.ID].EncryptedAPIKey)
	ak, sk, ok = unpackKlingCredentialBlob(plain)
	if !ok || ak != "ak-new-cccccccccccc" || sk != "sk-new-dddddddddddd" {
		t.Fatalf("partial SK update failed: ak=%q sk=%q ok=%v", ak, sk, ok)
	}
}

func TestKlingMintJWTClaimsAndCache(t *testing.T) {
	ak := "ak-jwt-issuer-00112233"
	sk := "sk-jwt-signer-44556677"
	now := time.Now()

	token1, err := klingMintJWT(ak, sk)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if token1 == "" {
		t.Fatal("empty token")
	}

	parsed, err := jwt.Parse(token1, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			t.Fatalf("alg=%v", token.Method)
		}
		return []byte(sk), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims type")
	}
	if fmt.Sprint(claims["iss"]) != ak {
		t.Fatalf("iss=%v", claims["iss"])
	}
	expF, _ := claims["exp"].(float64)
	nbfF, _ := claims["nbf"].(float64)
	exp := int64(expF)
	nbf := int64(nbfF)
	if exp < now.Unix()+1790 || exp > now.Unix()+1810 {
		t.Fatalf("exp=%d want ~now+1800", exp)
	}
	if nbf < now.Unix()-10 || nbf > now.Unix() {
		t.Fatalf("nbf=%d want ~now-5", nbf)
	}

	token2, err := klingMintJWT(ak, sk)
	if err != nil {
		t.Fatalf("mint2: %v", err)
	}
	if token2 != token1 {
		t.Fatal("expected in-process cache hit before exp-60s")
	}
}

func TestRedactVideoUpstreamSecretsStripsKlingAKSKAndJWT(t *testing.T) {
	ak := "akKlingRedact0011223344"
	sk := "skKlingRedact5566778899"
	token, err := klingMintJWT(ak, sk)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	body := fmt.Sprintf("auth failed ak=%s sk=%s jwt=%s Bearer %s", ak, sk, token, token)
	out := redactVideoUpstreamSecretsForKeys(body, ak, sk, token)
	if strings.Contains(out, ak) || strings.Contains(out, sk) || strings.Contains(out, token) {
		t.Fatalf("leaked secrets: %s", out)
	}
	if !strings.Contains(out, "已脱敏") {
		t.Fatalf("expected redaction marker: %s", out)
	}

	account := &VideoProviderAccount{
		PlainAccessKey: ak,
		PlainSecretKey: sk,
	}
	out2 := redactVideoUpstreamSecretsForAccount(account, body)
	if strings.Contains(out2, ak) || strings.Contains(out2, sk) || strings.Contains(out2, token) {
		t.Fatalf("account redaction leaked: %s", out2)
	}
}

func TestKlingAdapterGatesOnAKSKNotPlainAPIKey(t *testing.T) {
	adapter := &klingVideoAdapter{}
	account := &VideoProviderAccount{
		Provider:         VideoProviderKling,
		APIKeyConfigured: true,
		PlainAPIKey:      "",
		PlainAccessKey:   "ak-kling-gate-00112233",
		PlainSecretKey:   "sk-kling-gate-44556677",
	}
	// Without smoke-gate env, configured AK/SK must pass the credential gate and
	// fail later (smoke gate / model allowlist) — never "api key is not configured".
	_, createErr := adapter.CreateTask(t.Context(), account, &VideoTask{Model: "kling-v1", TaskType: VideoTaskTypeTextToVideo, Duration: 5})
	if createErr == nil {
		t.Fatal("expected post-credential gate error, got nil")
	}
	if createErr == ErrVideoProviderDisabled && strings.Contains(createErr.Error(), "api key is not configured") {
		t.Fatalf("configured Kling AK/SK must not look unconfigured: %v", createErr)
	}
	if strings.Contains(createErr.Error(), "api key is not configured") {
		t.Fatalf("configured Kling AK/SK must not look unconfigured: %v", createErr)
	}
	_, pollErr := adapter.PollTask(t.Context(), account, &VideoTask{Model: "kling-v1", TaskType: VideoTaskTypeTextToVideo, Duration: 5, UpstreamTaskID: "x"})
	if pollErr == nil {
		t.Fatal("expected post-credential gate error, got nil")
	}
	if strings.Contains(pollErr.Error(), "api key is not configured") {
		t.Fatalf("configured Kling AK/SK must not look unconfigured on poll: %v", pollErr)
	}

	assertKlingUnconfigured := func(t *testing.T, acc *VideoProviderAccount, label string) {
		t.Helper()
		_, err := adapter.CreateTask(t.Context(), acc, &VideoTask{})
		if err == nil {
			t.Fatalf("%s: expected unconfigured error, got nil", label)
		}
		if !errors.Is(err, ErrVideoProviderDisabled) && !strings.Contains(err.Error(), "api key is not configured") {
			t.Fatalf("%s: want unconfigured path, got %v", label, err)
		}
		_, err = adapter.PollTask(t.Context(), acc, &VideoTask{})
		if err == nil {
			t.Fatalf("%s poll: expected unconfigured error, got nil", label)
		}
		if !errors.Is(err, ErrVideoProviderDisabled) && !strings.Contains(err.Error(), "api key is not configured") {
			t.Fatalf("%s poll: want unconfigured path, got %v", label, err)
		}
	}

	assertKlingUnconfigured(t, &VideoProviderAccount{
		Provider:         VideoProviderKling,
		APIKeyConfigured: true,
		PlainAccessKey:   "",
		PlainSecretKey:   "",
	}, "configured flag with empty AK/SK")

	assertKlingUnconfigured(t, &VideoProviderAccount{
		Provider:         VideoProviderKling,
		APIKeyConfigured: true,
		PlainAPIKey:      "seedance-shaped-key-should-not-count",
		PlainAccessKey:   "",
		PlainSecretKey:   "",
	}, "PlainAPIKey alone on Kling")
}

func TestVideoProviderUpstreamEchoedCredentialChecksAKSK(t *testing.T) {
	ak := "akEchoCheck001122334455"
	sk := "skEchoCheck556677889900"
	account := &VideoProviderAccount{
		Provider:         VideoProviderKling,
		APIKeyConfigured: true,
		PlainAccessKey:   ak,
		PlainSecretKey:   sk,
	}
	if !seedanceUpstreamEchoedKey(account, "upstream rejected "+ak) {
		t.Fatal("expected AK echo to be detected")
	}
	if !seedanceUpstreamEchoedKey(account, "upstream rejected "+sk) {
		t.Fatal("expected SK echo to be detected")
	}
	token, err := klingMintJWT(ak, sk)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if !seedanceUpstreamEchoedKey(account, "Bearer "+token) {
		t.Fatal("expected derived JWT echo to be detected")
	}
	if seedanceUpstreamEchoedKey(account, "no secrets here") {
		t.Fatal("neutral body must not look like an echo")
	}
}

func TestRedactVideoUpstreamSecretsForAccountStripsJWTWhenMintUnavailable(t *testing.T) {
	// Only AK present → klingMintJWT cannot run, but JWT-shaped tokens in the body
	// must still be stripped by the pattern pass (fail-closed).
	account := &VideoProviderAccount{
		Provider:       VideoProviderKling,
		PlainAccessKey: "akPartialOnly00112233",
		PlainSecretKey: "",
	}
	jwtEcho := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJha1BhcnRpYWxPbmx5MDAxMTIyMzMifQ.signaturepartXX"
	body := "auth failed ak=akPartialOnly00112233 jwt=" + jwtEcho
	out := redactVideoUpstreamSecretsForAccount(account, body)
	if strings.Contains(out, "akPartialOnly00112233") {
		t.Fatalf("AK must still be stripped: %s", out)
	}
	if strings.Contains(out, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") || strings.Contains(out, jwtEcho) {
		t.Fatalf("JWT-shaped token must be stripped even when mint is unavailable: %s", out)
	}
}

func TestVideoProviderAccountStringAndLogValueRedactKlingFields(t *testing.T) {
	acc := VideoProviderAccount{
		ID:               7,
		Provider:         VideoProviderKling,
		DisplayName:      "Kling",
		PlainAPIKey:      "should-not-appear-api",
		PlainAccessKey:   "should-not-appear-ak",
		PlainSecretKey:   "should-not-appear-sk",
		APIKeyConfigured: true,
		MaskedKey:        "ak**sk",
	}
	s := acc.String()
	gs := acc.GoString()
	for _, dump := range []string{s, gs} {
		for _, secret := range []string{"should-not-appear-api", "should-not-appear-ak", "should-not-appear-sk"} {
			if strings.Contains(dump, secret) {
				t.Fatalf("String/GoString leaked %q in %q", secret, dump)
			}
		}
		if !strings.Contains(dump, "REDACTED") {
			t.Fatalf("expected REDACTED marker in %q", dump)
		}
	}
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("acc", "account", acc)
	logged := buf.String()
	for _, secret := range []string{"should-not-appear-api", "should-not-appear-ak", "should-not-appear-sk"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("LogValue leaked %q via slog: %s", secret, logged)
		}
	}
}
