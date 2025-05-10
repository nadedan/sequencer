package sequencer

import (
	"sync"

	"golang.org/x/exp/constraints"
)

type cache[K constraints.Ordered, V any] struct {
	m sync.RWMutex
	c map[K]V
}

func newCache[K constraints.Ordered, V any]() *cache[K, V] {
	return &cache[K, V]{
		c: make(map[K]V),
	}
}

func (c *cache[K, V]) store(k K, v V) {
	c.m.Lock()
	defer c.m.Unlock()
	c.c[k] = v
}

func (c *cache[K, V]) load(k K) (V, bool) {
	c.m.RLock()
	defer c.m.RUnlock()
	v, ok := c.c[k]
	return v, ok
}

func (c *cache[K, V]) delete(k K) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.c, k)
}

func (c *cache[K, V]) clear() {
	c.m.Lock()
	defer c.m.Unlock()
	c.c = make(map[K]V)
}
