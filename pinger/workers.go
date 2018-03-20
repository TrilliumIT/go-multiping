package pinger

import (
	"context"

	"github.com/TrilliumIT/go-multiping/ping"
	"github.com/TrilliumIT/go-multiping/pinger/internal/pending"
)

type procPing struct {
	ctx  context.Context
	p    *pending.Ping
	done func()
}

type procFunc func(context.Context, *pending.Ping, func())

// returns the proto func and a worker waitgroup which should be waited on before canceling ctx
func getProcFunc(ctx context.Context, workers, buffer int, m *pending.Map, h func(*ping.Ping, error)) procFunc {
	// start workers
	if workers < -1 {
		return func(ctx context.Context, p *pending.Ping, done func()) {
			p.Wait(ctx, m, h)
			done()
		}
	}

	if workers == 0 {
		return func(ctx context.Context, p *pending.Ping, done func()) {
			go func() {
				p.Wait(ctx, m, h)
				done()
			}()
		}
	}

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
			go func() {
				runWorker(ctx, pCh, m, h)
			}()
			select {
			case <-ctx.Done():
				return
			case pCh <- &procPing{pCtx, p, done}:
			}
		}
	}

	for w := 0; w < workers; w++ {
		go func() {
			runWorker(ctx, pCh, m, h)
		}()
	}

	return func(pCtx context.Context, p *pending.Ping, done func()) {
		select {
		case <-ctx.Done():
			return
		case pCh <- &procPing{pCtx, p, done}:
		}
	}
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
