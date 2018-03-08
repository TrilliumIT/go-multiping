package pinger

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"golang.org/x/net/icmp"

	"github.com/clinta/go-multiping/packet"
)

func (d *Dst) Run() error {
	e := &icmp.Echo{
		ID:  rand.Intn(65535),
		Seq: 0,
	}
	m := &icmp.Message{
		Type: d.pinger.SendType(),
		Code: 0,
		Body: e,
	}

	err := d.pinger.AddCallBack(d.dst, e.ID, d.onReply)
	if err != nil {
		return err
	}
	defer d.pinger.DelCallBack(d.dst, e.ID)

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
		err := d.send(m)
		if err != nil {
			return err
		}
		e.Seq += 1
		if e.Seq > d.count {
			time.Sleep(d.timeout)
			select {
			case <-d.stop:
			default:
				d.Stop()
			}
		}
	}

	return nil
}

func (d *Dst) send(m *icmp.Message) error {
	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		return fmt.Errorf("invalid icmp message")
	}

	var dAddr *net.IPAddr
	var err error
	dAddr, err = net.ResolveIPAddr("ip", d.dst.String())
	if err != nil {
		return err
	}

	e.Data = packet.TimeToBytes(time.Now())
	b, err := m.Marshal(nil)
	if err != nil {
		return err
	}

	_, err = d.pinger.Conn.WriteTo(b, dAddr)
	return err
}
