package handler

import (
	"os"
	"strings"
	"testing"
)

func TestOpenAIGatewayCompletionPointsRegisterGenerationCapture(t *testing.T) {
	tests := []struct {
		file string
		want int
	}{
		{file: "openai_gateway_handler.go", want: 2}, // Responses + Messages (not WebSocket)
		{file: "openai_chat_completions.go", want: 1},
	}
	for _, test := range tests {
		body, err := os.ReadFile(test.file)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(body), ".CollectGenerationContent(ctx,"); got != test.want {
			t.Fatalf("%s capture completion points=%d, want %d", test.file, got, test.want)
		}
		if got := strings.Count(string(body), ".SnapshotGenerationPrompt("); got != test.want {
			t.Fatalf("%s safe prompt snapshots=%d, want %d", test.file, got, test.want)
		}
		if got := strings.Count(string(body), "PromptBytes: generationPrompt.OriginalBytes"); got != test.want {
			t.Fatalf("%s original prompt byte metadata=%d, want %d", test.file, got, test.want)
		}
		if strings.Contains(string(body), "append([]byte(nil),") {
			t.Fatalf("%s directly copies an unbounded prompt body", test.file)
		}
		assertOpenAIGenerationCapturePrecedesUsageRecording(t, test.file, string(body))
		assertOpenAIGenerationCaptureCarriesUsageMetadata(t, test.file, string(body))
	}
}

func TestOpenAIChatCompletionsCaptureRegistersRequestPayloadHash(t *testing.T) {
	body, err := os.ReadFile("openai_chat_completions.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	if !strings.Contains(src, "requestPayloadHash := service.HashUsageRequestPayload(body)") {
		t.Fatal("chat completions success path must hash the request payload for generation-content joins")
	}
	if !strings.Contains(src, "RequestPayloadHash: requestPayloadHash") {
		t.Fatal("chat completions CollectGenerationContent must pass RequestPayloadHash consistent with RecordUsage")
	}
	capture := strings.Index(src, ".CollectGenerationContent(ctx,")
	if capture < 0 {
		t.Fatal("chat completions missing CollectGenerationContent")
	}
	blockEnd := strings.Index(src[capture:], "})")
	if blockEnd < 0 {
		t.Fatal("chat completions CollectGenerationContent block is incomplete")
	}
	block := src[capture : capture+blockEnd]
	if !strings.Contains(block, "RequestPayloadHash: requestPayloadHash") {
		t.Fatal("chat completions Collect args must include RequestPayloadHash")
	}
}

func TestOpenAIResponsesWebSocketSkipsGenerationCapture(t *testing.T) {
	body, err := os.ReadFile("openai_gateway_handler.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	start := strings.Index(src, "func (h *OpenAIGatewayHandler) ResponsesWebSocket(")
	if start < 0 {
		t.Fatal("ResponsesWebSocket handler not found")
	}
	rest := src[start+1:]
	next := strings.Index(rest, "\nfunc (h *OpenAIGatewayHandler) ")
	if next < 0 {
		t.Fatal("unable to bound ResponsesWebSocket function")
	}
	fn := src[start : start+1+next]
	if strings.Contains(fn, "CollectGenerationContent") || strings.Contains(fn, "SnapshotGenerationPrompt") {
		t.Fatal("ResponsesWebSocket must skip generation-content capture")
	}
	// Reject both the unexported install helper and the exported handler entrypoint.
	if strings.Contains(fn, "beginResponseCapture") || strings.Contains(fn, "BeginResponseCapture") {
		t.Fatal("ResponsesWebSocket must not install HTTP response capture")
	}
}

func TestOpenAIEmbeddingsAndImagesSkipGenerationCapture(t *testing.T) {
	for _, file := range []string{"openai_embeddings.go", "openai_images.go"} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		src := string(body)
		if strings.Contains(src, "CollectGenerationContent") || strings.Contains(src, "SnapshotGenerationPrompt") {
			t.Fatalf("%s must skip generation-content capture", file)
		}
		if strings.Contains(src, "beginResponseCapture") || strings.Contains(src, "BeginResponseCapture") {
			t.Fatalf("%s must not install HTTP response capture", file)
		}
	}
}

func TestOpenAICaptureUsesForwardedPromptBody(t *testing.T) {
	tests := []struct {
		file string
		want int
	}{
		{file: "openai_gateway_handler.go", want: 2}, // Responses + Messages
		{file: "openai_chat_completions.go", want: 1},
	}
	for _, test := range tests {
		body, err := os.ReadFile(test.file)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(body), "SnapshotGenerationPrompt(forwardBody)"); got != test.want {
			t.Fatalf("%s forwarded-prompt snapshots=%d, want %d", test.file, got, test.want)
		}
		if strings.Contains(string(body), "SnapshotGenerationPrompt(body)") {
			t.Fatalf("%s must snapshot forwardBody, not raw inbound body", test.file)
		}
	}
}

func assertOpenAIGenerationCapturePrecedesUsageRecording(t *testing.T, file, body string) {
	t.Helper()
	const captureNeedle = ".CollectGenerationContent(ctx,"
	for offset := 0; ; {
		captureRelative := strings.Index(body[offset:], captureNeedle)
		if captureRelative < 0 {
			return
		}
		capture := offset + captureRelative
		worker := strings.LastIndex(body[:capture], "h.submitOpenAIUsageRecordTask")
		if worker < 0 {
			t.Fatalf("%s capture has no bounded OpenAI worker", file)
		}
		if strings.Contains(body[worker:capture], ".RecordUsage(ctx,") || strings.Contains(body[worker:capture], ".RecordUsage(taskCtx,") {
			t.Fatalf("%s generation capture runs after usage recording and can be skipped by its panic/timeout", file)
		}
		offset = capture + len(captureNeedle)
	}
}

func assertOpenAIGenerationCaptureCarriesUsageMetadata(t *testing.T, file, body string) {
	t.Helper()
	const captureNeedle = ".CollectGenerationContent(ctx,"
	required := []string{
		"RequestID:",
		"UserID: subject.UserID",
		"APIKeyID: apiKey.ID",
		"GroupID: apiKey.GroupID",
		"AccountID: account.ID",
		"Model:",
		"RequestPayloadHash: requestPayloadHash",
	}
	for offset := 0; ; {
		captureRelative := strings.Index(body[offset:], captureNeedle)
		if captureRelative < 0 {
			return
		}
		capture := offset + captureRelative
		blockEndRel := strings.Index(body[capture:], "})")
		if blockEndRel < 0 {
			t.Fatalf("%s CollectGenerationContent block is incomplete", file)
		}
		block := body[capture : capture+blockEndRel]
		for _, needle := range required {
			if !strings.Contains(block, needle) {
				t.Fatalf("%s CollectGenerationContent missing metadata %q", file, needle)
			}
		}
		offset = capture + len(captureNeedle)
	}
}
