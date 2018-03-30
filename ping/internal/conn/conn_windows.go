// see https://github.com/golang/net/blob/master/ipv4/control_windows.go#L14
package conn

import (
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func setupV4Conn(c *ipv4.PacketConn) error {
	return nil
}

func setupV6Conn(c *ipv6.PacketConn) error {
	return nil
}
