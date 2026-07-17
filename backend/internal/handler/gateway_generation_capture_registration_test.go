package handler

import (
	"os"
	"strings"
	"testing"
)

func TestAnthropicGatewayCompletionPointsRegisterGenerationCapture(t *testing.T) {
	tests := []struct {
		file string
		want int
	}{
		{file: "gateway_handler.go", want: 2},
		{file: "gateway_handler_chat_completions.go", want: 1},
		{file: "gateway_handler_responses.go", want: 1},
	}
	for _, test := range tests {
		body, err := os.ReadFile(test.file)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(body), ".CollectGenerationContent(ctx,"); got != test.want {
			t.Fatalf("%s capture completion points=%d, want %d", test.file, got, test.want)
		}
		assertGenerationCapturePrecedesUsageRecording(t, test.file, string(body))
	}
}

func TestAnthropicFallbackCaptureUsesForwardedAttemptBody(t *testing.T) {
	body, err := os.ReadFile("gateway_handler.go")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(body), "generationPromptBody := append([]byte(nil), attemptParsedReq.Body.Bytes()...)"); got != 1 {
		t.Fatalf("fallback capture forwarded-attempt prompt references=%d, want 1", got)
	}
}

func assertGenerationCapturePrecedesUsageRecording(t *testing.T, file, body string) {
	t.Helper()
	const captureNeedle = ".CollectGenerationContent(ctx,"
	for offset := 0; ; {
		captureRelative := strings.Index(body[offset:], captureNeedle)
		if captureRelative < 0 {
			return
		}
		capture := offset + captureRelative
		worker := strings.LastIndex(body[:capture], "h.submitUsageRecordTask")
		if worker < 0 {
			t.Fatalf("%s capture has no bounded worker", file)
		}
		if strings.Contains(body[worker:capture], ".RecordUsage(ctx,") {
			t.Fatalf("%s generation capture runs after usage recording and can be skipped by its panic/timeout", file)
		}
		offset = capture + len(captureNeedle)
	}
}
