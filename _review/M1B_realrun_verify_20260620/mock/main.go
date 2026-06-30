// Throwaway deterministic mock upstream for M1-B BC verification.
// Listens on 127.0.0.1:9099, mimics Anthropic /v1/messages.
// EVERY response is byte-fixed (fixed Date, fixed x-request-id, no random/timestamp)
// so golden==after is byte-comparable. Logs each received request to stderr for P0.
// NOT product code; lives outside the backend/ Go module. ¥0, local-only.
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

const fixedDate = "Fri, 20 Jun 2026 00:00:00 GMT"
const fixedReqID = "mock-req-1"

// Fixed non-streaming Anthropic message JSON. Contains MOCK_REPLY + a phone number
// (exercises response redaction -> [PHONE]) and a usage block (billing path).
const nonStreamBody = `{"id":"msg_mock_0001","type":"message","role":"assistant","model":"claude-3-5-sonnet-20241022","content":[{"type":"text","text":"MOCK_REPLY 工单已收到,我们会尽快回电 +86-10-87654321 处理。"}],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":12,"output_tokens":18}}`

// Fixed SSE sequence (Anthropic style). Each event: "event: X\ndata: {...}\n\n".
var sseEvents = []string{
	`event: message_start
data: {"type":"message_start","message":{"id":"msg_mock_0001","type":"message","role":"assistant","model":"claude-3-5-sonnet-20241022","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":12,"output_tokens":0}}}`,
	`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
	`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"MOCK_REPLY 回电 +86-10-87654321"}}`,
	`event: content_block_stop
data: {"type":"content_block_stop","index":0}`,
	`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":18}}`,
	`event: message_stop
data: {"type":"message_stop"}`,
}

var counter int64

func handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	isStream := bytes.Contains(body, []byte(`"stream":true`)) || bytes.Contains(body, []byte(`"stream": true`))
	n := atomic.AddInt64(&counter, 1)
	fmt.Fprintf(os.Stderr, "[mock] #%d %s %s stream=%v bodylen=%d\n", n, r.Method, r.URL.String(), isStream, len(body))

	// Fixed deterministic headers on every response.
	h := w.Header()
	h.Set("Date", fixedDate)        // explicit -> Go won't add wall-clock Date
	h.Set("x-request-id", fixedReqID)

	if isStream {
		h.Set("Content-Type", "text/event-stream; charset=utf-8")
		h.Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for _, ev := range sseEvents {
			io.WriteString(w, ev+"\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
		return
	}
	h.Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, nonStreamBody)
}

func main() {
	addr := "127.0.0.1:9099"
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], ":") {
		addr = "127.0.0.1" + os.Args[1]
	}
	http.HandleFunc("/", handler)
	fmt.Fprintf(os.Stderr, "[mock] listening on %s (fixed Date=%q, x-request-id=%q)\n", addr, fixedDate, fixedReqID)
	log.Fatal(http.ListenAndServe(addr, nil))
}
