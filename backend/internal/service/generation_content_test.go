package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type generationContentMemoryRepo struct {
	rows  []*GenerationContent
	err   error
	panic bool
}

func (r *generationContentMemoryRepo) Create(_ context.Context, content *GenerationContent) error {
	if r.panic {
		panic("generation content repository panic")
	}
	if r.err != nil {
		return r.err
	}
	r.rows = append(r.rows, content)
	return nil
}

func (r *generationContentMemoryRepo) GetCaptureStats(context.Context) (*GenerationContentStats, error) {
	return &GenerationContentStats{}, r.err
}

func (r *generationContentMemoryRepo) GetRecent(context.Context, int) ([]GenerationContentSample, error) {
	return nil, r.err
}

func (r *generationContentMemoryRepo) UpdateTaskAdoption(context.Context, GenerationContentAdoptionInput) (*GenerationContentAdoption, error) {
	return nil, r.err
}

func (r *generationContentMemoryRepo) GetWeeklyReport(context.Context, time.Time, time.Time) (*GenerationContentWeeklyReport, error) {
	return &GenerationContentWeeklyReport{}, r.err
}

func generationCaptureConfig(promptMax, responseMax int) *config.Config {
	return &config.Config{Gateway: config.GatewayConfig{ContentCapture: config.ContentCaptureConfig{
		Enabled:          true,
		PromptMaxBytes:   promptMax,
		ResponseMaxBytes: responseMax,
	}}}
}

func TestGenerationContentCollectorRedactsBoundsAndAttributes(t *testing.T) {
	repo := &generationContentMemoryRepo{}
	collector := NewGenerationContentCollector(repo, generationCaptureConfig(96, 32))
	groupID := int64(7)
	prompt := []byte(`{"access_token":"tok_supersecret","messages":[{"content":"mail a@b.com call 13800138000 ` + strings.Repeat("x", 200) + `"}]}`)

	collector.Collect(context.Background(), GenerationContentCaptureArgs{
		RequestID:          "req-content-1",
		APIKeyID:           22,
		UserID:             11,
		GroupID:            &groupID,
		AccountID:          33,
		Model:              "claude-opus-4-8",
		RequestPayloadHash: "sha256:test",
		PromptBody:         prompt,
		Result: &ForwardResult{
			ResponseSample:    []byte("Bearer abcdef1234567890XYZ contact x@y.com"),
			ResponseBytes:     120,
			ResponseTruncated: true,
		},
	})

	if len(repo.rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(repo.rows))
	}
	row := repo.rows[0]
	if row.APIKeyID == nil || *row.APIKeyID != 22 || row.UserID == nil || *row.UserID != 11 || row.GroupID == nil || *row.GroupID != 7 || row.AccountID == nil || *row.AccountID != 33 {
		t.Fatalf("attribution mismatch: %+v", row)
	}
	for _, leak := range []string{"tok_supersecret", "a@b.com", "13800138000", "abcdef1234567890XYZ", "x@y.com"} {
		if strings.Contains(row.PromptRedacted+row.ResponseRedacted, leak) {
			t.Fatalf("redaction leaked %q: prompt=%q response=%q", leak, row.PromptRedacted, row.ResponseRedacted)
		}
	}
	if row.PromptBytes != len(prompt) {
		t.Fatalf("prompt bytes=%d, want original %d", row.PromptBytes, len(prompt))
	}
	if len(row.PromptRedacted) > 96 {
		t.Fatalf("bounded prompt len=%d, want <=96", len(row.PromptRedacted))
	}
	if row.ResponseBytes != 120 || !row.ResponseTruncated {
		t.Fatalf("response evidence mismatch: bytes=%d truncated=%v", row.ResponseBytes, row.ResponseTruncated)
	}
	if row.RedactionVersion <= 0 {
		t.Fatalf("redaction version=%d, want positive", row.RedactionVersion)
	}
}

