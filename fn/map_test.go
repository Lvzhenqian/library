package fn

import "testing"

func TestMap_Store(t *testing.T) {
	var m Map[string, string]

	m.Store("a", "1")
	m.Store("b", "2")
	m.Store("c", "4")
	m.Store("a", "3")

	m.Range(func(key, value string) {
		t.Logf("key:%s,value:%s", key, value)
	})
	t.Log(m.Load("b"))
	m.Delete("c")

	m.Range(func(key, value string) {
		t.Logf("key:%s,value:%s", key, value)
	})
}
