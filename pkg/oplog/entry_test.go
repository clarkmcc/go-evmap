package oplog

import (
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	v := "bar"
	e := Insert("foo", &v)
	fmt.Println(*e.v)
}
