//go:build integration

package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

const (
	smokePostgresImage = "postgres:18.1-alpine3.23"
	smokeRedisImage    = "redis:8.4-alpine"
	smokeModel         = "gpt-4o-mini"
)

type smokeEnvelope struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type smokeServer struct {
	baseURL string
	stop    func()
}

func TestAdminProvisioningGatewayUsageSmoke(t *testing.T) {
	backendRoot := smokeBackendRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	postgresPassword := smokeSecret(t, 16)
	postgresContainer, err := tcpostgres.Run(
		ctx,
		smokePostgresImage,
		tcpostgres.WithDatabase("sub2api_smoke"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword(postgresPassword),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start isolated postgres: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := postgresContainer.Terminate(cleanupCtx); err != nil {
			t.Errorf("temporary postgres cleanup: %v", err)
		}
	})

	redisContainer, err := tcredis.Run(ctx, smokeRedisImage)
	if err != nil {
		t.Fatalf("start isolated redis: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := redisContainer.Terminate(cleanupCtx); err != nil {
			t.Errorf("temporary redis cleanup: %v", err)
		}
	})

	postgresHost := smokeContainerHost(t, ctx, postgresContainer.Host)
	postgresPort := smokeContainerPort(t, ctx, "postgres", func(ctx context.Context) (string, error) {
		port, err := postgresContainer.MappedPort(ctx, "5432/tcp")
		return port.Port(), err
	})
	redisHost := smokeContainerHost(t, ctx, redisContainer.Host)
	redisPort := smokeContainerPort(t, ctx, "redis", func(ctx context.Context) (string, error) {
		port, err := redisContainer.MappedPort(ctx, "6379/tcp")
		return port.Port(), err
	})

	pricingBody, err := os.ReadFile(filepath.Join(backendRoot, "resources", "model-pricing", "model_prices_and_context_window.json"))
	if err != nil {
		t.Fatalf("read repository pricing fallback: %v", err)
	}
	pricingHash := sha256.Sum256(pricingBody)
	providerKey := smokeSecret(t, 24)
	upstream := smokeOpenAIUpstream(t, providerKey, pricingBody, hex.EncodeToString(pricingHash[:]))

	adminEmail := fmt.Sprintf("admin-smoke-%d@test.local", time.Now().UnixNano())
	adminPassword := smokeSecret(t, 24)
	runtimeDir := t.TempDir()
	server := smokeStartCurrentServer(t, backendRoot, map[string]string{
		"AUTO_SETUP":            "true",
		"DATA_DIR":              runtimeDir,
		"DATABASE_HOST":         postgresHost,
		"DATABASE_PORT":         postgresPort,
		"DATABASE_USER":         "postgres",
		"DATABASE_PASSWORD":     postgresPassword,
		"DATABASE_DBNAME":       "sub2api_smoke",
		"DATABASE_SSLMODE":      "disable",
		"REDIS_HOST":            redisHost,
		"REDIS_PORT":            redisPort,
		"REDIS_PASSWORD":        "",
		"REDIS_DB":              "0",
		"REDIS_ENABLE_TLS":      "false",
		"ADMIN_EMAIL":           adminEmail,
		"ADMIN_PASSWORD":        adminPassword,
		"JWT_SECRET":            smokeSecret(t, 32),
		"TOTP_ENCRYPTION_KEY":   smokeSecret(t, 32),
		"RUN_MODE":              "standard",
		"SERVER_HOST":           "127.0.0.1",
		"SERVER_MODE":           "release",
		"TZ":                    "UTC",
		"PRICING_REMOTE_URL":    upstream.URL + "/pricing.json",
		"PRICING_HASH_URL":      upstream.URL + "/pricing.sha256",
		"PRICING_DATA_DIR":      filepath.Join(runtimeDir, "pricing"),
		"PRICING_FALLBACK_FILE": filepath.Join(backendRoot, "resources", "model-pricing", "model_prices_and_context_window.json"),
	}, []string{adminPassword, postgresPassword, providerKey})
	defer server.stop()

	client := &http.Client{Timeout: 15 * time.Second}
	adminToken := smokeLogin(t, client, server.baseURL, adminEmail, adminPassword)
	smokeAPIObject(t, client, http.MethodPost, server.baseURL+"/api/v1/admin/compliance/accept", adminToken, map[string]any{
		"phrase":   service.AdminComplianceAckPhraseEN,
		"language": "en",
	}, "accept admin compliance")

	group := smokeAPIObject(t, client, http.MethodPost, server.baseURL+"/api/v1/admin/groups", adminToken, map[string]any{
		"name":              fmt.Sprintf("smoke-openai-%d", time.Now().UnixNano()),
		"description":       "isolated local gateway smoke",
		"platform":          "openai",
		"rate_multiplier":   1.0,
		"subscription_type": "standard",
	}, "create openai group")
	groupID := smokeID(t, group, "group")

	userEmail := fmt.Sprintf("user-smoke-%d@test.local", time.Now().UnixNano())
	userPassword := smokeSecret(t, 24)
	zeroBalance := 0.0
	user := smokeAPIObject(t, client, http.MethodPost, server.baseURL+"/api/v1/admin/users", adminToken, map[string]any{
		"email":          userEmail,
		"password":       userPassword,
		"username":       "smoke-user",
		"role":           "user",
		"balance":        zeroBalance,
		"concurrency":    1,
		"allowed_groups": []int64{groupID},
	}, "admin create user")
	userID := smokeID(t, user, "user")

	smokeAPIObject(t, client, http.MethodPost, server.baseURL+"/api/v1/admin/redeem-codes/create-and-redeem", adminToken, map[string]any{
		"code":    "SMOKE-" + strings.ToUpper(smokeSecret(t, 8)),
		"type":    "balance",
		"value":   10.0,
		"user_id": userID,
		"notes":   "isolated integration smoke",
	}, "create and redeem balance code")

	userBefore := smokeAPIObject(t, client, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/users/%d", server.baseURL, userID), adminToken, nil, "read balance before gateway")
	balanceBefore := smokeFloat(t, userBefore, "balance")
	if balanceBefore <= 0 {
		t.Fatalf("redeemed balance must be positive, got %.8f", balanceBefore)
	}

	userToken := smokeLogin(t, client, server.baseURL, userEmail, userPassword)
	apiKeyData := smokeAPIObject(t, client, http.MethodPost, server.baseURL+"/api/v1/keys", userToken, map[string]any{
		"name":     "smoke-gateway-key",
		"group_id": groupID,
	}, "user create api key")
	apiKey := smokeString(t, apiKeyData, "key")

	smokeAPIObject(t, client, http.MethodPost, server.baseURL+"/api/v1/admin/accounts", adminToken, map[string]any{
		"name":     "smoke-loopback-openai",
		"platform": "openai",
		"type":     "apikey",
		"credentials": map[string]any{
			"api_key":       providerKey,
			"base_url":      upstream.URL,
			"model_mapping": map[string]any{smokeModel: smokeModel},
		},
		"extra":                      map[string]any{},
		"concurrency":                2,
		"priority":                   1,
		"group_ids":                  []int64{groupID},
		"confirm_mixed_channel_risk": true,
	}, "create loopback provider account")

	usageBefore := smokeUsageTotal(t, client, server.baseURL, userToken)
	smokeGatewayCall(t, client, server.baseURL, apiKey)

	deadline := time.Now().Add(15 * time.Second)
	var usageAfter int64
	var balanceAfter float64
	for time.Now().Before(deadline) {
		usageAfter = smokeUsageTotal(t, client, server.baseURL, userToken)
		userAfter := smokeAPIObject(t, client, http.MethodGet, fmt.Sprintf("%s/api/v1/admin/users/%d", server.baseURL, userID), adminToken, nil, "poll balance after gateway")
		balanceAfter = smokeFloat(t, userAfter, "balance")
		if usageAfter > usageBefore && balanceAfter < balanceBefore {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if usageAfter <= usageBefore {
		t.Fatalf("usage did not increase: before=%d after=%d", usageBefore, usageAfter)
	}
	if balanceAfter >= balanceBefore {
		t.Fatalf("balance did not decrease: before=%.8f after=%.8f", balanceBefore, balanceAfter)
	}

	t.Logf(
		"admin card flow verified: user_id=%d group_id=%d api_key=%s usage_delta=%d balance_delta=%.8f",
		userID,
		groupID,
		smokeMask(apiKey),
		usageAfter-usageBefore,
		balanceBefore-balanceAfter,
	)
}