func TestGenerationContentRedactionCoversStructuredSecrets(t *testing.T) {
	const (
		cnID   = "11010519491231002X"
		card   = "4111 1111 1111 1111"
		opaque = "Ab3Cd6Ef9Gh2Jk5Lm8Np1Qr4"
	)
	out := redactGenerationPrompt([]byte(`{"messages":[{"content":"身份证` + cnID + ` 卡号` + card + ` token=` + opaque + `"}]}`))
	for _, leak := range []string{cnID, card, opaque} {
		if strings.Contains(out, leak) {
			t.Fatalf("structured secret leaked %q: %s", leak, out)
		}
	}
	for _, marker := range []string{"[ID]", "[CARD]", "[已脱敏]"} {
		if !strings.Contains(out, marker) {
			t.Fatalf("missing marker %q: %s", marker, out)
		}
	}
}

func TestGenerationContentStructuredRedactionPreservesOrdinaryIdentifiers(t *testing.T) {
	for _, value := range []string{
		"1234567890123456",
		"123456789012345678",
		"12345678901234567890",
		"antidisestablishmentarianism",
		"claude-opus-4-8",
	} {
		if got := redactGenerationStructuredPII(value); got != value {
			t.Fatalf("ordinary identifier %q changed to %q", value, got)
		}
	}
}

func TestGatewayGenerationPromptSnapshotDisabledReturnsNil(t *testing.T) {
	svc := &GatewayService{cfg: &config.Config{}}
	snapshot := svc.SnapshotGenerationPrompt([]byte(strings.Repeat("x", 1024)))
	if snapshot.Body != nil || snapshot.OriginalBytes != 0 || snapshot.Truncated {
		t.Fatalf("disabled capture created snapshot: %+v", snapshot)
	}
}

func TestGatewayGenerationPromptSnapshotEnabledCopiesOnlyBoundedPrefix(t *testing.T) {
	svc := &GatewayService{cfg: generationCaptureConfig(64, 32)}
	body := []byte(strings.Repeat("a", 1024))
	snapshot := svc.SnapshotGenerationPrompt(body)
	if len(snapshot.Body) != 64 || snapshot.OriginalBytes != len(body) || !snapshot.Truncated {
		t.Fatalf("snapshot mismatch: body=%d original=%d truncated=%v", len(snapshot.Body), snapshot.OriginalBytes, snapshot.Truncated)
	}
	body[0] = 'z'
	if snapshot.Body[0] != 'a' {
		t.Fatal("snapshot must own its bounded bytes")
	}
}

func TestGatewayGenerationPromptSnapshotHonorsHardCap(t *testing.T) {
	svc := &GatewayService{cfg: generationCaptureConfig(1<<30, 32)}
	snapshot := svc.SnapshotGenerationPrompt([]byte(strings.Repeat("x", maxGenerationPromptMaxBytes+1024)))
	if len(snapshot.Body) != maxGenerationPromptMaxBytes || !snapshot.Truncated {
		t.Fatalf("hard-cap snapshot body=%d truncated=%v", len(snapshot.Body), snapshot.Truncated)
	}
}

func TestGenerationContentCollectorProcessesBoundedSnapshotAndKeepsOriginalByteCount(t *testing.T) {
	repo := &generationContentMemoryRepo{}
	collector := NewGenerationContentCollector(repo, generationCaptureConfig(96, 32))
	prompt := []byte(`{"messages":[{"content":"safe-prefix ` + strings.Repeat("long text ", 80) + `"}]}`)
	collector.Collect(context.Background(), GenerationContentCaptureArgs{RequestID: "req-long-json", PromptBody: prompt, PromptBytes: len(prompt)})
	if len(repo.rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(repo.rows))
	}
	row := repo.rows[0]
	if row.PromptBytes != len(prompt) {
		t.Fatalf("persisted prompt bytes=%d, want original %d", row.PromptBytes, len(prompt))
	}
	if len(row.PromptRedacted) > 96 {
		t.Fatalf("bounded prompt len=%d, want <=96", len(row.PromptRedacted))
	}
}

