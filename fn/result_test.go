package fn

import (
	"os"
	"testing"
)

func do() Result[*os.File] {
	return Try(os.Open("./somefile"))
}

func TestOk(t *testing.T) {
	r := do()
	r.Match(
		func(v *os.File) (*os.File, error) {
			t.Log(v.Name())
			return nil, v.Close()
		},

		func(e error) (*os.File, error) {
			t.Fatal(e)
			return nil, nil
		},
	)
}
