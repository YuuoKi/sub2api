package service

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

func TestCapturingResponseWriterCapsCopyWithoutChangingClientBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	sink := newCappedSink(8)
	w := &capturingResponseWriter{ResponseWriter: c.Writer, sink: sink}
	payload := strings.Repeat("A", 20)

	if _, err := io.WriteString(w, payload); err != nil {
		t.Fatal(err)
	}
	if recorder.Body.String() != payload {
		t.Fatalf("client body changed: %q", recorder.Body.String())
	}
	if got := string(sink.Bytes()); got != strings.Repeat("A", 8) || sink.Total() != 20 || !sink.Truncated() {
		t.Fatalf("capture got=%q total=%d truncated=%v", got, sink.Total(), sink.Truncated())
	}
}

func TestBeginResponseCaptureDisabledDoesNotReplaceWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &GatewayService{cfg: &config.Config{}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	original := c.Writer

	sink, restore := svc.beginResponseCapture(c)
	defer restore()
	if sink != nil || c.Writer != original {
		t.Fatal("disabled capture must not allocate or replace writer")
	}
}

func TestCapturingResponseWriterPreservesJSONAndSSEProtocolSignals(t *testing.T) {
	for _, test := range []struct {
		name        string
		contentType string
		status      int
		payload     string
		flush       bool
	}{
		{name: "json", contentType: "application/json", status: 201, payload: `{"ok":true}`},
		{name: "sse", contentType: "text/event-stream", status: 200, payload: "event: message\ndata: {\"delta\":\"hi\"}\n\n", flush: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			svc := &GatewayService{cfg: generationCaptureConfig(128, 128)}
			sink, restore := svc.beginResponseCapture(c)
			defer restore()

			c.Header("Content-Type", test.contentType)
			c.Status(test.status)
			if _, err := c.Writer.WriteString(test.payload); err != nil {
				t.Fatal(err)
			}
			if test.flush {
				c.Writer.Flush()
			}

			if recorder.Code != test.status || recorder.Header().Get("Content-Type") != test.contentType {
				t.Fatalf("protocol changed: status=%d content-type=%q", recorder.Code, recorder.Header().Get("Content-Type"))
			}
			if recorder.Body.String() != test.payload {
				t.Fatalf("client bytes changed: %q", recorder.Body.String())
			}
			if test.flush && !recorder.Flushed {
				t.Fatal("SSE flush was not delegated")
			}
			result := &ForwardResult{}
			fillResponseSample(result, sink)
			if string(result.ResponseSample) != test.payload || result.ResponseBytes != len(test.payload) || result.ResponseTruncated {
				t.Fatalf("capture evidence mismatch: %+v", result)
			}
		})
	}
}
