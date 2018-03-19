package messages

import (
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/TrilliumIT/go-multiping/ping"
)

// RecvMsg is a message recieved from a socket listener
type RecvMsg struct {
	V4cm       *ipv4.ControlMessage
	V6cm       *ipv6.ControlMessage
	Recieved   time.Time
	Payload    []byte
	PayloadLen int
}

// ToPing turns a RecvMsg into a ping
func (r *RecvMsg) ToPing() *ping.Ping {
	var proto int
	var typ icmp.Type
	p := &ping.Ping{
		Recieved: r.Recieved,
		Len:      r.PayloadLen,
	}
	switch {
	case r.V4cm != nil:
		p.Src = r.V4cm.Dst
		p.Dst = r.V4cm.Src
		p.TTL = r.V4cm.TTL
		proto = ping.ProtocolICMP
		typ = ipv4.ICMPTypeEchoReply
	case r.V6cm != nil:
		p.Src = r.V6cm.Dst
		p.Dst = r.V6cm.Src
		p.TTL = r.V6cm.HopLimit
		proto = ping.ProtocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoReply
	}

	if len(r.Payload) < r.PayloadLen {
		return nil
	}

	var m *icmp.Message
	var err error
	m, err = icmp.ParseMessage(proto, r.Payload[:r.PayloadLen])
	if err != nil {
		return nil
	}
	if m.Type != typ {
		return nil
	}

	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		return nil
	}
	p.ID = e.ID
	p.Seq = e.Seq

	p.Sent, err = ping.BytesToTime(e.Data)
	if err != nil {
		return nil
	}

	return p
}
