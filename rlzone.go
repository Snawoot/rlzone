// Generic rate limit by key using sliding window algorithm.
package rlzone

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CounterValue is a type constraint for unsigned integer values of counter.
type CounterValue interface {
	uint | uint8 | uint16 | uint32 | uint64
}

// Ratelimit is a generic interface for specific key type, but for any uint counter size.
type Ratelimiter[K comparable] interface {
	Allow(key K) bool
	GetWindowValue(key K) float64
}

// RatelimitZone controls how frequently events are allowed to happen. It implements
// sliding window of duration `window`, allowing approximately `limit` events within that
// time frame.
//
// A RatelimitZone is safe for concurrent use by multiple goroutines.
type RatelimitZone[K comparable, V CounterValue] struct {
	prevMap      map[K]V
	currMap      map[K]V
	prevWndStart time.Time
	currWndStart time.Time
	window       time.Duration
	limit        V
	mux          sync.Mutex
}

const (
	uint8Max  = uint64(^uint8(0))
	uint16Max = uint64(^uint16(0))
	uint32Max = uint64(^uint32(0))
)

// Must is a helper that wraps a call to a function returning (Ratelimiter[K], error)
// and panics if the error is non-nil. It is intended for use in rate limiter
// initializations such as
//
//	rl := Must(New[string](1*Second, 10))
func Must[K comparable](rl Ratelimiter[K], err error) Ratelimiter[K] {
	if err != nil {
		panic(err)
	}
	return rl
}

// NewSmallest creates a new rate limiter with counter type wide enough to fit limit value.
func NewSmallest[K comparable](window time.Duration, limit uint64) (Ratelimiter[K], error) {
	switch {
	case limit <= uint8Max:
		return New[K](window, uint8(limit))
	case limit <= uint16Max:
		return New[K](window, uint16(limit))
	case limit <= uint32Max:
		return New[K](window, uint32(limit))
	}
	return New[K](window, limit)
}

// FromString creates a new rate limiter from string specification <limit>/<duration>.
// E.g. "100/20m" corresponds to 100 allowed events in 20 minutes sliding time window.
//
// See https://pkg.go.dev/time#ParseDuration for reference of duration format.
func FromString[K comparable](limiterSpec string) (Ratelimiter[K], error) {
	parts := strings.SplitN(limiterSpec, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("bad limiter specification format, expected: <count>/<duration>, error: %w",
			errors.New("slash is missing"))
	}

	count, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad limiter specification format, expected: <count>/<duration>, error: %w", err)
	}

	window, err := time.ParseDuration(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad limiter specification format, expected: <count>/<duration>, error: %w", err)
	}

	return NewSmallest[K](window, count)
}

// New creates new RatelimitZone with key type K and counter type V.
// It will allow approximately `limit` events within `window` time frame.
func New[K comparable, V CounterValue](window time.Duration, limit V) (*RatelimitZone[K, V], error) {
	if window <= 0 {
		return nil, errors.New("zero window value passed to ratelimit constructor!")
	}
	if limit == 0 {
		return nil, errors.New("zero limit value passed to ratelimit constructor!")
	}
	return &RatelimitZone[K, V]{
		prevMap: make(map[K]V),
		currMap: make(map[K]V),
		window:  window,
		limit:   limit,
	}, nil
}

// Allow reports whether and event may happen now.
func (z *RatelimitZone[K, V]) Allow(key K) bool {
	reqPrevWndStart, reqCurrWndStart, now := z.getTimePoints()

	z.mux.Lock()
	defer z.mux.Unlock()

	val := z.getWindowValue(key, reqPrevWndStart, reqCurrWndStart, now)
	if val+1 > float64(z.limit) {
		return false
	}

	z.shiftMaps(reqPrevWndStart, reqCurrWndStart)

	z.incCounter(key, reqCurrWndStart)

	return true
}

func (z *RatelimitZone[K, V]) getTimePoints() (time.Time, time.Time, time.Time) {
	now := time.Now().UTC().Truncate(0)
	reqCurrWndStart := now.Truncate(z.window)
	reqPrevWndStart := reqCurrWndStart.Add(-z.window)
	return reqPrevWndStart, reqCurrWndStart, now
}

func (z *RatelimitZone[K, V]) shiftMaps(prevStart, currStart time.Time) {
	newPrevMap := z.getWndMap(prevStart, true)
	z.prevMap = newPrevMap
	z.prevWndStart = prevStart
	newCurrMap := z.getWndMap(currStart, true)
	z.currMap = newCurrMap
	z.currWndStart = currStart
}

// GetWindowValue returns estimation how many events happend for a `key` within
// sliding time window.
func (z *RatelimitZone[K, V]) GetWindowValue(key K) float64 {
	reqPrevWndStart, reqCurrWndStart, now := z.getTimePoints()
	z.mux.Lock()
	defer z.mux.Unlock()
	return z.getWindowValue(key, reqPrevWndStart, reqCurrWndStart, now)
}

func (z *RatelimitZone[K, V]) getWindowValue(key K, prevWndStart, currWndStart, now time.Time) float64 {
	prevCtr := z.getCounter(key, prevWndStart)
	currCtr := z.getCounter(key, currWndStart)
	multiplier := 1 - (float64(now.Sub(currWndStart)) / float64(z.window))
	res := float64(prevCtr)*multiplier + float64(currCtr)
	return res
}

func (z *RatelimitZone[K, V]) getCounter(key K, wndStart time.Time) V {
	if m := z.getWndMap(wndStart, false); m != nil {
		return m[key]
	}
	return 0
}

func (z *RatelimitZone[K, V]) incCounter(key K, wndStart time.Time) {
	if m := z.getWndMap(wndStart, false); m != nil {
		m[key]++
	}
}

func (z *RatelimitZone[K, V]) getWndMap(wndStart time.Time, create bool) map[K]V {
	switch {
	case wndStart.Equal(z.currWndStart):
		return z.currMap
	case wndStart.Equal(z.prevWndStart):
		return z.prevMap
	}
	if create {
		return make(map[K]V)
	}
	return nil
}
