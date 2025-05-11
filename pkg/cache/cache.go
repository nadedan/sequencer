package cache

import (
	"sync"

	"golang.org/x/exp/constraints"
)

type Cache[K constraints.Ordered, V any] struct {
	m sync.RWMutex
	c map[K]V
}

func New[K constraints.Ordered, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		c: make(map[K]V),
	}
}

func (c *Cache[K, V]) Store(k K, v V) {
	c.m.Lock()
	defer c.m.Unlock()
	c.c[k] = v
}

func (c *Cache[K, V]) Load(k K) (V, bool) {
	c.m.RLock()
	defer c.m.RUnlock()
	v, ok := c.c[k]
	return v, ok
}

func (c *Cache[K, V]) Delete(k K) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.c, k)
}

func (c *Cache[K, V]) Clear() {
	c.m.Lock()
	defer c.m.Unlock()
	c.c = make(map[K]V)
}
