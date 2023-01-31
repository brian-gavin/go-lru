package lru

import (
	"container/heap"
	"sync"
	"time"
)

type lruItem[K any] struct {
	k      K
	expire time.Time
	index  int
}

type priorityQueue[K any] []*lruItem[K]

func (pq priorityQueue[K]) Len() int { return len(pq) }

// Less returns true if a < b. a < b if either a is expired, otherwise, if a.expire < b.expire
func (pq priorityQueue[K]) Less(i, j int) bool {
	a, b := pq[i], pq[j]
	return a.expire.Before(time.Now()) || a.expire.Before(b.expire)
}

func (pq priorityQueue[K]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue[K]) Push(x any) {
	n := len(*pq)
	item := x.(*lruItem[K])
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue[K]) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

type cacheItem[K, V any] struct {
	lruItem *lruItem[K]
	v       V
}

type Cache[K comparable, V any] struct {
	mu        sync.Mutex
	items     map[K]cacheItem[K, V]
	lru       priorityQueue[K]
	size      int
	ttl       time.Duration
	onEvicted func(V)
}

func New[K comparable, V any](size int, ttl time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		size:  size,
		items: make(map[K]cacheItem[K, V], size),
		lru:   priorityQueue[K]{},
		ttl:   ttl,
	}
	heap.Init(&c.lru)
	return c
}

func (c *Cache[K, V]) evict() {
	x := heap.Pop(&c.lru)
	if x == nil {
		panic("evict called with empty tlru")
	}
	evict := x.(*lruItem[K])
	item := c.items[evict.k]
	c.onEvicted(item.v)
	delete(c.items, evict.k)
}

func (c *Cache[K, V]) update(item cacheItem[K, V], v V) {
	item.v = v
	c.items[item.lruItem.k] = item
	c.refresh(item)
}

func (c *Cache[K, V]) refresh(item cacheItem[K, V]) {
	item.lruItem.expire = time.Now().Add(c.ttl)
	heap.Fix(&c.lru, item.lruItem.index)
}

func (c *Cache[K, V]) add(item cacheItem[K, V]) {
	c.items[item.lruItem.k] = item
	heap.Push(&c.lru, item.lruItem)
}

func (c *Cache[K, V]) delete(item cacheItem[K, V]) {
	heap.Remove(&c.lru, item.lruItem.index)
	c.onEvicted(item.v)
	delete(c.items, item.lruItem.k)
}

func (c *Cache[K, V]) Put(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item, exists := c.items[k]; exists {
		c.update(item, v)
		return
	}
	if len(c.items) == c.size {
		c.evict()
	}
	item := cacheItem[K, V]{
		v: v,
		lruItem: &lruItem[K]{
			k:      k,
			expire: time.Now().Add(c.ttl),
		},
	}
	c.add(item)
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, exists := c.items[k]
	if !exists {
		var v V
		return v, false
	}
	c.refresh(item)
	return item.v, true
}

func (c *Cache[K, V]) Remove(k K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, exists := c.items[k]
	if !exists {
		return
	}
	c.delete(item)
}
