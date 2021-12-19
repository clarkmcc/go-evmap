package eventual

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func BenchmarkReads(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		m := newStdMap()

		// Fill the map
		for i := 0; i < 1_000_000; i++ {
			m.Insert(i, &i)
		}

		// Read from the map
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			m.Get(i)
		}
	})
	b.Run("evmap", func(b *testing.B) {
		m := NewMap[int, int]()
		reader := m.Reader()

		// Fill the map
		for i := 0; i < 1_000_000; i++ {
			m.Insert(i, &i)
		}

		// Expose the writes to the readers
		m.Refresh()

		// Read from the map
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reader.Get(i)
		}
	})
}

func BenchmarkParallelReads(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		m := newStdMap()

		// Fill the map
		for i := 0; i < 1_000_000; i++ {
			m.Insert(i, &i)
		}

		// Read from the map
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Get(rand.Intn(1_000_000))
			}
		})
	})
	b.Run("evmap", func(b *testing.B) {
		m := NewMap[int, int]()
		reader := m.Reader()

		// Fill the map
		for i := 0; i < 1_000_000; i++ {
			m.Insert(i, &i)
		}

		// Expose the writes to the readers
		m.Refresh()

		// Read from the map
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				reader.Get(rand.Intn(1_000_000))
			}
		})
	})
}

func BenchmarkWrites(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		m := newStdMap()

		// Fill the map
		b.ResetTimer()
		for i := 0; i < 1_000_000; i++ {
			m.Insert(i, &i)
		}
	})
	b.Run("evmap", func(b *testing.B) {
		m := NewMap[int, int]()

		// Fill the map
		b.ResetTimer()
		for i := 0; i < 1_000_000; i++ {
			m.Insert(i, &i)
		}
	})
}

func BenchmarkReadsAndWrites(b *testing.B) {
	b.Run("std", func(b *testing.B) {
		m := newStdMap()

		// Single Writer
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					key := rand.Intn(1_000_000)
					m.Insert(key, &key)
				}
			}
		}()

		// Multiple Readers
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Get(rand.Intn(1_000_000))
			}
		})
		close(done)
	})
	b.Run("evmap", func(b *testing.B) {
		m := NewMap[int, int]()

		// Single Writer
		done := make(chan struct{})
		go func() {
			i := 0
			for {
				select {
				case <-done:
					return
				default:
					key := rand.Intn(1_000_000)
					m.Insert(key, &key)
					i++
					// Replicate every 10,000 writes
					if i%10000 == 0 {
						m.Refresh()
					}
				}
			}
		}()

		// Multiple Readers
		b.RunParallel(func(pb *testing.PB) {
			reader := m.Reader()
			for pb.Next() {
				reader.Get(rand.Intn(1_000_000))
			}
		})
	})
}

type stdMap struct {
	lock sync.RWMutex
	m    map[int]*int
}

func (t *stdMap) Insert(key int, value *int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.m[key] = value
}

func (t *stdMap) Get(key int) (*int, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	v, ok := t.m[key]
	return v, ok
}

func newStdMap() *stdMap {
	return &stdMap{
		m: map[int]*int{},
	}
}
