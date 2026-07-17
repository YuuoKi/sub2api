package service

// cappedSink is a fail-open, size-capped capture buffer. It always reports the
// full write length while retaining at most max bytes in its independently
// owned buffer.
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
	capacity := max
	if capacity > 4096 {
		capacity = 4096
	}
	return &cappedSink{max: max, buf: make([]byte, 0, capacity)}
}

func (s *cappedSink) Write(p []byte) (int, error) {
	if s == nil {
		return len(p), nil
	}
	s.total += len(p)
	remaining := s.max - len(s.buf)
	if remaining <= 0 {
		if len(p) > 0 {
			s.truncated = true
		}
		return len(p), nil
	}
	take := len(p)
	if take > remaining {
		take = remaining
		s.truncated = true
	}
	s.buf = append(s.buf, p[:take]...)
	return len(p), nil
}

func (s *cappedSink) Bytes() []byte {
	if s == nil {
		return nil
	}
	return s.buf
}

func (s *cappedSink) Truncated() bool {
	return s != nil && s.truncated
}

func (s *cappedSink) Total() int {
	if s == nil {
		return 0
	}
	return s.total
}
