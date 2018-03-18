package process

import (
	"context"
	"net"

	"golang.org/x/net/icmp"

	"github.com/TrilliumIT/go-multiping/internal/listenMap/internal/messages"
	"github.com/TrilliumIT/go-multiping/ping"
)

func ProcessMessage(ctx context.Context, r *messages.RecvMsg, getCb func(net.IP, uint16) func(context.Context, *ping.Ping)) {
	var proto int
	var typ icmp.Type
	p := &ping.Ping{}
	p.Recieved = r.Recieved
	p.Dst, p.Src, p.TTL, proto, typ = r.Props()
	if p.Dst == nil {
		return
	}

	if len(r.Payload) < r.PayloadLen {
		return
	}

	var m *icmp.Message
	var err error
	m, err = icmp.ParseMessage(proto, r.Payload[:r.PayloadLen])
	if err != nil {
		return
	}
	if m.Type != typ {
		return
	}

	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		return
	}
	p.ID = e.ID
	p.Seq = e.Seq

	p.Sent, err = ping.BytesToTime(e.Data)
	if err != nil {
		return
	}

	p.Len = r.PayloadLen

	cb := getCb(p.Dst, uint16(p.ID))
	if cb == nil {
		return
	}

	cb(ctx, p)
}
