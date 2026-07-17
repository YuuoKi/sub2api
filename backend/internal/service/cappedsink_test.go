package service

import "testing"

func TestCappedSinkCapsCountsAndOwnsBytes(t *testing.T) {
	sink := newCappedSink(5)
	first := []byte("abc")
	if n, err := sink.Write(first); err != nil || n != len(first) {
		t.Fatalf("first write n=%d err=%v", n, err)
	}
	first[0] = 'X'
	if n, err := sink.Write([]byte("defg")); err != nil || n != 4 {
		t.Fatalf("second write n=%d err=%v", n, err)
	}
	if got := string(sink.Bytes()); got != "abcde" {
		t.Fatalf("captured=%q, want abcde", got)
	}
	if sink.Total() != 7 || !sink.Truncated() {
		t.Fatalf("total=%d truncated=%v", sink.Total(), sink.Truncated())
	}
}
