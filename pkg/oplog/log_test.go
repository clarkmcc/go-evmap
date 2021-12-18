package oplog

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLog(t *testing.T) {
	log := NewLog[string, int]()
	m := map[string]*int{}

	// Each of these tests piggyback on each other and cannot be run separately
	t.Run("Insert", func(t *testing.T) {
		v1 := 1
		v2 := 2
		log.Push(Insert("foo", &v1))
		log.Push(Insert("bar", &v2))
		log.Apply(&m)
		log.Clear()

		assert.Len(t, m, 2)
		assert.Equal(t, v1, *m["foo"])
	})
	t.Run("Delete", func(t *testing.T) {
		log.Push(Delete[string, int]("foo"))
		log.Apply(&m)
		log.Clear()

		assert.Len(t, m, 1)
	})
	t.Run("Clear", func(t *testing.T) {
		log.Push(Clear[string, int]())
		log.Apply(&m)

		assert.Len(t, m, 0)
	})
	t.Run("PushAndApply", func(t *testing.T) {
		v1 := 1
		log.PushAndApply(Insert("foo", &v1), &m)
		assert.Len(t, m, 1)
	})
}
