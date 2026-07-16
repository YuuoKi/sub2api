package migrations

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestVideoGatewayRuntimeMigrationIsAdditiveAndComplete(t *testing.T) {
	b, err := os.ReadFile("175_wujie_video_gateway_runtime_contract.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(b))
	for _, required := range []string{"api_key_id", "group_id", "duration_seconds", "resolution", "last_frame_url", "usage_total_tokens", "cost_amount", "currency", "real_dispatch_count", "provider_error_code", "provider_error_message"} {
		if !strings.Contains(sql, required) {
			t.Errorf("missing %s", required)
		}
	}
	destructive := regexp.MustCompile(`(?m)^\s*(drop|truncate|delete)\b|insert\s+into[\s\S]+select\s`)
	if destructive.MatchString(sql) {
		t.Fatal("runtime migration must be additive and must not rewrite history")
	}
}

func TestVideoGatewayControlPlaneMigrationIsAdditiveAndComplete(t *testing.T) {
	b, err := os.ReadFile("176_wujie_video_gateway_control_plane.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(b))
	for _, required := range []string{"video_single_smoke_consumptions", "gate_key", "video_task_id", "group_id"} {
		if !strings.Contains(sql, required) {
			t.Errorf("missing %s", required)
		}
	}
	if regexp.MustCompile(`(?m)^\s*(drop|truncate|delete)\b|insert\s+into[\s\S]+select\s`).MatchString(sql) {
		t.Fatal("control plane migration must be additive")
	}
}
