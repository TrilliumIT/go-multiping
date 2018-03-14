package pinger

import (
	"net"
	"syscall"
	"time"

	"github.com/clinta/go-multiping/packet"
)

// Send sends a packet. This also sets the sent time on the packet
// nolint: interfacer
// I want only an IPAddr not net.Addr others will throw erros
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
