package pinger

import (
	"context"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/TrilliumIT/go-multiping/ping"
)

type recvMsg struct {
	v4cm       *ipv4.ControlMessage
	v6cm       *ipv6.ControlMessage
	recieved   time.Time
	payload    []byte
	payloadLen int
}

func processMessage(ctx context.Context, lm *listenMap, r *recvMsg) {
	var proto int
	var typ icmp.Type
	p := &ping.Ping{}
	p.Recieved = r.recieved
	if r.v4cm != nil {
		p.Dst = r.v4cm.Src
		p.Src = r.v4cm.Dst
		p.TTL = r.v4cm.TTL
		proto = ping.ProtocolICMP
		typ = ipv4.ICMPTypeEchoReply
	}
	if r.v6cm != nil {
		p.Dst = r.v6cm.Src
		p.Src = r.v6cm.Dst
		p.TTL = r.v6cm.HopLimit
		proto = ping.ProtocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoReply
	}
	if p.Dst == nil {
		return
	}

	if len(r.payload) < r.payloadLen {
		return
	}

	var m *icmp.Message
	var err error
	m, err = icmp.ParseMessage(proto, r.payload[:r.payloadLen])
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

	p.Len = r.payloadLen

	lme.cb(ctx, p)
}
