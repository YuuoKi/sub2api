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
	return installResponseCapture(c, s.responseCaptureMaxBytes())
}

func (s *GatewayService) responseCaptureMaxBytes() int {
	if s != nil {
		return generationResponseCaptureMaxBytes(s.cfg)
	}
	return defaultGenerationResponseMaxBytes
}

func fillResponseSample(result *ForwardResult, sink *cappedSink) {
	if result == nil {
		return
	}
	fillCappedResponseSample(&result.ResponseSample, &result.ResponseBytes, &result.ResponseTruncated, sink)
}
