package groupsync

import (
	"github.com/panjf2000/ants/v2"
	"sync"
)

type Options func(group *Option)

type Option struct {
	worker        int
	receiver      int
	limit         int
	channelBuffer int
}

type Group[T any] struct {
	collector *[]T
	channel   chan T

	wg   *sync.WaitGroup
	pool *ants.Pool
}

func NewGroup[T any](collector *[]T, opt ...Options) (*Group[T], error) {
	var err error
	group := &Group[T]{
		collector: collector,
		wg:        new(sync.WaitGroup),
	}
	option := &Option{
		worker:   10,
		receiver: 3,
	}
	if option.limit < option.worker+option.receiver {
		option.limit = option.worker + option.receiver
	}
	group.channel = make(chan T, option.channelBuffer)

	for _, fn := range opt {
		fn(option)
	}
	group.pool, err = ants.NewPool(option.limit, ants.WithPreAlloc(true))
	if err != nil {
		return nil, err
	}

	group.startReceiver(option)
	return group, nil
}

func (g *Group[T]) startReceiver(opt *Option) {
	for i := 0; i < opt.receiver; i++ {
		g.pool.Submit(func() {
			for s := range g.channel {
				*g.collector = append(*g.collector, s)
			}
		})
	}
}

func (g *Group[T]) Go(fn func() T) error {
	g.wg.Add(1)
	return g.pool.Submit(func() {
		g.channel <- fn()
		g.wg.Done()
	})
}

func (g *Group[T]) Wait() {
	defer g.pool.Release()
	g.wg.Wait()
	close(g.channel)

	return
}

func WithLimit(limit int) Options {
	return func(opt *Option) {
		opt.limit = limit
	}
}

func WithWorker(worker int) Options {
	return func(opt *Option) {
		opt.worker = worker
	}
}

func WithReceivers(recv int) Options {
	return func(opt *Option) {
		opt.receiver = recv
	}
}

func WithChannelBuffer(size int) Options {
	return func(opt *Option) {
		opt.channelBuffer = size
	}
}
