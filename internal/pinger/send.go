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

func (pp *Pinger) Send(dst *net.IPAddr, m *icmp.Message, onSend func(*packet.SentPacket), onSendError func(*packet.SentPacket, error)) error {
	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		return fmt.Errorf("invalid icmp message")
	}

	for {
		t := time.Now()
		e.Data = packet.TimeToBytes(t)
		b, err := m.Marshal(nil)
		if err != nil {
			return err
		}

		sp := &packet.SentPacket{
			Dst:  dst.IP,
			ID:   e.ID,
			Seq:  e.Seq,
			Sent: t,
		}
		if onSend != nil {
			onSend(sp)
		}

		_, err = pp.Conn.WriteTo(b, dst)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
			if onSendError != nil {
				onSendError(sp, err)
			} else {
				return err
			}
		}
		break
	}
	return nil
}
