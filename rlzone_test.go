package rlzone

import (
	"testing"
	"time"
)

func TestRatelimitZone(t *testing.T) {
	const limit = 20
	z := New[string](time.Second, limit)
	for i := 0; i < 40; i++ {
		if !z.Allow("user1") {
			t.Fatalf("unexpected deny on iteration %d", i)
		}
		time.Sleep(100 * time.Millisecond)
	}

	if z.Allow("user1") {
		val := z.GetWindowValue("user1")
		t.Fatalf("unexpected allow after exhausting limit (%d), value: %.2f", limit, val)
	}
}
