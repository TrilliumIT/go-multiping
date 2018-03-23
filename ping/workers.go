package ping

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
	"github.com/TrilliumIT/go-multiping/pinger/ping/internal/pending"
)

type procPing struct {
	ctx  context.Context
	p    *pending.Ping
	done func()
}

type procFunc func(context.Context, *pending.Ping, func())

// returns the proto func and a worker waitgroup which should be waited on before canceling ctx
func getProcFunc(ctx context.Context, workers, buffer int, m *pending.Map, h func(*ping.Ping, error)) (procFunc, func()) {
	// start workers
	if workers < -1 {
		return func(ctx context.Context, p *pending.Ping, done func()) {
			p.Wait(ctx, m, h)
			done()
		}, func() {}
	}

	if workers == 0 {
		return func(ctx context.Context, p *pending.Ping, done func()) {
			go func() {
				p.Wait(ctx, m, h)
				done()
			}()
		}, func() {}
	}

	workersRunning := sync.WaitGroup{}
	pCh := make(chan *procPing, buffer)
	if workers == -1 {
		return func(pCtx context.Context, p *pending.Ping, done func()) {
			select {
			case <-ctx.Done():
				return
			case pCh <- &procPing{pCtx, p, done}:
				return
			default:
			}
			workersRunning.Add(1)
			go func() {
				runWorker(ctx, pCh, m, h)
				workersRunning.Done()
			}()
			select {
			case <-ctx.Done():
				return
			case pCh <- &procPing{pCtx, p, done}:
			}
		}, workersRunning.Wait
	}

	for w := 0; w < workers; w++ {
		workersRunning.Add(1)
		go func() {
			runWorker(ctx, pCh, m, h)
			workersRunning.Done()
		}()
	}

	return func(pCtx context.Context, p *pending.Ping, done func()) {
		select {
		case <-ctx.Done():
			return
		case pCh <- &procPing{pCtx, p, done}:
		}
	}, workersRunning.Wait
}

func runWorker(ctx context.Context, pCh <-chan *procPing, m *pending.Map, h func(*ping.Ping, error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-pCh:
			p.p.Wait(p.ctx, m, h)
			p.done()
		}
	}
}
