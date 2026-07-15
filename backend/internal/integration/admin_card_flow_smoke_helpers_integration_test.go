//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func smokeBackendRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("backend root with go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func smokeSecret(t *testing.T, byteLength int) string {
	t.Helper()
	buf := make([]byte, byteLength)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate isolated test secret: %v", err)
	}
	return hex.EncodeToString(buf)
}

func smokeContainerHost(t *testing.T, ctx context.Context, host func(context.Context) (string, error)) string {
	t.Helper()
	value, err := host(ctx)
	if err != nil {
		t.Fatalf("resolve temporary container host: %v", err)
	}
	return value
}

func smokeContainerPort(t *testing.T, ctx context.Context, name string, port func(context.Context) (string, error)) string {
	t.Helper()
	value, err := port(ctx)
	if err != nil {
		t.Fatalf("resolve temporary %s port: %v", name, err)
	}
	return value
}

func smokeOpenAIUpstream(t *testing.T, providerKey string, pricingBody []byte, pricingHash string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pricing.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(pricingBody)
		case "/pricing.sha256":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, pricingHash+"\n")
		case "/v1/chat/completions":
			if r.Header.Get("Authorization") != "Bearer "+providerKey {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"chatcmpl-smoke","object":"chat.completion","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"local smoke ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1000,"completion_tokens":500,"total_tokens":1500}}`)
		case "/v1/responses":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":{"message":"responses endpoint intentionally absent in local smoke"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func smokeStartCurrentServer(t *testing.T, backendRoot string, env map[string]string, secrets []string) smokeServer {
	t.Helper()
	buildDir := t.TempDir()
	binary := filepath.Join(buildDir, "sub2api-smoke")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/server")
	build.Dir = backendRoot
	build.Env = smokeSubprocessEnv(nil)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build current server source: %v: %s", err, smokeSanitize(string(output), secrets))
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback server port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release loopback server port: %v", err)
	}
	env["SERVER_PORT"] = strconv.Itoa(port)

	cmd := exec.Command(binary)
	cmd.Dir = t.TempDir()
	cmd.Env = smokeSubprocessEnv(env)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start current server source: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			if cmd.Process != nil {
				if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
					t.Errorf("stop temporary server process: %v", err)
				}
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("temporary server process did not exit within cleanup timeout")
			}
		})
	}
	t.Cleanup(stop)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			stopOnce.Do(func() {})
			t.Fatalf("server exited before readiness: %v: %s", err, smokeSanitize(logs.String(), secrets))
		default:
		}
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return smokeServer{baseURL: baseURL, stop: stop}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	stop()
	t.Fatalf("server did not become ready: %s", smokeSanitize(logs.String(), secrets))
	return smokeServer{}
}

func smokeSubprocessEnv(overrides map[string]string) []string {
	blocked := map[string]struct{}{
		"ANTHROPIC_API_KEY": {},
		"CLAUDE_API_KEY":    {},
		"DATABASE_URL":      {},
		"GEMINI_API_KEY":    {},
		"GOOGLE_API_KEY":    {},
		"GROK_API_KEY":      {},
		"HTTP_PROXY":        {},
		"HTTPS_PROXY":       {},
		"ALL_PROXY":         {},
		"NO_PROXY":          {},
		"OPENAI_API_KEY":    {},
		"REDIS_URL":         {},
		"XAI_API_KEY":       {},
	}
	result := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(key)
		if _, denied := blocked[upper]; denied {
			continue
		}
		if _, replaced := overrides[key]; replaced {
			continue
		}
		if _, replaced := overrides[upper]; replaced {
			continue
		}
		result = append(result, entry)
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

func smokeLogin(t *testing.T, client *http.Client, baseURL, email, password string) string {
	t.Helper()
	data := smokeAPIObject(t, client, http.MethodPost, baseURL+"/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": password,
	}, "login")
	return smokeString(t, data, "access_token")
}

func smokeAPIObject(t *testing.T, client *http.Client, method, url, token string, payload any, label string) map[string]any {
	t.Helper()
	body := smokeJSONBody(t, payload)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("%s request: %v", label, err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s transport: %v", label, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		t.Fatalf("%s response read: %v", label, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("%s: status=%d message=%q", label, resp.StatusCode, smokeResponseMessage(responseBody))
	}
	var envelope smokeEnvelope
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		t.Fatalf("%s response envelope: %v", label, err)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return map[string]any{}
	}
	decoder := json.NewDecoder(bytes.NewReader(envelope.Data))
	decoder.UseNumber()
	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		t.Fatalf("%s response data: %v", label, err)
	}
	return data
}

func smokeJSONBody(t *testing.T, payload any) io.Reader {
	t.Helper()
	if payload == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal smoke request: %v", err)
	}
	return bytes.NewReader(data)
}

func smokeGatewayCall(t *testing.T, client *http.Client, baseURL, apiKey string) {
	t.Helper()
	body := smokeJSONBody(t, map[string]any{
		"model": smokeModel,
		"messages": []map[string]string{
			{"role": "user", "content": "reply with local smoke ok"},
		},
		"stream": false,
	})
	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", body)
	if err != nil {
		t.Fatalf("gateway request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("gateway transport: %v", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		t.Fatalf("gateway response read: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("gateway call: status=%d message=%q", resp.StatusCode, smokeResponseMessage(responseBody))
	}
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		t.Fatalf("gateway response json: %v", err)
	}
	if id, _ := payload["id"].(string); id == "" {
		t.Fatal("gateway response missing id")
	}
}

func smokeUsageTotal(t *testing.T, client *http.Client, baseURL, userToken string) int64 {
	t.Helper()
	data := smokeAPIObject(t, client, http.MethodGet, baseURL+"/api/v1/usage?page=1&page_size=20", userToken, nil, "read user usage")
	if total, ok := smokeNumber(data["total"]); ok {
		return int64(total)
	}
	if pagination, ok := data["pagination"].(map[string]any); ok {
		if total, ok := smokeNumber(pagination["total"]); ok {
			return int64(total)
		}
	}
	t.Fatal("usage response missing total")
	return 0
}

func smokeID(t *testing.T, data map[string]any, label string) int64 {
	t.Helper()
	value, ok := smokeNumber(data["id"])
	if !ok || value <= 0 {
		t.Fatalf("%s response missing positive id", label)
	}
	return int64(value)
}

func smokeFloat(t *testing.T, data map[string]any, key string) float64 {
	t.Helper()
	value, ok := smokeNumber(data[key])
	if !ok {
		t.Fatalf("response missing numeric %s", key)
	}
	return value
}

func smokeNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func smokeString(t *testing.T, data map[string]any, key string) string {
	t.Helper()
	value, ok := data[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		t.Fatalf("response missing %s", key)
	}
	return value
}

func smokeResponseMessage(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "non-json response withheld"
	}
	if message, ok := payload["message"].(string); ok && message != "" {
		return message
	}
	if nested, ok := payload["error"].(map[string]any); ok {
		if message, ok := nested["message"].(string); ok && message != "" {
			return message
		}
	}
	return "response details withheld"
}

func smokeMask(secret string) string {
	if len(secret) < 10 {
		return "[masked]"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}

func smokeSanitize(value string, secrets []string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[redacted]")
		}
	}
	value = strings.TrimSpace(value)
	if len(value) > 4000 {
		return value[len(value)-4000:]
	}
	return value
}
