package pinger

import (
	"context"

	"golang.org/x/net/icmp"

	"github.com/TrilliumIT/go-multiping/internal/messages"
	"github.com/TrilliumIT/go-multiping/ping"
)

func processMessage(ctx context.Context, lm *listenMap, r *messages.RecvMsg) {
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

	lme, ok := lm.get(p.Dst, p.ID)
	if !ok {
		return
	}

	if lme == nil {
		return
	}

	if lme.cb == nil {
		return
	}

	p.Len = r.PayloadLen

	lme.cb(ctx, p)
}
