package rlzone

import (
	"testing"
	"time"
)

func TestRatelimitZone(t *testing.T) {
	const limit = 20
	const wnd = time.Second
	const sleepStep = wnd / limit
	z := New[string](wnd, uint8(limit))
	for i := 0; i < limit-1; i++ {
		if i != 0 {
			time.Sleep(sleepStep)
		}
		if !z.Allow("user1") {
			val := z.GetWindowValue("user1")
			t.Fatalf("unexpected deny on iteration %d, window value = %.2f", i, val)
		}
	}

	denied := false
	for i := 0; i < limit+3/4+2; i++ {
		if z.Allow("user1") {
			denied = true
			break
		}
	}
	if !denied {
		val := z.GetWindowValue("user1")
		t.Fatalf("unexpected allow after exhausting limit (%d), value: %.2f, time: %s", limit, val, time.Now().UTC().String())
	}

	if !z.Allow("user2") {
		t.Fatalf("unexpected deny to unrelated user")
	}

	time.Sleep(sleepStep * 4)
	if !z.Allow("user1") {
		t.Fatalf("ratelimit doesn't cool down!")
	}

	denied = false
	for i := 0; i < 6; i++ {
		if !z.Allow("user1") {
			denied = true
			break
		}
	}

	if !denied {
		t.Fatal("ratelimit doesn't account past events!")
	}
}

func TestNewSmallest(t *testing.T) {
	wnd := time.Second
	if _, ok := NewSmallest[struct{}](wnd, uint64(1)).(*RatelimitZone[struct{}, uint8]); !ok {
		t.Fatal("expected uint8 variant of structure")
	}
	if _, ok := NewSmallest[struct{}](wnd, uint64(255)).(*RatelimitZone[struct{}, uint8]); !ok {
		t.Fatal("expected uint8 variant of structure")
	}
	if _, ok := NewSmallest[struct{}](wnd, uint64(256)).(*RatelimitZone[struct{}, uint16]); !ok {
		t.Fatal("expected uint16 variant of structure")
	}
	if _, ok := NewSmallest[struct{}](wnd, uint64(65535)).(*RatelimitZone[struct{}, uint16]); !ok {
		t.Fatal("expected uint16 variant of structure")
	}
	if _, ok := NewSmallest[struct{}](wnd, uint64(65536)).(*RatelimitZone[struct{}, uint32]); !ok {
		t.Fatal("expected uint32 variant of structure")
	}
	if _, ok := NewSmallest[struct{}](wnd, uint64(4294967295)).(*RatelimitZone[struct{}, uint32]); !ok {
		t.Fatal("expected uint32 variant of structure")
	}
	if _, ok := NewSmallest[struct{}](wnd, uint64(4294967296)).(*RatelimitZone[struct{}, uint64]); !ok {
		t.Fatal("expected uint64 variant of structure")
	}
}
