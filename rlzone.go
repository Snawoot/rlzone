package rlzone

import (
	"sync"
	"time"
)

type RatelimitZone[K comparable] struct {
	prevMap      map[K]uint32
	currMap      map[K]uint32
	prevWndStart time.Time
	currWndStart time.Time
	window       time.Duration
	limit        uint32
	mux          sync.Mutex
}

func New[K comparable](window time.Duration, limit uint32) *RatelimitZone[K] {
	return &RatelimitZone[K]{
		prevMap: make(map[K]uint32),
		currMap: make(map[K]uint32),
		window:  window,
		limit:   limit,
	}
}

func (z *RatelimitZone[K]) Allow(key K) bool {
	reqPrevWndStart, reqCurrWndStart, now := z.getTimePoints()

	z.mux.Lock()
	defer z.mux.Unlock()

	val := z.getWindowValue(key, reqPrevWndStart, reqCurrWndStart, now)
	if float64(z.limit) <= val {
		return false
	}

	z.orderMaps(reqPrevWndStart, reqCurrWndStart)

	z.incCounter(key, reqCurrWndStart)

	return true
}

func (z *RatelimitZone[K]) getTimePoints() (time.Time, time.Time, time.Time) {
	now := time.Now().UTC().Truncate(0)
	reqCurrWndStart := now.Truncate(z.window)
	reqPrevWndStart := reqCurrWndStart.Add(-z.window)
	return reqPrevWndStart, reqCurrWndStart, now
}

func (z *RatelimitZone[K]) orderMaps(prevStart, currStart time.Time) {
	newPrevMap := z.getWndMap(prevStart, true)
	z.prevMap = newPrevMap
	z.prevWndStart = prevStart
	newCurrMap := z.getWndMap(currStart, true)
	z.currMap = newCurrMap
	z.currWndStart = currStart
}

func (z *RatelimitZone[K]) GetWindowValue(key K) float64 {
	reqPrevWndStart, reqCurrWndStart, now := z.getTimePoints()
	z.mux.Lock()
	defer z.mux.Unlock()
	return z.getWindowValue(key, reqPrevWndStart, reqCurrWndStart, now)
}

func (z *RatelimitZone[K]) getWindowValue(key K, prevWndStart, currWndStart, now time.Time) float64 {
	prevCtr := z.getCounter(key, prevWndStart)
	currCtr := z.getCounter(key, currWndStart)
	multiplier := 1 - (float64(now.Sub(currWndStart)) / float64(z.window))
	return float64(prevCtr)*multiplier + float64(currCtr)
}

func (z *RatelimitZone[K]) getCounter(key K, wndStart time.Time) uint32 {
	if m := z.getWndMap(wndStart, false); m != nil {
		return m[key]
	}
	return 0
}

func (z *RatelimitZone[K]) incCounter(key K, wndStart time.Time) {
	if m := z.getWndMap(wndStart, false); m != nil {
		m[key]++
	}
}

func (z *RatelimitZone[K]) getWndMap(wndStart time.Time, create bool) map[K]uint32 {
	switch {
	case wndStart.Equal(z.currWndStart):
		return z.currMap
	case wndStart.Equal(z.prevWndStart):
		return z.prevMap
	}
	if create {
		return make(map[K]uint32)
	}
	return nil
}
