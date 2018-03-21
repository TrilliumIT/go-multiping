package socket

import (
	"net"
)

type Conn struct {
	dst *net.IPAddr
}
