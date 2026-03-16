package crawler

import (
	"sync"
	"time"
)

type cacheEntry[T any] struct {
	e       T
	addTime time.Time
}

type CacheTTL[T any, R comparable] struct {
	rw         sync.Mutex
	values     map[R]cacheEntry[T]
	stopChanel chan struct{}
}

func NewCacheTTL[T any, R comparable]() *CacheTTL[T, R] {
	c := &CacheTTL[T, R]{
		values:     make(map[R]cacheEntry[T]),
		stopChanel: make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopChanel:
				return
			case <-ticker.C:
				c.cleanCache()
			}
		}
	}()

	return c
}

func (c *CacheTTL[T, R]) CloseCacheTTL() {
	close(c.stopChanel)
}

func (c *CacheTTL[T, R]) cleanCache() {
	withLock(&c.rw, func() {
		for key, v := range c.values {
			if time.Since(v.addTime) > 1*time.Second {
				delete(c.values, key)
			}
		}
	})
}

func (c *CacheTTL[T, R]) Get(key R) (T, bool) {
	c.rw.Lock()
	defer c.rw.Unlock()

	v, ok := c.values[key]
	if !ok {
		var zeroValue T
		return zeroValue, false
	}

	if time.Since(v.addTime) > 1*time.Second {
		delete(c.values, key)
		var zeroValue T
		return zeroValue, false
	}

	return v.e, true
}

func (c *CacheTTL[T, R]) Set(key R, value T) {
	withLock(&c.rw, func() {
		c.values[key] = cacheEntry[T]{e: value, addTime: time.Now()}
	})
}
