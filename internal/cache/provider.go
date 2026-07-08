package cache

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("cache: key not found")

// Provider định nghĩa interface chung cho caching layer.
// Cho phép swap giữa in-memory, Redis, hoặc bất kỳ backend nào.
type Provider interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte, ttl time.Duration)
	Delete(key string)
	// GetOrSet lấy từ cache, nếu miss thì gọi fn, set vào cache rồi return.
	GetOrSet(key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error)
}
