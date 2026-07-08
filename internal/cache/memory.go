package cache

import (
	"sync"
	"time"
)

type item struct {
	data      []byte
	expiresAt time.Time
}

// MemoryCache là in-memory implementation của Provider.
// Thread-safe với sync.RWMutex.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]item
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]item),
	}
}

func (c *MemoryCache) Get(key string) ([]byte, error) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	if !it.expiresAt.IsZero() && time.Now().After(it.expiresAt) {
		c.Delete(key)
		return nil, ErrNotFound
	}
	return it.data, nil
}

func (c *MemoryCache) Set(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it := item{data: data}
	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = it
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *MemoryCache) GetOrSet(key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	data, err := c.Get(key)
	if err == nil {
		return data, nil
	}
	data, err = fn()
	if err != nil {
		return nil, err
	}
	c.Set(key, data, ttl)
	return data, nil
}
