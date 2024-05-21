package cmd

import "sync"

type Cache[T any] struct {
	items map[string]T
	mux   sync.RWMutex
}

func newCache[T any]() *Cache[T] {
	return &Cache[T]{
		items: map[string]T{},
	}
}

func (cache *Cache[T]) Get(key string) T {
	cache.mux.RLock()
	defer cache.mux.RUnlock()
	return cache.items[key]
}

func (cache *Cache[T]) Set(key string, val T) {
	cache.mux.Lock()
	defer cache.mux.Unlock()
	cache.items[key] = val
}

func (cache *Cache[T]) Del(key string) {
	cache.mux.Lock()
	defer cache.mux.Unlock()
	if _, ok := cache.items[key]; !ok {
		return
	}
	delete(cache.items, key)
}
