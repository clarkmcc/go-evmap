package eventual

import (
	"sync"
	"unsafe"
)

type Reader[K comparable, V any] struct {
	closed bool
	m      *Map[K, V]
	lock   sync.Mutex

	readable unsafe.Pointer
}

func (r *Reader[K, V]) Get(key K) (*V, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		panic("reader closed")
	}
	v, ok := (*((*map[K]*V)(r.readable)))[key]
	return v, ok
}

func (r *Reader[K, V]) Has(key K) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.closed {
		panic("reader closed")
	}
	_, ok := (*((*map[K]*V)(r.readable)))[key]
	return ok
}

// Close removes the reader from the map. The caller will not be able
// to use the reader anymore. Reading after close will result in a panic
func (r *Reader[K, V]) Close() {
	r.m.readersLock.Lock()
	defer r.m.readersLock.Unlock()
	for idx, reader := range r.m.readers {
		if unsafe.Pointer(reader) == unsafe.Pointer(r) {
			remove[*Reader[K, V]](r.m.readers, idx)
			break
		}
	}
}

func (r *Reader[K, V]) swapReadable(m *map[K]*V) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.readable = unsafe.Pointer(m)
}

func NewReader[K comparable, V any](m *Map[K, V]) *Reader[K, V] {
	return &Reader[K, V]{m: m, readable: unsafe.Pointer(m.readable)}
}

func remove[V any](s []V, i int) []V {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
