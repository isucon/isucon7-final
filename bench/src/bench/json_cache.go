package main

import (
	"bench/counter"
	"sync"
)

var (
	StrictCheckCacheConflict bool

	jsonCacheMtx  sync.Mutex
	binCache      = map[uint64]*GameStatus{}
	addingCache   = map[uint64][]Adding{}
	scheduleCache = map[uint64][]Schedule{}
	itemsCache    = map[uint64][]Item{}
	onSaleCache   = map[uint64][]OnSale{}
)

func retrieveBinCache(b []byte, h uint64) (*GameStatus, bool) {
	jsonCacheMtx.Lock()
	v, ok := binCache[h]
	jsonCacheMtx.Unlock()

	if !ok {
		return nil, false
	}

	if !StrictCheckCacheConflict {
		counter.IncKey("hash-bin-hit")
		return v, true
	}

	if checkSameAdding(b, v.Adding) && checkSameSchedule(b, v.Schedule) &&
		checkSameItems(b, v.Items) && checkSameOnSale(b, v.OnSale) {
		counter.IncKey("hash-bin-hit")
		return v, true
	} else {
		counter.IncKey("hash-bin-conflict")
		return nil, false
	}
}

func storeBinCache(h uint64, gs *GameStatus) {
	jsonCacheMtx.Lock()
	binCache[h] = gs
	jsonCacheMtx.Unlock()
}

func retrieveAddingCache(b []byte, h uint64) ([]Adding, bool) {
	jsonCacheMtx.Lock()
	v, ok := addingCache[h]
	jsonCacheMtx.Unlock()

	if !ok {
		return nil, false
	}

	if !StrictCheckCacheConflict {
		counter.IncKey("hash-adding-hit")
		return v, true
	}

	if checkSameAdding(b, v) {
		counter.IncKey("hash-adding-hit")
		return v, true
	} else {
		counter.IncKey("hash-adding-conflict")
		return nil, false
	}
}

func storeAddingCache(h uint64, adding []Adding) {
	jsonCacheMtx.Lock()
	addingCache[h] = adding
	jsonCacheMtx.Unlock()
}

func retrieveScheduleCache(b []byte, h uint64) ([]Schedule, bool) {
	jsonCacheMtx.Lock()
	v, ok := scheduleCache[h]
	jsonCacheMtx.Unlock()

	if !ok {
		return nil, false
	}

	if !StrictCheckCacheConflict {
		counter.IncKey("hash-schedule-hit")
		return v, true
	}

	if checkSameSchedule(b, v) {
		counter.IncKey("hash-schedule-hit")
		return v, true
	} else {
		counter.IncKey("hash-schedule-conflict")
		return nil, false
	}
}

func storeScheduleCache(h uint64, schedule []Schedule) {
	jsonCacheMtx.Lock()
	scheduleCache[h] = schedule
	jsonCacheMtx.Unlock()
}

func retrieveItemsCache(b []byte, h uint64) ([]Item, bool) {
	jsonCacheMtx.Lock()
	v, ok := itemsCache[h]
	jsonCacheMtx.Unlock()

	if !ok {
		return nil, false
	}

	if !StrictCheckCacheConflict {
		counter.IncKey("hash-items-hit")
		return v, true
	}

	if checkSameItems(b, v) {
		counter.IncKey("hash-items-hit")
		return v, true
	} else {
		counter.IncKey("hash-items-conflict")
		return nil, false
	}
}

func storeItemsCache(h uint64, items []Item) {
	jsonCacheMtx.Lock()
	itemsCache[h] = items
	jsonCacheMtx.Unlock()
}

func retrieveOnSaleCache(b []byte, h uint64) ([]OnSale, bool) {
	jsonCacheMtx.Lock()
	v, ok := onSaleCache[h]
	jsonCacheMtx.Unlock()

	if !ok {
		return nil, false
	}

	if !StrictCheckCacheConflict {
		counter.IncKey("hash-onsale-hit")
		return v, true
	}

	if checkSameOnSale(b, v) {
		counter.IncKey("hash-onsale-hit")
		return v, true
	} else {
		counter.IncKey("hash-onsale-conflict")
		return nil, false
	}
}

func storeOnSaleCache(h uint64, onSale []OnSale) {
	jsonCacheMtx.Lock()
	onSaleCache[h] = onSale
	jsonCacheMtx.Unlock()
}
