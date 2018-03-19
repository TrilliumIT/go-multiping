package listener

import (
	"context"
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/listenmap/internal/messages"
	"github.com/TrilliumIT/go-multiping/ping"
)

func getProcFunc(ctx context.Context, workers, buffer int, wWg *sync.WaitGroup) func(*procMsg) {
	// start workers
	proc := processMessage
	if workers == 0 {
		proc = func(p *procMsg) {
			wWg.Add(1)
			go func() {
				processMessage(p)
				wWg.Done()
			}()
		}
	}

	if workers == -1 || workers > 0 {
		wCh := make(chan *procMsg, buffer)
		if workers == -1 {
			proc = func(p *procMsg) {
				select {
				case wCh <- p:
					return
				default:
				}
				wWg.Add(1)
				go func() {
					runWorker(ctx, wCh)
					wWg.Done()
				}()
				wCh <- p
			}
		}
		if workers > 0 {
			proc = func(p *procMsg) {
				wCh <- p
			}
		}
		for w := 0; w < workers; w++ {
			wWg.Add(1)
			go func() {
				runWorker(ctx, wCh)
				wWg.Done()
			}()
		}
	}
	return proc
}

func runWorker(ctx context.Context, wCh <-chan *procMsg) {
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-wCh:
			processMessage(p)
		}
	}
}

type procMsg struct {
	ctx   context.Context
	r     *messages.RecvMsg
	getCb func(net.IP, uint16) func(context.Context, *ping.Ping)
}

func processMessage(pm *procMsg) {
	p := pm.r.ToPing()
	if p == nil {
		return
	}

	cb := pm.getCb(p.Dst, uint16(p.ID))
	if cb == nil {
		return
	}

	cb(pm.ctx, p)
}
