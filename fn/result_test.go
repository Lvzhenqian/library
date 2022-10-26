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
	if r.IsErr() {
		t.Fatal(r.Err())
	}
	file := r.Some()
	defer file.Close()
	t.Log(file.Name())
}
