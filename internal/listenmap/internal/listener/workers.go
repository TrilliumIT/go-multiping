package listener

import (
	"context"
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/listenmap/internal/messages"
	"github.com/TrilliumIT/go-multiping/ping"
)

type procFunc func(
	ctx context.Context,
	r *messages.RecvMsg,
	getCb func(net.IP, uint16) func(context.Context, *ping.Ping),
	done func(),
)

func getProcFunc(ctx context.Context, workers, buffer int) (procFunc, func()) {
	// start workers
	if workers < -1 {
		return func(
			ctx context.Context,
			r *messages.RecvMsg,
			getCb func(net.IP, uint16) func(context.Context, *ping.Ping),
			done func(),
		) {
			processMessage(ctx, r, getCb)
			done()
		}, func() {}
	}

	if workers == 0 {
		return func(
			ctx context.Context,
			r *messages.RecvMsg,
			getCb func(net.IP, uint16) func(context.Context, *ping.Ping),
			done func(),
		) {
			go func() {
				processMessage(ctx, r, getCb)
				done()
			}()
		}, func() {}
	}

	wCh := make(chan *procMsg, buffer)
	wWg := sync.WaitGroup{}
	if workers == -1 {
		return func(
			ctx context.Context,
			r *messages.RecvMsg,
			getCb func(net.IP, uint16) func(context.Context, *ping.Ping),
			done func(),
		) {
			select {
			case wCh <- &procMsg{ctx, r, getCb, done}:
				return
			case <-ctx.Done():
				return
			default:
			}
			wWg.Add(1)
			go func() {
				runWorker(ctx, wCh)
				wWg.Done()
			}()
			select {
			case wCh <- &procMsg{ctx, r, getCb, done}:
				return
			case <-ctx.Done():
				return
			}
		}, wWg.Wait
	}

	for w := 0; w < workers; w++ {
		wWg.Add(1)
		go func() {
			runWorker(ctx, wCh)
			wWg.Done()
		}()
	}

	return func(
		ctx context.Context,
		r *messages.RecvMsg,
		getCb func(net.IP, uint16) func(context.Context, *ping.Ping),
		done func(),
	) {
		select {
		case wCh <- &procMsg{ctx, r, getCb, done}:
			return
		case <-ctx.Done():
			return
		}
	}, wWg.Wait
}

func runWorker(ctx context.Context, wCh <-chan *procMsg) {
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-wCh:
			processMessage(p.ctx, p.r, p.getCb)
			p.done()
		}
	}
}

type procMsg struct {
	ctx   context.Context
	r     *messages.RecvMsg
	getCb func(net.IP, uint16) func(context.Context, *ping.Ping)
	done  func()
}

func processMessage(
	ctx context.Context,
	r *messages.RecvMsg,
	getCb func(net.IP, uint16) func(context.Context, *ping.Ping),
) {
	p := r.ToPing()
	if p == nil {
		return
	}

	cb := getCb(p.Dst, uint16(p.ID))
	if cb == nil {
		return
	}

	cb(ctx, p)
}
