package rlzone

import (
	"testing"
	"time"
)

func TestRatelimitZone(t *testing.T) {
	const limit = 20
	const wnd = time.Second
	const sleepStep = wnd / limit

	z := Must(New[string](wnd, uint8(limit)))
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
	if _, ok := Must(NewSmallest[struct{}](wnd, uint64(1))).(*RatelimitZone[struct{}, uint8]); !ok {
		t.Fatal("expected uint8 variant of structure")
	}
	if _, ok := Must(NewSmallest[struct{}](wnd, uint64(255))).(*RatelimitZone[struct{}, uint8]); !ok {
		t.Fatal("expected uint8 variant of structure")
	}
	if _, ok := Must(NewSmallest[struct{}](wnd, uint64(256))).(*RatelimitZone[struct{}, uint16]); !ok {
		t.Fatal("expected uint16 variant of structure")
	}
	if _, ok := Must(NewSmallest[struct{}](wnd, uint64(65535))).(*RatelimitZone[struct{}, uint16]); !ok {
		t.Fatal("expected uint16 variant of structure")
	}
	if _, ok := Must(NewSmallest[struct{}](wnd, uint64(65536))).(*RatelimitZone[struct{}, uint32]); !ok {
		t.Fatal("expected uint32 variant of structure")
	}
	if _, ok := Must(NewSmallest[struct{}](wnd, uint64(4294967295))).(*RatelimitZone[struct{}, uint32]); !ok {
		t.Fatal("expected uint32 variant of structure")
	}
	if _, ok := Must(NewSmallest[struct{}](wnd, uint64(4294967296))).(*RatelimitZone[struct{}, uint64]); !ok {
		t.Fatal("expected uint64 variant of structure")
	}
}

type limiterCharacteristic struct {
	window time.Duration
	limit  uint64
}

func characterize[K comparable](limiter Ratelimiter[K]) limiterCharacteristic {
	switch l := limiter.(type) {
	case *RatelimitZone[K, uint8]:
		return limiterCharacteristic{
			window: l.window,
			limit:  uint64(l.limit),
		}
	case *RatelimitZone[K, uint16]:
		return limiterCharacteristic{
			window: l.window,
			limit:  uint64(l.limit),
		}
	case *RatelimitZone[K, uint32]:
		return limiterCharacteristic{
			window: l.window,
			limit:  uint64(l.limit),
		}
	case *RatelimitZone[K, uint64]:
		return limiterCharacteristic{
			window: l.window,
			limit:  uint64(l.limit),
		}
	case *RatelimitZone[K, uint]:
		return limiterCharacteristic{
			window: l.window,
			limit:  uint64(l.limit),
		}
	}
	panic("unknown type!")
}

func TestFromString(t *testing.T) {
	testCases := []struct {
		in  string
		out Ratelimiter[string]
	}{
		{
			in:  "100/1m",
			out: Must(NewSmallest[string](1*time.Minute, 100)),
		},
		{
			in:  "10/1m30s",
			out: Must(NewSmallest[string](90*time.Second, 10)),
		},
		{
			in:  "10000/1h",
			out: Must(NewSmallest[string](1*time.Hour, 10000)),
		},
		{
			in:  "70/1s",
			out: Must(NewSmallest[string](1*time.Second, 70)),
		},
		{
			in:  "10000000000/24h",
			out: Must(NewSmallest[string](24*time.Hour, 10000000000)),
		},
	}

	for i, tc := range testCases {
		created := Must(FromString[string](tc.in))
		if characterize(created) != characterize(tc.out) {
			t.Errorf("test case #%d failed: created %#v != %#v", i, created, tc.out)
		}
		if created.String() != tc.out.String() {
			t.Errorf("test case #%d failed: created.String() result %q != %q", i, created.String(), tc.out.String())
		}
		if created.Limit() != tc.out.Limit() {
			t.Errorf("test case #%d failed: created.Limit() result %d != %d", i, created.Limit(), tc.out.Limit())
		}
		if created.Window() != tc.out.Window() {
			t.Errorf("test case #%d failed: created.String() result %q != %q",
				i, created.Window().String(), tc.out.Window().String())
		}
	}
}

func TestAllowN(t *testing.T) {
	const limit = 20
	const wnd = time.Second

	z := Must(New[string](wnd, uint8(limit)))
	if !z.AllowN("", limit) {
		t.Fatalf("unexpected deny")
	}
	if z.Allow("") {
		t.Fatalf("unexpected allow")
	}
}

func benchmarkAllow[V CounterValue](b *testing.B) {
	const limit = 100
	z := Must(NewSmallest[int](time.Minute, limit+1))
	useBuckets := b.N + limit - 1 / limit
	for i := 0; i < b.N; i++ {
		if !z.Allow(i%useBuckets) {
			b.FailNow()
		}
	}
}

func BenchmarkAllowUint8(b *testing.B)  { benchmarkAllow[uint8](b) }
func BenchmarkAllowUint16(b *testing.B) { benchmarkAllow[uint16](b) }
func BenchmarkAllowUint32(b *testing.B) { benchmarkAllow[uint32](b) }
func BenchmarkAllowUint64(b *testing.B) { benchmarkAllow[uint64](b) }
