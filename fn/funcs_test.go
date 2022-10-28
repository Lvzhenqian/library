package fn

import (
	"fmt"
	"testing"
)

func TestSliceTurning(t *testing.T) {
	size := 53
	tt := make([]string, size)
	for i := 0; i < size; i++ {
		tt[i] = fmt.Sprintf("%d", i+1)
	}

	cut := SliceTurning(tt, 3)
	for item := range cut {
		t.Log(item)
	}
}

func TestToString(t *testing.T) {
	b := []byte("hello")
	t.Log(b)
	s := ToString(b)
	t.Log(s)
	bt := ToBytes(s)
	t.Log(cap(b) == cap(bt))
}
