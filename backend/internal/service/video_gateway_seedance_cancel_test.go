package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSeedanceCancelDeletesUpstreamTask(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotAuth   string
		called    bool
	)
	adapter, acc := newSmokeGatedSeedanceFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"ark-cancel-1","status":"cancelled"}`))
	})

	res, err := adapter.CancelTask(context.Background(), acc, &VideoTask{
		Model:          "doubao-seedance-2-0-260128",
		Duration:       5,
		UpstreamTaskID: "ark-cancel-1",
	})
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	if !called {
		t.Fatal("expected upstream DELETE call")
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method=%q want DELETE", gotMethod)
	}
	if gotPath != "/contents/generations/tasks/ark-cancel-1" {
		t.Fatalf("path=%q want /contents/generations/tasks/ark-cancel-1", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") || !strings.Contains(gotAuth, acc.PlainAPIKey) {
		t.Fatalf("Authorization=%q", gotAuth)
	}
	if res.Status != VideoStatusCancelled {
		t.Fatalf("status=%q want cancelled", res.Status)
	}
	if mode, _ := res.Payload["mode"].(string); mode != "upstream_deleted" {
		t.Fatalf("mode=%q want upstream_deleted; payload=%#v", mode, res.Payload)
	}
}

func TestSeedanceCancelWithoutUpstreamIDIsLocalOnly(t *testing.T) {
	called := false
	adapter, acc := newSmokeGatedSeedanceFixture(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{}`)
	})

	res, err := adapter.CancelTask(context.Background(), acc, &VideoTask{
		Model:    "doubao-seedance-2-0-260128",
		Duration: 5,
	})
	if err != nil {
		t.Fatalf("CancelTask: %v", err)
	}
	if called {
		t.Fatal("must not call upstream when UpstreamTaskID is empty")
	}
	if res.Status != VideoStatusCancelled {
		t.Fatalf("status=%q", res.Status)
	}
	if mode, _ := res.Payload["mode"].(string); mode != "local_cancel_no_upstream_id" {
		t.Fatalf("mode=%q want local_cancel_no_upstream_id", mode)
	}
}

func TestSeedanceCancelMapsAlreadyTerminalUpstream(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		wantMode   string
	}{
		{name: "not found", statusCode: http.StatusNotFound, body: `{"error":{"message":"task not found"}}`, wantMode: "upstream_already_gone"},
		{name: "conflict", statusCode: http.StatusConflict, body: `{"error":{"message":"already completed"}}`, wantMode: "upstream_already_terminal"},
		{name: "deleted status body", statusCode: http.StatusBadRequest, body: `{"status":"deleted"}`, wantMode: "upstream_already_terminal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter, acc := newSmokeGatedSeedanceFixture(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			})
			res, err := adapter.CancelTask(context.Background(), acc, &VideoTask{
				Model:          "doubao-seedance-2-0-260128",
				Duration:       5,
				UpstreamTaskID: "ark-done-1",
			})
			if err != nil {
				t.Fatalf("CancelTask: %v", err)
			}
			if res.Status != VideoStatusCancelled {
				t.Fatalf("status=%q", res.Status)
			}
			if mode, _ := res.Payload["mode"].(string); mode != tc.wantMode {
				t.Fatalf("mode=%q want %q; payload=%#v", mode, tc.wantMode, res.Payload)
			}
		})
	}
}

func TestSeedanceCancelRejectsHardUpstreamError(t *testing.T) {
	adapter, acc := newSmokeGatedSeedanceFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	})
	_, err := adapter.CancelTask(context.Background(), acc, &VideoTask{
		Model:          "doubao-seedance-2-0-260128",
		Duration:       5,
		UpstreamTaskID: "ark-auth-fail",
	})
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cancel") {
		t.Fatalf("error=%q", err.Error())
	}
}
