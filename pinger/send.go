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

	err := d.pinger.AddCallBack(d.dst.IP, e.ID, d.onReply)
	if err != nil {
		fmt.Println("err adding callback")
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
		err := d.send(m)
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

func (d *Dst) send(m *icmp.Message) error {
	e, ok := m.Body.(*icmp.Echo)
	if !ok {
		return fmt.Errorf("invalid icmp message")
	}

	var err error
	e.Data = packet.TimeToBytes(time.Now())
	b, err := m.Marshal(nil)
	if err != nil {
		return err
	}

	fmt.Printf("%p\n", d.dst)
	fmt.Printf("%#v\n", d.dst)
	fmt.Printf("%s\n", d.dst.String())
	fmt.Printf("%s\n", d.dst.Network())
	for {
		_, err = d.pinger.Conn.WriteTo(b, d.dst)
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
