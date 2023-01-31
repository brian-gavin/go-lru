package lru

import (
	"strconv"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := New[string, int](2, time.Hour)
	c.Put("A", 1)
	time.Sleep(5 * time.Millisecond)
	c.Put("B", 2)
	if l := len(c.items); l != 2 {
		t.Fatalf("items size %d is not 2", l)
	}
	if l := c.lru.Len(); l != 2 {
		t.Fatalf("tlru size %d is not 2", l)
	}
	// LRU: [A, B]
	// Put C evicts A
	// [B, C]
	c.Put("C", 3)
	if _, e := c.Get("A"); e {
		t.Fatal("'A' should not be in the cache anymore!")
	}
	// refresh b: now LRU [C, B]
	if b, _ := c.Get("B"); b != 2 {
		t.Fatalf("'B' value %d is not 2", b)
	}
	c.Put("D", 4)
	if _, e := c.Get("C"); e {
		t.Log(c.items)
		t.Fatal("'C' should not be in the cache anymore!")
	}
}

func BenchmarkPutRemoveLargeCache(b *testing.B) {
	const size = 10_000
	c := New[string, int](size, time.Hour)
	for i := 0; i < size; i++ {
		c.Put(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.Put("a", 1)
		c.Remove("a")
	}
}

func BenchmarkPutRemoveLargeCacheLargeItems(b *testing.B) {
	const size = 10_000
	type item [64]byte
	c := New[string, item](size, time.Hour)
	for i := 0; i < size; i++ {
		c.Put(strconv.Itoa(i), item{})
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.Put("a", item{})
		c.Remove("a")
	}
}

func BenchmarkPut(b *testing.B) {
	c := New[string, int](1, time.Hour)
	for n := 0; n < b.N; n++ {
		c.Put("a", 1)
	}
}

func BenchmarkEviction(b *testing.B) {
	c := New[string, int](1, time.Hour)
	for n := 0; n < b.N; n++ {
		if n%2 == 0 {
			c.Put("a", 1)
		} else {
			c.Put("b", 2)
		}
	}
}
