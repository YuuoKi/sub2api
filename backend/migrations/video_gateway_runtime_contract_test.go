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

func TestVideoGatewayReservationMigrationIsAdditiveAndComplete(t *testing.T) {
	b, err := os.ReadFile("177_wujie_video_gateway_reservations.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(b))
	for _, required := range []string{
		"reserved_cost_usd", "reservation_state", "reservation_window_5h_start",
		"reservation_window_1d_start", "reservation_window_7d_start", "provider_actual_cost_usd",
		"completion_tokens", "charged_cost_usd",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("missing %s", required)
		}
	}
	if regexp.MustCompile(`(?m)^\s*(drop|truncate|delete)\b|insert\s+into[\s\S]+select\s`).MatchString(sql) {
		t.Fatal("reservation migration must be additive")
	}
}

func TestVideoGatewayProviderAuthorizationAndCanonicalContract(t *testing.T) {
	authorization, err := os.ReadFile("178_wujie_video_admin_control_plane.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"tiny_real_authorized_at", "tiny_real_authorized_by", "tiny_real_consumed_at"} {
		if !strings.Contains(strings.ToLower(string(authorization)), required) {
			t.Errorf("missing %s", required)
		}
	}
	contract, err := os.ReadFile("179_wujie_video_provider_contract.sql")
	if err != nil {
		t.Fatal(err)
	}
	contractSQL := strings.ToLower(string(contract))
	for _, required := range []string{"doubao-seedance-2-0-260128", "https://ark.cn-beijing.volces.com/api/v3"} {
		if !strings.Contains(contractSQL, required) {
			t.Errorf("missing %s", required)
		}
	}
	if regexp.MustCompile(`(?m)^\s*(update|delete|drop|truncate|create\s+unique\s+index)\b`).MatchString(contractSQL) {
		t.Fatal("provider contract migration must be additive-only and must not rewrite or reject historical rows")
	}
}

func TestVideoTaskEvidenceMigrationIsAdditiveAndKeepsUnknownProviderFactsNullable(t *testing.T) {
	b, err := os.ReadFile("181_video_task_delivery_evidence.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(b))
	for _, required := range []string{
		"upstream_model", "upstream_duration_seconds", "upstream_resolution",
		"billing_model", "billing_duration_seconds", "billing_resolution",
		"balance_before_usd", "balance_after_usd", "balance_delta_usd",
		"authorization_consumed_at", "authorization_consumed_by",
	} {
		if !strings.Contains(sql, required) {
			t.Errorf("missing %s", required)
		}
	}
	for _, nullable := range []string{
		"upstream_model", "upstream_duration_seconds", "upstream_resolution",
		"billing_model", "billing_duration_seconds", "billing_resolution",
	} {
		if regexp.MustCompile(nullable + `[^,;]*not\s+null`).MatchString(sql) {
			t.Errorf("%s must remain nullable when the provider omits it", nullable)
		}
	}
	if regexp.MustCompile(`(?m)^\s*(update|delete|drop|truncate)\b|insert\s+into[\s\S]+select\s`).MatchString(sql) {
		t.Fatal("video task evidence migration must be additive and must not rewrite history")
	}
}