func TestGenerationContentCollectorPreservesPreviewFromTruncatedLargeJSON(t *testing.T) {
	repo := &generationContentMemoryRepo{}
	cfg := generationCaptureConfig(128, 32)
	svc := &GatewayService{cfg: cfg}
	collector := NewGenerationContentCollector(repo, cfg)
	prompt := []byte(`{"messages":[{"content":"safe prompt prefix ` + strings.Repeat("multimodal filler ", 100) + `"}]}`)
	snapshot := svc.SnapshotGenerationPrompt(prompt)
	collector.Collect(context.Background(), GenerationContentCaptureArgs{
		RequestID: "req-large-preview", PromptBody: snapshot.Body, PromptBytes: snapshot.OriginalBytes,
	})

	if len(repo.rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(repo.rows))
	}
	got := repo.rows[0].PromptRedacted
	if strings.Contains(got, "non-json payload redacted") || !strings.Contains(got, "safe prompt prefix") {
		t.Fatalf("bounded preview was replaced instead of preserved: %q", got)
	}
	if len(got) > 128 {
		t.Fatalf("preview len=%d, want <=128", len(got))
	}
}

func TestGenerationContentCollectorRedactsSecretWhenSnapshotEndsInsideValue(t *testing.T) {
	repo := &generationContentMemoryRepo{}
	cfg := generationCaptureConfig(96, 32)
	svc := &GatewayService{cfg: cfg}
	collector := NewGenerationContentCollector(repo, cfg)
	secretPrefix := "LEAKTOKEN1234567890"
	prompt := []byte(`{"messages":[{"content":"safe"}],"api_key":"` + secretPrefix + strings.Repeat("X9", 100) + `"}`)
	snapshot := svc.SnapshotGenerationPrompt(prompt)
	collector.Collect(context.Background(), GenerationContentCaptureArgs{
		RequestID: "req-truncated-secret", PromptBody: snapshot.Body, PromptBytes: snapshot.OriginalBytes,
	})

	got := repo.rows[0].PromptRedacted
	if strings.Contains(got, secretPrefix) || strings.Contains(got, "non-json payload redacted") {
		t.Fatalf("truncated secret leaked or preview collapsed: %q", got)
	}
	if !strings.Contains(got, "api_key") || (!strings.Contains(got, "***") && !strings.Contains(got, "[已脱敏]")) {
		t.Fatalf("truncated sensitive field lacks redaction evidence: %q", got)
	}
}

func TestGenerationContentCollectorSuppressesLargeBase64AndOpaquePayloads(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
	}{
		{name: "opaque", payload: strings.Repeat("Ab3Cd6Ef9Gh2Jk5Lm8Np1Qr4", 20)},
		{name: "base64", payload: strings.Repeat("QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo0123456789", 20)},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &generationContentMemoryRepo{}
			cfg := generationCaptureConfig(256, 32)
			svc := &GatewayService{cfg: cfg}
			collector := NewGenerationContentCollector(repo, cfg)
			prompt := []byte(`{"messages":[{"content":"safe"}],"image_data":"` + test.payload + `"}`)
			snapshot := svc.SnapshotGenerationPrompt(prompt)
			collector.Collect(context.Background(), GenerationContentCaptureArgs{
				RequestID: "req-large-" + test.name, PromptBody: snapshot.Body, PromptBytes: snapshot.OriginalBytes,
			})

			row := repo.rows[0]
			if strings.Contains(row.PromptRedacted, test.payload[:24]) || !strings.Contains(row.PromptRedacted, "[已脱敏]") {
				t.Fatalf("%s payload not suppressed: %q", test.name, row.PromptRedacted)
			}
			if len(row.PromptRedacted) > 256 || row.PromptBytes != len(prompt) {
				t.Fatalf("%s bounds/bytes mismatch: preview=%d original=%d", test.name, len(row.PromptRedacted), row.PromptBytes)
			}
		})
	}
}

