package oplog

// Log stores a slice of oplog entries that can be applied to a map. This
// data structure is not thread-safe, which means that any implementors
// should provide the concurrency synchronization guarantees.
type Log[K comparable, V any] struct {
	entries []*entry[K, V]

	// The most recent entry applied to the log
	latest *entry[K, V]
}

// Push pushes a new entry into the oplog and updates the oplog's latest entry
func (l *Log[K, V]) Push(e *entry[K, V]) {
	l.entries = append(l.entries, e)
	l.latest = e
}

// PushAndApply pushes a new entry to the oplog and applies that same entry to
// the provided map.
func (l *Log[K, V]) PushAndApply(e *entry[K, V], m *map[K]*V) {
	l.entries = append(l.entries, e)
	l.latest = e
	applyEntry(e, m)
}

// Apply applies the oplog to the specified map
func (l *Log[K, V]) Apply(m *map[K]*V) {
	for _, e := range l.entries {
		applyEntry(e, m)
	}
}

// Clear empties the oplog
func (l *Log[K, V]) Clear() {
	l.entries = []*entry[K, V]{}
}

// Len returns the current length of the oplog
func (l *Log[K, V]) Len() int {
	return len(l.entries)
}

// NewLog creates a new oplog with the given types
func NewLog[K comparable, V any]() *Log[K, V] {
	return &Log[K, V]{entries: []*entry[K, V]{}}
}

// applyEntry is a helper function for applying a single oplog entry to
// the destination map.
func applyEntry[K comparable, V any](e *entry[K, V], m *map[K]*V) {
	switch e.t {
	case entryTypeInsert:
		(*m)[e.k] = e.v
	case entryTypeDelete:
		delete(*m, e.k)
	case entryTypeClear:
		for k := range *m {
			delete(*m, k)
		}
	}
}
