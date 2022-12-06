package groupsync

import (
	"github.com/panjf2000/ants/v2"
	"sync"
)

type Group[T any] struct {
	collector []T
	channel   chan T

	wg   *sync.WaitGroup
	pool *ants.Pool
}

func NewSendGroup[T any](limit int) (*Group[T], error) {
	pool, err := ants.NewPool(limit, ants.WithPreAlloc(true))
	if err != nil {
		return nil, err
	}
	receiver := (limit / 5) + 1
	group := &Group[T]{
		collector: make([]T, 0),
		pool:      pool,
		channel:   make(chan T, receiver),
		wg:        new(sync.WaitGroup),
	}

	group.receiver(receiver)
	return group, nil
}

func (g *Group[T]) receiver(ss int) {
	for i := 0; i < ss; i++ {
		g.pool.Submit(func() {
			for s := range g.channel {
				g.collector = append(g.collector, s)
			}
		})
	}
}

func (g *Group[T]) Go(fn func(c chan<- T)) error {
	g.wg.Add(1)
	return g.pool.Submit(func() {
		fn(g.channel)
		g.wg.Done()
	})
}

func (g *Group[T]) Wait() []T {
	defer g.pool.Release()
	g.wg.Wait()
	close(g.channel)

	return g.collector
}