func TestGenerationContentPromptRedactsShortValuesForCommonKeyAliases(t *testing.T) {
	aliases := []string{
		"api_key", "api-key", "apiKey", "apikey",
		"x_api_key", "x-api-key", "xApiKey", "xapikey",
		"client_secret", "client-secret", "clientSecret", "clientsecret",
		"access_token", "access-token", "accessToken", "accesstoken",
		"refresh_token", "refresh-token", "refreshToken", "refreshtoken",
		"id_token", "id-token", "idToken", "idtoken",
		"session_token", "session-token", "sessionToken", "sessiontoken",
		"private_key", "private-key", "privateKey", "privatekey",
		"secret_key", "secret-key", "secretKey", "secretkey",
		"access_key", "access-key", "accessKey", "accesskey",
		"token", "session", "cookie", "set_cookie", "set-cookie", "setCookie",
		"authorization", "bearer", "password", "passwd", "pwd", "secret",
	}
	const shortSecret = "tiny123"
	for _, alias := range aliases {
		for _, mode := range []string{"valid", "truncated_invalid"} {
			t.Run(alias+"/"+mode, func(t *testing.T) {
				body := []byte(`{"prompt":"safe prompt","` + alias + `":"` + shortSecret + `"}`)
				if mode == "truncated_invalid" {
					full := []byte(`{"prompt":"safe prompt","` + alias + `":"` + shortSecret + `","large":"` + strings.Repeat("x", 512) + `"}`)
					svc := &GatewayService{cfg: generationCaptureConfig(128, 32)}
					snapshot := svc.SnapshotGenerationPrompt(full)
					if !snapshot.Truncated {
						t.Fatal("test fixture must truncate after the closed secret value")
					}
					body = snapshot.Body
				}

				out := redactGenerationPrompt(body)
				if strings.Contains(out, shortSecret) {
					t.Fatalf("alias %q leaked short value in %s JSON: %q", alias, mode, out)
				}
				if !strings.Contains(out, "safe prompt") {
					t.Fatalf("alias %q lost safe preview in %s JSON: %q", alias, mode, out)
				}
			})
		}
	}
}

func TestGenerationContentCollectorIsFailOpen(t *testing.T) {
	for _, repo := range []*generationContentMemoryRepo{
		{err: errors.New("database unavailable")},
		{panic: true},
	} {
		collector := NewGenerationContentCollector(repo, generationCaptureConfig(64, 64))
		collector.Collect(context.Background(), GenerationContentCaptureArgs{RequestID: "req-fail-open", PromptBody: []byte(`{"messages":[]}`)})
	}
}

func TestGatewayGenerationContentCaptureDefaultsDisabled(t *testing.T) {
	repo := &generationContentMemoryRepo{}
	svc := &GatewayService{cfg: &config.Config{}, generationCollector: NewGenerationContentCollector(repo, &config.Config{})}
	svc.CollectGenerationContent(context.Background(), GenerationContentCaptureArgs{RequestID: "disabled", PromptBody: []byte(`{}`)})
	if len(repo.rows) != 0 {
		t.Fatalf("disabled capture stored %d rows", len(repo.rows))
	}
}

func TestGenerationContentConfiguredBoundsAreClamped(t *testing.T) {
	collector := NewGenerationContentCollector(&generationContentMemoryRepo{}, generationCaptureConfig(1<<30, 1<<30))
	if got := collector.promptMaxBytes(); got != maxGenerationPromptMaxBytes {
		t.Fatalf("prompt max=%d, want clamp %d", got, maxGenerationPromptMaxBytes)
	}
	if got := collector.responseMaxBytes(); got != maxGenerationResponseMaxBytes {
		t.Fatalf("response max=%d, want clamp %d", got, maxGenerationResponseMaxBytes)
	}
	svc := &GatewayService{cfg: generationCaptureConfig(1<<30, 1<<30)}
	if got := svc.responseCaptureMaxBytes(); got != maxGenerationResponseMaxBytes {
		t.Fatalf("response capture max=%d, want clamp %d", got, maxGenerationResponseMaxBytes)
	}
}
