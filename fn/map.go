package fn

import (
	"sync"
	"sync/atomic"
)

type Map[K comparable, V any] struct {
	data map[K]V

	readonly atomic.Value
	lock     sync.Mutex
}

func (r *Map[K, V]) Store(key K, value V) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if len(r.data) == 0 {
		r.data = make(map[K]V)
	}

	r.data[key] = value

	m, _ := r.readonly.Load().(map[K]V)
	if m != nil {
		m[key] = value
		r.readonly.Store(m)
	} else {
		r.readonly.Store(map[K]V{
			key: value,
		})
	}

}

func (r *Map[K, V]) loadSource(key K) (V, bool) {
	for {
		if r.lock.TryLock() {
			value, ok := r.data[key]
			r.lock.Unlock()
			return value, ok
		}
	}
}

func (r *Map[K, V]) Load(key K) (V, bool) {
	m, _ := r.readonly.Load().(map[K]V)
	if m == nil {
		value, ok := r.loadSource(key)
		if !ok {
			return value, false
		}
		r.readonly.Store(map[K]V{
			key: value,
		})
		return value, true
	}

	val, ok := m[key]
	if !ok {
		val, ok = r.loadSource(key)
		if !ok {
			return val, false
		}
		m[key] = val
		r.readonly.Store(m)
	}
	return val, ok
}

func (r *Map[K, V]) Range(fn func(key K, value V)) {
	m, _ := r.readonly.Load().(map[K]V)
	if m != nil {
		for k, v := range m {
			fn(k, v)
		}
	} else {
		r.lock.Lock()
		for k, v := range r.data {
			fn(k, v)
		}
		r.lock.Unlock()
	}
}

func (r *Map[K, V]) Delete(key K) V {
	m, _ := r.readonly.Load().(map[K]V)
	if m != nil {
		value, ok := m[key]
		if !ok {
			value, ok = r.loadSource(key)
			if ok {
				r.lock.Lock()
				delete(r.data, key)
				r.lock.Unlock()
			}
		}
		delete(m, key)
		r.readonly.Store(m)
		return value
	}

	value, ok := r.loadSource(key)
	if ok {
		r.lock.Lock()
		delete(r.data, key)
		r.lock.Unlock()
		return value
	}
	return value
}
