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

type items[K comparable, V any] struct {
	keyToItem map[K]*item[K, V]
	pq        []*item[K, V]
}

func makeItems[K comparable, V any](size int) items[K, V] {
	i := items[K, V]{
		keyToItem: make(map[K]*item[K, V], size),
		pq:        make([]*item[K, V], 0, size),
	}
	heap.Init(&i)
	return i
}

func (is items[K, V]) Len() int { return len(is.pq) }

// Less returns true if a < b. a < b if either a is expired, otherwise, if a.expire < b.expire
func (is items[K, V]) Less(i, j int) bool {
	a, b := is.pq[i], is.pq[j]
	return a.expire.Before(time.Now()) || a.expire.Before(b.expire)
}

func (is items[K, V]) Swap(i, j int) {
	pq := is.pq
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (is *items[K, V]) Push(x any) {
	n := len(is.pq)
	item := x.(*item[K, V])
	item.index = n
	is.keyToItem[item.k] = item
	is.pq = append(is.pq, item)
}

func (is *items[K, V]) Pop() any {
	old := is.pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	is.pq = old[0 : n-1]
	delete(is.keyToItem, item.k)
	return item
}

type Cache[K comparable, V any] struct {
	mu        sync.Mutex
	items     items[K, V]
	size      int
	ttl       time.Duration
	onEvicted func(V)
}

func New[K comparable, V any](size int, ttl time.Duration, onEvicted func(V)) *Cache[K, V] {
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
		panic("evict called with empty tlru")
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
	if item, exists := c.items.keyToItem[k]; exists {
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
	item, exists := c.items.keyToItem[k]
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
	item, exists := c.items.keyToItem[k]
	if !exists {
		return
	}
	c.delete(item)
}
