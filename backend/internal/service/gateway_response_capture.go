package service

import "github.com/gin-gonic/gin"

type capturingResponseWriter struct {
	gin.ResponseWriter
	sink *cappedSink
}

func (w *capturingResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		_, _ = w.sink.Write(p[:n])
	}
	return n, err
}

func (w *capturingResponseWriter) WriteString(value string) (int, error) {
	n, err := w.ResponseWriter.WriteString(value)
	if n > 0 {
		_, _ = w.sink.Write([]byte(value[:n]))
	}
	return n, err
}

func (s *GatewayService) beginResponseCapture(c *gin.Context) (*cappedSink, func()) {
	if c == nil || !s.contentCaptureEnabled() {
		return nil, func() {}
	}
	sink := newCappedSink(s.responseCaptureMaxBytes())
	original := c.Writer
	c.Writer = &capturingResponseWriter{ResponseWriter: original, sink: sink}
	return sink, func() { c.Writer = original }
}

func (s *GatewayService) responseCaptureMaxBytes() int {
	if s != nil && s.cfg != nil {
		return boundedGenerationBytes(s.cfg.Gateway.ContentCapture.ResponseMaxBytes, defaultGenerationResponseMaxBytes, maxGenerationResponseMaxBytes)
	}
	return defaultGenerationResponseMaxBytes
}

func fillResponseSample(result *ForwardResult, sink *cappedSink) {
	if result == nil || sink == nil {
		return
	}
	result.ResponseSample = append([]byte(nil), sink.Bytes()...)
	result.ResponseTruncated = sink.Truncated()
	result.ResponseBytes = sink.Total()
}
