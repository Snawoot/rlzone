package rlzone

import (
	"sync"
	"time"
)

type CounterValue interface {
	uint | uint8 | uint16 | uint32 | uint64
}

type RatelimitZone[K comparable, V CounterValue] struct {
	prevMap      map[K]V
	currMap      map[K]V
	prevWndStart time.Time
	currWndStart time.Time
	window       time.Duration
	limit        V
	mux          sync.Mutex
}

func New[K comparable, V CounterValue](window time.Duration, limit V) *RatelimitZone[K, V] {
	if window == 0 {
		panic("zero window value passed to ratelimit constructor!")
	}
	if limit == 0 {
		panic("zero limit value passed to ratelimit constructor!")
	}
	return &RatelimitZone[K,V]{
		prevMap: make(map[K]V),
		currMap: make(map[K]V),
		window:  window,
		limit:   limit,
	}
}

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
