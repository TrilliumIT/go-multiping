package pinger

import (
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

func (p *Pinger) createListener() error {
	var err error
	p.Conn, err = icmp.ListenPacket(p.network, p.src)
	return err
}

func (p *Pinger) listen() error {
	var wg sync.WaitGroup
	// ipv4 listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-p.stop:
				return
			default:
				if p.Conn.IPv4PacketConn() == nil {
					return
				}
				r := &recvMsg{}
				b := []byte{}
				var err error
				r.payloadLen, r.v4cm, _, err = p.Conn.IPv4PacketConn().ReadFrom(b)
				if err != nil {
					continue
				}
				r.recieved = time.Now()
				go p.processMessage(r)
			}
		}
	}()

	// ipv6 listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-p.stop:
				return
			default:
				if p.Conn.IPv6PacketConn() == nil {
					return
				}
				r := &recvMsg{}
				var err error
				r.payloadLen, r.v6cm, _, err = p.Conn.IPv6PacketConn().ReadFrom(r.payload)
				if err != nil {
					continue
				}
				r.recieved = time.Now()
				go p.processMessage(r)
			}
		}
	}()

	wg.Wait()
	return nil
}
