package pinger

import (
	"fmt"
	"math/rand"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/icmp"

	"github.com/clinta/go-multiping/packet"
)

// NewEcho returns a new icmp echo and icmp message
func NewEcho() (*icmp.Echo, *icmp.Message) {
	e := &icmp.Echo{
		ID:   rand.Intn(1<<16 - 1),
		Seq:  0,
		Data: packet.TimeToBytes(time.Now()),
	}
	return e, &icmp.Message{
		Code: 0,
		Body: e,
	}
}

func (pp *Pinger) Send(dst *net.IPAddr, m *icmp.Message) (*packet.Packet, error) {
	sp := &packet.Packet{
		Dst:  dst.IP,
		Sent: time.Now(),
	}

	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		return sp, fmt.Errorf("invalid icmp message")
	}

	sp.ID = e.ID
	sp.Seq = e.Seq

	var err error
	for {
		sp.Sent = time.Now()
		e.Data = packet.TimeToBytes(sp.Sent)
		var b []byte
		b, err = m.Marshal(nil)
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
	return sp, err
}
