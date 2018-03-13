package pinger

import (
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (p *Pinger) listen() (func() error, error) {
	var err error
	var wg sync.WaitGroup
	retF := func() error { return nil }

	p.Conn, err = icmp.ListenPacket(p.network, p.src)
	if err != nil {
		return retF, err
	}

	if p.Conn.IPv4PacketConn() != nil {
		err = p.Conn.IPv4PacketConn().SetControlMessage(ipv4.FlagDst|ipv4.FlagSrc|ipv4.FlagTTL, true)
		if err != nil {
			return retF, err
		}
	}

	if p.Conn.IPv6PacketConn() != nil {
		err = p.Conn.IPv6PacketConn().SetControlMessage(ipv6.FlagDst|ipv6.FlagSrc|ipv6.FlagHopLimit, true)
		if err != nil {
			return retF, err
		}
	}

	var swg sync.WaitGroup

	// close conn on stop
	ech := make(chan error)
	wg.Add(1)
	swg.Add(1)
	go func() {
		defer wg.Done()
		swg.Done()
		<-p.stop
		ech <- p.Conn.Close()
	}()

	retF = func() error {
		err := <-ech
		wg.Wait()
		return err
	}

	// ipv4 listener
	wg.Add(1)
	swg.Add(1)
	go func() {
		defer wg.Done()
		swg.Done()
		for {
			select {
			case <-p.stop:
				return
			default:
				r := &recvMsg{
					payload: make([]byte, p.expectedLen),
				}
				var err error
				if p.Conn.IPv6PacketConn() != nil {
					r.payloadLen, r.v6cm, _, err = p.Conn.IPv6PacketConn().ReadFrom(r.payload)
				}
				if p.Conn.IPv4PacketConn() != nil {
					r.payloadLen, r.v4cm, _, err = p.Conn.IPv4PacketConn().ReadFrom(r.payload)
				}
				if err != nil {
					continue
				}
				r.recieved = time.Now()
				go p.processMessage(r)
			}
		}
	}()

	swg.Wait()
	return retF, nil
}
