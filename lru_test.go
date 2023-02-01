package lru

import (
	"strconv"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := New[string](2, time.Hour, func(i int) {})
	c.Put("A", 1)
	time.Sleep(5 * time.Millisecond)
	c.Put("B", 2)
	if l := c.items.Len(); l != 2 {
		t.Fatalf("items size %d is not 2", l)
	}
	if l := c.items.Len(); l != 2 {
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
	t.Run("OnEvicted", func(t *testing.T) {
		called := false
		c := New[string](1, time.Hour, func(v int) {
			called = true
		})
		c.Put("a", 1)
		c.Put("b", 2)
		if !called {
			t.Fatal("not called when 'A' was evicted.")
		}
		called = true
		c.Remove("b")
		if !called {
			t.Fatal("not called when 'B' was removed.")
		}
	})
}

func BenchmarkPutRemove(b *testing.B) {
	b.Run("SmallCacheSmallItem", func(b *testing.B) {
		c := New[string](1, time.Hour, func(i int) {})
		for n := 0; n < b.N; n++ {
			c.Put("a", 1)
			c.Remove("a")
		}
	})
	// benchmark the log(n) insert / pop.
	b.Run("LargeCache", func(b *testing.B) {
		const size = 10_000
		c := New[string](size, time.Hour, func(i int) {})
		for i := 0; i < size-1; i++ {
			c.Put(strconv.Itoa(i), i)
		}
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			c.Put("a", 1)
			c.Remove("a")
		}
	})

	b.Run("LargeCacheLargeItems", func(b *testing.B) {
		const size = 10_000
		type item [64]byte
		c := New[string](size, time.Hour, func(i item) {})
		for i := 0; i < size-1; i++ {
			c.Put(strconv.Itoa(i), item{})
		}
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			c.Put("a", item{})
			c.Remove("a")
		}
	})
}

func BenchmarkPutGet(b *testing.B) {
	b.Run("Put", func(b *testing.B) {
		c := New[string](1, time.Hour, func(i int) {})
		for n := 0; n < b.N; n++ {
			c.Put("a", 1)
		}
	})
	b.Run("Get", func(b *testing.B) {
		c := New[string](1, time.Hour, func(i int) {})
		c.Put("a", 1)
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			_, _ = c.Get("a")
		}
	})
}

func BenchmarkEviction(b *testing.B) {
	c := New[string](1, time.Hour, func(i int) {})
	for n := 0; n < b.N; n++ {
		if n%2 == 0 {
			c.Put("a", 1)
		} else {
			c.Put("b", 2)
		}
	}
}

func BenchmarkAccess(b *testing.B) {
	type S struct{ i int }
	b.Run("MapToPtr", func(b *testing.B) {
		setup := func() map[string]*S {
			m := make(map[string]*S, 1)
			m["a"] = new(S)
			m["b"] = new(S)
			m["c"] = new(S)
			return m
		}
		m := setup()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var k string
			switch i % 3 {
			case 0:
				k = "a"
			case 1:
				k = "b"
			case 2:
				k = "c"
			}
			s := m[k]
			s.i = i
		}
	})
	b.Run("MapToIndexToSlice", func(b *testing.B) {
		setup := func() (map[string]int, []S) {
			m := make(map[string]int)
			s := make([]S, 3)
			m["a"] = 0
			m["b"] = 1
			m["c"] = 2
			return m, s
		}
		m, s := setup()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var k string
			switch i % 3 {
			case 0:
				k = "a"
			case 1:
				k = "b"
			case 2:
				k = "c"
			}
			s[m[k]].i = i
		}
	})
	b.Run("SwapPointers", func(b *testing.B) {
		a := []*item[string, *int]{{}, {}}
		for i := 0; i < b.N; i++ {
			a[0], a[1] = a[1], a[0]
		}
	})
	b.Run("SwapItem", func(b *testing.B) {
		a := []item[string, *int]{{}, {}}
		for i := 0; i < b.N; i++ {
			a[0], a[1] = a[1], a[0]
		}
	})
	b.Run("SwapLargeObject", func(b *testing.B) {
		type largeItem [128]byte
		a := []largeItem{{}, {}}
		for i := 0; i < b.N; i++ {
			a[0], a[1] = a[1], a[0]
		}
	})
}
