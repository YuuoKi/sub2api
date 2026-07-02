package service

import "testing"

func TestCappedSinkCapsAndCounts(t *testing.T) {
	s := newCappedSink(10)
	if n, err := s.Write([]byte("hello")); err != nil || n != 5 {
		t.Fatalf("first write: n=%d err=%v", n, err)
	}
	// 7 more bytes; only 5 fit (cap 10), rest dropped.
	if n, err := s.Write([]byte("world!!")); err != nil || n != 7 {
		t.Fatalf("second write must report full length (fail-open): n=%d err=%v", n, err)
	}
	if got := string(s.Bytes()); got != "helloworld" {
		t.Fatalf("buf = %q, want %q", got, "helloworld")
	}
	if !s.Truncated() {
		t.Fatalf("expected truncated=true after exceeding cap")
	}
	if s.Total() != 12 {
		t.Fatalf("total = %d, want 12", s.Total())
	}
}

func TestCappedSinkOwnsBuffer(t *testing.T) {
	p := []byte("abc")
	s := newCappedSink(100)
	if n, err := s.Write(p); err != nil || n != len(p) {
		t.Fatalf("write before caller mutation: n=%d err=%v", n, err)
	}
	p[0] = 'X' // mutate caller's slice after write
	if got := string(s.Bytes()); got != "abc" {
		t.Fatalf("sink buffer aliased caller slice: got %q", got)
	}
}

func TestCappedSinkZeroAndNilSafe(t *testing.T) {
	z := newCappedSink(0)
	if n, err := z.Write([]byte("data")); err != nil || n != 4 {
		t.Fatalf("zero-cap write: n=%d err=%v", n, err)
	}
	if len(z.Bytes()) != 0 || !z.Truncated() || z.Total() != 4 {
		t.Fatalf("zero-cap sink: bytes=%d truncated=%v total=%d", len(z.Bytes()), z.Truncated(), z.Total())
	}
	var nilSink *cappedSink
	if n, err := nilSink.Write([]byte("x")); err != nil || n != 1 {
		t.Fatalf("nil sink write must be safe: n=%d err=%v", n, err)
	}
	if nilSink.Bytes() != nil || nilSink.Truncated() || nilSink.Total() != 0 {
		t.Fatalf("nil sink accessors must be zero-safe")
	}
}
