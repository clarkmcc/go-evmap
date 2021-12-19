package eventual

import (
	"github.com/clarkmcc/go-evmap/pkg/oplog"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Map is a generic hashmap that provides low-contention, concurrent access
// to the underlying values. Readers don't block writers and vice versa with
// makes this data structure optimal for high-read, low-write scenarios. It
// does this by introducing eventual consistency, where readers are exposed
// to writes only when you explicitly say so.
//
// The underlying data structure is two maps (readable and writable). Writes
// are written to writable and reads are read from readable. At the point where
// a writer wants to expose it's writes to the reader, the writer calls Refresh.
// At this moment, the pointers to the readable and writable maps are atomically
// swapped, the readers now perform all their reads against (what was) the
// writable map and the writes written since the last Refresh are applied to
// (what was) the readable map, after which the writers start applying reads
// to the new (what was the readable map) writable map.
type Map[K comparable, V any] struct {
	// readable contains the values that are currently visible to the readers
	// and which is not being modified by the writer.
	readable *map[K]*V

	// writable contains the values that are currently being modified by the
	// writer(s).
	writable *map[K]*V

	// A slice of references to every reader that we need to monitor
	readers     []*Reader[K, V]
	readersLock sync.Mutex

	// This should be acquired as soon as we swapLocked readable and writable pointers
	// and should be released when we can prove that all readers are now looking
	// at writable.
	writeLock sync.Mutex

	// Used for replicating writes to m.writable after it's just been swapped
	// from m.readable
	oplog *oplog.Log[K, V]
}

// swapLocked takes the pointers to the readable and writable maps and swaps them
// so that the map that was previously used by the readers is now used by
// the writers and the map that was previously written to by the writers is
// now being read by the readers.
func (m *Map[K, V]) swapLocked() {
	readable := unsafe.Pointer(m.readable)
	writable := unsafe.Pointer(m.writable)
	m.readable = (*map[K]*V)(atomic.SwapPointer(&writable, readable))
	m.writable = (*map[K]*V)(atomic.SwapPointer(&readable, writable))
}

// syncLocked ensures that the value pointed to by m.readable is up-to-date with the
// value pointed to by m.writable. The only reason to call this function is after
// first calling swapLocked which causes the map that is most up to date (the map pointed
// to by m.writable before the swapLocked) to be switched to reader mode and the map
// that is least up to date (the map pointed to by m.readable before the swapLocked)
// to be switched to writer mode. After performing the swapLocked, we want to replicate
// of our writes syncLocked the previous syncLocked to the map that is now (after the swapLocked)
// pointed to by m.writable.
func (m *Map[K, V]) syncLocked() {
	// Clear the oplog after the syncLocked because we don't want to re-apply the same
	// operations more than once.
	defer m.oplog.Clear()

	// Apply the operations from the oplog to the map currently pointed to by
	// m.writable.
	m.oplog.Apply(m.writable)
}

// Refresh exposes the current state of the map to the readers. Under the hood
// refreshing causes the readable and writable maps to be swapped and the new
// writable map to be synced with the old writable map (now m.readable) using
// an internal oplog.
func (m *Map[K, V]) Refresh() {
	// Writers should be unable to apply writes to the map while we're getting up
	// to syncLocked. This same lock protects the oplog from being modified since all
	// modifications to this map are also applied to the oplog.
	m.writeLock.Lock()
	defer m.writeLock.Unlock()

	// Swap the readable and writable maps globally. This only swaps the pointers
	// in this data structure, but does not touch any of the readers.
	m.swapLocked()

	// Swap each reader's readable pointer with the new readable pointer
	for _, r := range m.readers {
		r.swapReadable(m.readable)
	}

	// We can assume at this point that all readers are now looking at the new
	// readable map which means the writable map is safe to perform writes against.
	m.syncLocked()
}

func (m *Map[K, V]) Reader() *Reader[K, V] {
	m.readersLock.Lock()
	defer m.readersLock.Unlock()
	r := NewReader(m)
	m.readers = append(m.readers, r)
	return r
}

func (m *Map[K, V]) Insert(key K, value *V) {
	m.writeLock.Lock()
	defer m.writeLock.Unlock()

	// This is a map modification so push the insert to the oplog and then apply
	// the same modification to the map itself
	m.oplog.PushAndApply(oplog.Insert[K, V](key, value), m.writable)
}

// Delete attempts to delete the key from the map and returns a boolean representing
// whether the key existed.
func (m *Map[K, V]) Delete(key K) bool {
	m.writeLock.Lock()
	defer m.writeLock.Unlock()

	// Check if the key exists before applying the deletion for obvious reasons
	_, ok := (*m.writable)[key]

	// This is a map modification so push the insert to the oplog and then apply
	// the same modification to the map itself
	m.oplog.PushAndApply(oplog.Delete[K, V](key), m.writable)
	return ok
}

// Clear removes all the keys from the map. Under-the-hood this function does
// not change the map pointer.
func (m *Map[K, V]) Clear() {
	m.writeLock.Lock()
	defer m.writeLock.Unlock()

	m.oplog.PushAndApply(oplog.Clear[K, V](), m.writable)
}

// NewMap creates a new Map of the given type with the provided options.
func NewMap[K comparable, V any]() *Map[K, V] {
	r := make(map[K]*V)
	w := make(map[K]*V)
	return &Map[K, V]{
		readable: &r,
		writable: &w,
		readers:  []*Reader[K, V]{},
		oplog:    oplog.NewLog[K, V](),
	}
}
