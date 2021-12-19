package eventual

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMap(t *testing.T) {
	m := NewMap[string, any]()

	t.Run("Insert", func(t *testing.T) {
		m.Insert("foo", nil)
		m.Insert("bar", nil)

		// Check that these keys are in the writable map
		assert.Len(t, *m.writable, 2)
		assert.Len(t, *m.readable, 0)
	})
	t.Run("Refresh", func(t *testing.T) {
		m.Refresh()

		// Check that the keys have been moved to the readable map
		assert.Len(t, *m.readable, 2)

		// Check that the keys have been re-applied to the new writable map
		assert.Len(t, *m.writable, 2)
	})
	t.Run("Delete", func(t *testing.T) {
		m.Delete("foo")

		// Check that the readers haven't seen this change
		assert.Len(t, *m.readable, 2)

		// But the writers have
		assert.Len(t, *m.writable, 1)
	})
	t.Run("has & get", func(t *testing.T) {
		reader := m.Reader()
		v, ok := reader.Get("foo")

		// Readers haven't seen this key deleted yet
		assert.True(t, ok)
		assert.True(t, reader.Has("foo"))

		// Run a refresh
		m.Refresh()

		// Readers should see the key missing now
		v, ok = reader.Get("foo")
		assert.Nil(t, v)
		assert.False(t, ok)
		assert.False(t, reader.Has("foo"))
	})
	t.Run("Clear", func(t *testing.T) {
		m.Clear()

		// Readers shouldn't see the clear yet
		assert.Len(t, *m.readable, 1, "reader shouldn't see the clear yet")
		assert.Len(t, *m.writable, 0, "writer should have seen the clear")

		m.Refresh()

		assert.Len(t, *m.readable, 0, "reader should see the clear after refresh")
	})
}

func TestMap_swap(t *testing.T) {
	m := NewMap[string, any]()

	// Check the pointers
	ptr1 := m.writable
	ptr2 := m.readable

	// Swap
	m.swapLocked()

	// Check the pointers again
	assert.Equal(t, m.writable, ptr2)
	assert.Equal(t, m.readable, ptr1)
}

func TestMap_sync(t *testing.T) {
	m := NewMap[string, any]()

	// Add a value
	m.Insert("foo", nil)
	assert.Equal(t, m.oplog.Len(), 1)

	// Check the writable map
	assert.Len(t, *m.writable, 1, "one value should have been written to writable")

	// Perform the swapLocked
	m.swapLocked()
	assert.Len(t, *m.writable, 0, "writable has been swapped with readable and the new writable should be empty")

	// Perform the syncLocked
	m.syncLocked()
	assert.Len(t, *m.writable, 1, "the new writable has been synced with the old writable and should have the inserted value")
}
