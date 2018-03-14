package pinger

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func setPacketCon(c *icmp.PacketConn) error {
	return nil
}

func listenPacket(c *icmp.PacketConn, r *recvMsg) error {
	var err error
	switch {
	case c.IPv4PacketConn() != nil:
		r.v4cm = &ipv4.ControlMessage{}
		r.payloadLen, r.v4cm.Src, err = c.ReadFrom(r.payload)
	case c.IPv6PacketConn() != nil:
		r.v6cm = &ipv6.ControlMessage{}
		r.payloadLen, r.v6cm.Src, err = c.ReadFrom(r.payload)
	}
	return err
}
