package pinger

import (
	"net"
	"syscall"
	"time"

	"github.com/clinta/go-multiping/packet"
)

func (pp *Pinger) Send(dst *net.IPAddr, sp *packet.Packet) error {
	var err error
	for {
		sp.Sent = time.Now()
		var b []byte
		b, err = sp.ToICMPMsg()
		if err != nil {
			break
		}

		_, err = pp.Conn.WriteTo(b, dst)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
		}
		break
	}
	return err
}
