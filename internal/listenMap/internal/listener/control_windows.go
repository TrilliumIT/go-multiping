// see https://github.com/golang/net/blob/master/ipv4/control_windows.go#L14

package listener

import (
	"fmt"
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func setPacketCon(c *icmp.PacketConn) error {
	var err error
	switch {
	case c.IPv4PacketConn() != nil:
	case c.IPv6PacketConn() != nil:
	default:
		err = fmt.Errorf("no valid connections")
	}
	return err
}

func readPacket(c *icmp.PacketConn, r *recvMsg) error {
	var err error
	switch {
	case c.IPv4PacketConn() != nil:
		r.v4cm = &ipv4.ControlMessage{
			Src: net.IPv4zero,
		}
		r.payloadLen, _, _, err = c.IPv4PacketConn().ReadFrom(r.payload)
	case c.IPv6PacketConn() != nil:
		r.v6cm = &ipv6.ControlMessage{
			Src: net.IPv6zero,
		}
		r.payloadLen, _, _, err = c.IPv6PacketConn().ReadFrom(r.payload)
	}
	return err
}
