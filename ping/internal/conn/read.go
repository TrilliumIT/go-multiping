package conn

import (
	"net"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func readV4(c *ipv4.PacketConn, len int) (
	payload []byte,
	srcAddr net.Addr,
	src, dst net.IP,
	rlen, ttl int,
	received time.Time,
	err error,
) {
	payload = make([]byte, len)
	var cm *ipv4.ControlMessage
	rlen, cm, srcAddr, err = c.ReadFrom(payload)
	received = time.Now()
	if cm != nil {
		src, dst, ttl = cm.Src, cm.Dst, cm.TTL
	}
	if src == nil {
		src = net.IPv4zero
	}
	if dst == nil {
		dst = net.IPv4zero
	}
	return
}

func readV6(c *ipv6.PacketConn, len int) (
	payload []byte,
	srcAddr net.Addr,
	src, dst net.IP,
	rlen, ttl int,
	received time.Time,
	err error,
) {
	payload = make([]byte, len)
	var cm *ipv6.ControlMessage
	rlen, cm, srcAddr, err = c.ReadFrom(payload)
	received = time.Now()
	if cm != nil {
		src, dst, ttl = cm.Src, cm.Dst, cm.HopLimit
	}
	if src == nil {
		src = net.IPv6zero
	}
	if dst == nil {
		dst = net.IPv6zero
	}
	return
}
