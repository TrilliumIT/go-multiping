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

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func (d *Dst) Run() error {
	e := &icmp.Echo{
		ID:   rand.Intn(1<<16 - 1),
		Seq:  0,
		Data: packet.TimeToBytes(time.Now()),
	}
	m := &icmp.Message{
		Type: d.pinger.SendType(),
		Code: 0,
		Body: e,
	}

	onReply, onSend := wrapCallbacks(
		d.onReply, d.onSend, d.onTimeout,
		d.stop, d.timeout, d.interval)

	err := d.pinger.AddCallBack(d.dst.IP, e.ID, onReply)
	if err != nil {
		return err
	}
	defer d.pinger.DelCallBack(d.dst.IP, e.ID)

	t := make(chan struct{})
	go func() {
		ti := time.NewTicker(d.interval)
		defer ti.Stop()
		t <- struct{}{}
		for {
			select {
			case <-ti.C:
				t <- struct{}{}
			case <-d.stop:
				close(t)
				return
			}
		}
	}()

	for range t {
		err := d.send(m, onSend)
		if err != nil {
			return err
		}
		e.Seq += 1
		if d.count > 0 && e.Seq >= d.count {
			time.Sleep(d.timeout)
			break
		}
	}

	select {
	case <-d.stop:
	default:
		d.Stop()
	}

	return nil
}

func (d *Dst) send(m *icmp.Message, onSend func(*packet.SentPacket)) error {
	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		return fmt.Errorf("invalid icmp message")
	}

	var err error
	t := time.Now()
	e.Data = packet.TimeToBytes(t)
	b, err := m.Marshal(nil)
	if err != nil {
		return err
	}

	for {
		_, err := d.pinger.Conn.WriteTo(b, d.dst)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
		}
		if onSend != nil {
			onSend(&packet.SentPacket{
				Dst:  d.dst.IP,
				ID:   e.ID,
				Seq:  e.Seq,
				Sent: t,
			})
		}
		break
	}
	return err
}
