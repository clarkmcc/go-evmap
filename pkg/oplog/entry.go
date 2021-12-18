package oplog

// Indicates the supported types of oplog entries that can be stored in the oplog. These
// types are limited to the modifications that can be made to a map.
type entryType uint8

const (
	entryTypeInsert entryType = iota
	entryTypeDelete
	entryTypeClear
)

// entry is an oplog entry that may (but not always) be associated with a v
type entry[K comparable, V any] struct {
	t entryType
	k K
	v *V
}

// newEntry creates a new oplog entry with the associated type and v
func newEntry[K comparable, V any](t entryType, key K, value *V) *entry[K, V] {
	return &entry[K, V]{
		t: t,
		k: key,
		v: value,
	}
}

// Insert creates an oplog entry that inserts a v into the map
func Insert[K comparable, V any](key K, value *V) *entry[K, V] {
	return newEntry(entryTypeInsert, key, value)
}

// Delete creates an oplog entry that deletes a v from the map
func Delete[K comparable, V any](key K) *entry[K, V] {
	return newEntry[K, V](entryTypeDelete, key, nil)
}

// Clear clears the entire contents from the map
func Clear[K comparable, V any]() *entry[K, V] {
	return &entry[K, V]{
		t: entryTypeClear,
	}
}
