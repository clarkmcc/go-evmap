/*
Copyright (C) 2020 Print Tracker, LLC - All Rights Reserved

Unauthorized copying of this file, via any medium is strictly prohibited
as this source code is proprietary and confidential. Dissemination of this
information or reproduction of this material is strictly forbidden unless
prior written permission is obtained from Print Tracker, LLC.
*/

package eventual

import (
	"github.com/clarkmcc/go-evmap/pkg/oplog"
	"sync"
	"sync/atomic"
	unsafe "unsafe"
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

	// This should be acquired as soon as we swap readable and writable pointers
	// and should be released when we can prove that all readers are now looking
	// at writable.
	writeLock sync.Mutex

	// Tracks the number of writes since the last refresh. This should also
	// match the length of the oplog. It's access should always be guarded
	// by writeLock.
	replicationWriteLag int

	// The number of writes that are allowed to occur without making them available
	// to the readers.
	maxReplicationWriteLag int

	// Used for replicating writes to m.writable after it's just been swapped
	// from m.readable
	oplog *oplog.Log[K, V]

	// Guards the initial setup processes and ensures that they only happen once
	// and also allows this struct to be initialized with its zero value.
	initOnce sync.Once
}

// init initializes the map with all the default fields.
func (m *Map[K, V]) init() {
	m.initOnce.Do(func() {
		r := make(map[K]*V)
		m.readable = &r
		w := make(map[K]*V)
		m.writable = &w
		m.oplog = oplog.NewLog[K, V]()
	})
}

// swap takes the pointers to the readable and writable maps and swaps them
// so that the map that was previously used by the readers is now used by
// the writers and the map that was previously written to by the writers is
// now being read by the readers.
func (m *Map[K, V]) swap() {
	readable := unsafe.Pointer(m.readable)
	writable := unsafe.Pointer(m.writable)
	m.readable = (*map[K]*V)(atomic.SwapPointer(&writable, readable))
	m.writable = (*map[K]*V)(atomic.SwapPointer(&readable, writable))
}

// sync ensures that the value pointed to by m.readable is up-to-date with the
// value pointed to by m.writable. The only reason to call this function is after
// first calling swap which causes the map that is most up to date (the map pointed
// to by m.writable before the swap) to be switched to reader mode and the map
// that is least up to date (the map pointed to by m.readable before the swap)
// to be switched to writer mode. After performing the swap, we want to replicate
// of our writes sync the previous sync to the map that is now (after the swap)
// pointed to by m.writable.
func (m *Map[K, V]) sync() {
	// Writers should be unable to apply writes to the map while we're getting up
	// to sync. This same lock protects the oplog from being modified since all
	// modifications to this map are also applied to the oplog.
	m.writeLock.Lock()
	defer m.writeLock.Unlock()

	// Clear the oplog after the sync because we don't want to re-apply the same
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
	m.swap()
	m.sync()
	m.replicationWriteLag = 0
}

func (m *Map[K, V]) Insert(key K, value *V) {
	m.writeLock.Lock()
	defer m.observeWrite()
	defer m.writeLock.Unlock()

	// This is a map modification so push the insert to the oplog and then apply
	// the same modification to the map itself
	m.oplog.PushAndApply(oplog.Insert[K, V](key, value), m.writable)
}

// Delete attempts to delete the key from the map and returns a boolean representing
// whether the key existed.
func (m *Map[K, V]) Delete(key K) bool {
	m.writeLock.Lock()
	defer m.observeWrite()
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
	defer m.observeWrite()
	defer m.writeLock.Unlock()

	m.oplog.PushAndApply(oplog.Clear[K, V](), m.writable)
}

// Has returns whether the map has the specified key
func (m *Map[K, V]) Has(key K) bool {
	_, ok := (*m.readable)[key]
	return ok
}

// Get returns the value at the provided key. This will return a pointer to the
// value at the key, as well as whether the key exists or not. It's possible to
// add a nil value to the map, so this distinction is relevant.
func (m *Map[K, V]) Get(key K) (*V, bool) {
	v, ok := (*m.readable)[key]
	return v, ok
}

// observeWrite observes a write and determines whether to refresh based on configuration
func (m *Map[K, V]) observeWrite() {
	m.replicationWriteLag++
	if m.maxReplicationWriteLag > 0 && m.replicationWriteLag > m.maxReplicationWriteLag {
		m.Refresh()
	}
}

// NewMap creates a new Map of the given type with the provided options.
func NewMap[K comparable, V any](options ...OptionFunc) *Map[K, V] {
	m := Map[K, V]{}
	m.init()
	opts := Options{}
	for _, fn := range options {
		fn(&opts)
	}
	m.maxReplicationWriteLag = opts.MaxReplicationWriteLag
	return &m
}
