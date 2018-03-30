// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package conn

import (
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func setupV4Conn(c *ipv4.PacketConn) error {
	err := c.SetControlMessage(ipv4.FlagDst|ipv4.FlagSrc|ipv4.FlagTTL, true)
	if err != nil {
		return err
	}

	var f ipv4.ICMPFilter
	f.SetAll(true)
	f.Accept(ipv4.ICMPTypeEchoReply)
	err = c.SetICMPFilter(&f)
	return err
}

func setupV6Conn(c *ipv6.PacketConn) error {
	err := c.SetControlMessage(ipv6.FlagDst|ipv6.FlagSrc|ipv6.FlagHopLimit, true)
	if err != nil {
		return err
	}
	var f ipv6.ICMPFilter
	f.SetAll(true)
	f.Accept(ipv6.ICMPTypeEchoReply)
	err = c.SetICMPFilter(&f)
	return err
}
