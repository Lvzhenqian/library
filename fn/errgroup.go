package fn

import (
	"context"
	"sync"
)

type ErrGroup struct {
	cancel  func()
	wg      *sync.WaitGroup
	errs    []error
	errChan chan error
}

func WithContext(ctx context.Context) (*ErrGroup, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	eg := &ErrGroup{
		cancel:  cancel,
		wg:      new(sync.WaitGroup),
		errs:    make([]error, 0),
		errChan: make(chan error),
	}
	go eg.errRec(ctx)
	return eg, ctx
}

func (g *ErrGroup) errRec(ctx context.Context) {
	for {
		select {
		case err := <-g.errChan:
			g.errs = append(g.errs, err)
		case <-ctx.Done():
			return
		}
	}
}

func (g *ErrGroup) Wait() []error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.errs
}

func (g *ErrGroup) Go(fn func() error) {
	g.wg.Add(1)
	go func() {
		defer g.done()
		if err := fn(); err != nil {
			g.errChan <- err
		}
	}()
}

func (g *ErrGroup) done() {
	g.wg.Done()
}
