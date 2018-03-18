package process

import (
	"context"
	"net"

	"github.com/TrilliumIT/go-multiping/internal/listenMap/internal/messages"
	"github.com/TrilliumIT/go-multiping/ping"
)

func ProcessMessage(ctx context.Context, r *messages.RecvMsg, getCb func(net.IP, uint16) func(context.Context, *ping.Ping)) {
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
