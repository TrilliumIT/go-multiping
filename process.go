package ping

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func processMessage(r *recvMsg, onReply func(*Packet), dst map[string]struct{}) {
	var proto int
	var typ icmp.Type
	p := &Packet{}
	p.Recieved = r.recieved
	if r.v4cm != nil {
		p.Src = r.v4cm.Src
		p.Dst = r.v4cm.Dst
		p.TTL = r.v4cm.TTL
		proto = ProtocolICMP
		typ = ipv4.ICMPTypeEchoReply
	}
	if r.v6cm != nil {
		p.Src = r.v6cm.Src
		p.Dst = r.v6cm.Dst
		p.TTL = r.v6cm.HopLimit
		proto = ProtocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoReply
	}
	if p.Dst == nil {
		return
	}

	if len(r.payload) < r.lenPayload {
		// Bad packet, skip
	}
	var m *icmp.Message
	var err error
	m, err = icmp.ParseMessage(proto, r.payload[:r.lenPayload])
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

	if _, ok := dst[dstStr(p.Src, p.ID)]; !ok {
		// this is not our packet
		return
	}

	if len(e.Data) < timeSliceLength {
		return
	}

	p.Sent = bytesToTime(e.Data[:timeSliceLength])
	p.RTT = p.Recieved.Sub(p.Sent)

	onReply(p)
}
