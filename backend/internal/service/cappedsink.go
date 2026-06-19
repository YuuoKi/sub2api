package service

// cappedSink is a fail-open, size-capped capture buffer used to sample the
// client-facing response stream for the M1 采集口 (content collector) without
// risking the relay hot path.
//
// Contract (判断基准4 — never break/slow the main route):
//   - Write never returns an error and never allocates beyond max bytes.
//   - Once full it drops further bytes but keeps counting Total (so the caller
//     can record the true response size and a truncation flag).
//   - The internal buffer is independently owned: Write copies input bytes, so
//     it is safe to tee a pooled/reused buffer (e.g. an SSE scanner buffer) into
//     it without use-after-return corruption.
//
// It implements io.Writer so it can be composed with io.MultiWriter when teeing.
type cappedSink struct {
	buf       []byte
	max       int
	total     int
	truncated bool
}

func newCappedSink(max int) *cappedSink {
	if max < 0 {
		max = 0
	}
	initial := max
	if initial > 4096 {
		initial = 4096
	}
	return &cappedSink{max: max, buf: make([]byte, 0, initial)}
}

// Write copies up to the remaining capacity from p into the owned buffer and
// always reports the full length as written (io.Writer contract, fail-open).
func (s *cappedSink) Write(p []byte) (int, error) {
	if s == nil {
		return len(p), nil
	}
	s.total += len(p)
	remaining := s.max - len(s.buf)
	if remaining > 0 {
		take := len(p)
		if take > remaining {
			take = remaining
			s.truncated = true
		}
		s.buf = append(s.buf, p[:take]...)
	} else if len(p) > 0 {
		s.truncated = true
	}
	return len(p), nil
}

// Bytes returns the captured (capped) sample. The slice is owned by the sink.
func (s *cappedSink) Bytes() []byte {
	if s == nil {
		return nil
	}
	return s.buf
}

// Truncated reports whether any bytes were dropped because the cap was hit.
func (s *cappedSink) Truncated() bool {
	if s == nil {
		return false
	}
	return s.truncated
}

// Total reports the total number of bytes seen (including dropped ones).
func (s *cappedSink) Total() int {
	if s == nil {
		return 0
	}
	return s.total
}
