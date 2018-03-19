// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package listener

import (
	"fmt"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/TrilliumIT/go-multiping/internal/listenmap/internal/messages"
)

func setPacketCon(c *icmp.PacketConn) error {
	var err error
	switch {
	case c.IPv4PacketConn() != nil:
		err = c.IPv4PacketConn().SetControlMessage(ipv4.FlagDst|ipv4.FlagSrc|ipv4.FlagTTL, true)
	case c.IPv6PacketConn() != nil:
		err = c.IPv6PacketConn().SetControlMessage(ipv6.FlagDst|ipv6.FlagSrc|ipv6.FlagHopLimit, true)
	default:
		err = fmt.Errorf("no valid connections")
	}
	return err
}

func readPacket(c *icmp.PacketConn, r *messages.RecvMsg) error {
	var err error
	switch {
	case c.IPv4PacketConn() != nil:
		r.PayloadLen, r.V4cm, _, err = c.IPv4PacketConn().ReadFrom(r.Payload)
	case c.IPv6PacketConn() != nil:
		r.PayloadLen, r.V6cm, _, err = c.IPv6PacketConn().ReadFrom(r.Payload)
	}
	return err
}
