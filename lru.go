package lru

import (
	"container/heap"
	"sync"
	"time"
)

type item[K any, V any] struct {
	k      K
	v      V
	expire time.Time
	index  int
}

// itemKVHeap implements the heap.Interface and maintains a mapping from K keys to items.
type itemKVHeap[K comparable, V any] struct {
	keyToItem map[K]*item[K, V] // maps a key to a *item
	pq        []*item[K, V]     // priority queue of *item ordered by expiration time
}

func makeItems[K comparable, V any](size int) itemKVHeap[K, V] {
	h := itemKVHeap[K, V]{
		keyToItem: make(map[K]*item[K, V], size),
		pq:        make([]*item[K, V], 0, size),
	}
	heap.Init(&h)
	return h
}

func (h itemKVHeap[K, V]) Len() int { return len(h.pq) }

// Less returns true if a < b. a < b if either a is expired, otherwise, if a.expire < b.expire
func (h itemKVHeap[K, V]) Less(i, j int) bool {
	a, b := h.pq[i], h.pq[j]
	return a.expire.Before(time.Now()) || a.expire.Before(b.expire)
}

func (h itemKVHeap[K, V]) Swap(i, j int) {
	pq := h.pq
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (h *itemKVHeap[K, V]) Push(x any) {
	n := len(h.pq)
	item := x.(*item[K, V])
	item.index = n
	h.keyToItem[item.k] = item
	h.pq = append(h.pq, item)
}

func (h *itemKVHeap[K, V]) Pop() any {
	old := h.pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	h.pq = old[0 : n-1]
	delete(h.keyToItem, item.k)
	return item
}

func (h *itemKVHeap[K, V]) Item(k K) (item *item[K, V], exists bool) {
	item, exists = h.keyToItem[k]
	return
}

type Cache[K comparable, V any] struct {
	mu        sync.Mutex
	items     itemKVHeap[K, V]
	size      int
	ttl       time.Duration
	onEvicted func(V)
}

func New[K comparable, V any](size int, ttl time.Duration, onEvicted func(V)) *Cache[K, V] {
	if size <= 0 {
		panic("Cache: cannot have 0 or negative size")
	}
	return &Cache[K, V]{
		size:      size,
		items:     makeItems[K, V](size),
		ttl:       ttl,
		onEvicted: onEvicted,
	}
}

func (c *Cache[K, V]) evict() {
	x := heap.Pop(&c.items)
	if x == nil {
		panic("evict called with empty heap")
	}
	evict := x.(*item[K, V])
	c.onEvicted(evict.v)
}

func (c *Cache[K, V]) update(item *item[K, V], v V) {
	item.v = v
	c.refresh(item)
}

func (c *Cache[K, V]) refresh(item *item[K, V]) {
	item.expire = time.Now().Add(c.ttl)
	heap.Fix(&c.items, item.index)
}

func (c *Cache[K, V]) add(item *item[K, V]) {
	heap.Push(&c.items, item)
}

func (c *Cache[K, V]) delete(item *item[K, V]) {
	heap.Remove(&c.items, item.index)
	c.onEvicted(item.v)
}

func (c *Cache[K, V]) Put(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item, exists := c.items.Item(k); exists {
		c.update(item, v)
		return
	}
	if c.items.Len() == c.size {
		c.evict()
	}
	item := &item[K, V]{
		v:      v,
		k:      k,
		expire: time.Now().Add(c.ttl),
	}
	c.add(item)
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, exists := c.items.Item(k)
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
	item, exists := c.items.Item(k)
	if !exists {
		return
	}
	c.delete(item)
}
