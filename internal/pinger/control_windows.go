// see https://github.com/golang/net/blob/master/ipv4/control_windows.go#L14

package pinger

import (
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func setPacketCon(c *icmp.PacketConn) error {
	return nil
}

func listenPacket(c *icmp.PacketConn, r *recvMsg) error {
	var err error
	var src net.IPAddr
	r.payloadLen, src, err = c.ReadFrom(r.payload)
	switch {
	case c.IPv4PacketConn() != nil:
		r.v4cm = &ipv4.ControlMessage{Src: src.IP}
	case c.IPv6PacketConn() != nil:
		r.v6cm = &ipv6.ControlMessage{Src: src.IP}
	}
	return err
}
