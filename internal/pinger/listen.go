package pinger

import (
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (p *Pinger) listen() (func(), error) {
	var err error
	var wg sync.WaitGroup

	p.Conn, err = icmp.ListenPacket(p.network, p.src)
	if err != nil {
		return wg.Done, err
	}
	if p.Conn.IPv4PacketConn() != nil {
		err = p.Conn.IPv4PacketConn().SetControlMessage(ipv4.FlagDst|ipv4.FlagSrc|ipv4.FlagTTL, true)
		if err != nil {
			return wg.Done, err
		}
	}

	if p.Conn.IPv6PacketConn() != nil {
		err = p.Conn.IPv6PacketConn().SetControlMessage(ipv6.FlagDst|ipv6.FlagSrc|ipv6.FlagHopLimit, true)
		if err != nil {
			return wg.Done, err
		}
	}

	var swg sync.WaitGroup

	// close conn on stop
	wg.Add(1)
	swg.Add(1)
	go func() {
		defer wg.Done()
		swg.Done()
		<-p.stop
		p.Conn.Close()
	}()

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
				if p.Conn.IPv4PacketConn() == nil {
					return
				}
				r := &recvMsg{
					payload: make([]byte, p.expectedLen, p.expectedLen),
				}
				var err error
				r.payloadLen, r.v4cm, _, err = p.Conn.IPv4PacketConn().ReadFrom(r.payload)
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
	swg.Add(1)
	go func() {
		defer wg.Done()
		swg.Done()
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

	swg.Wait()
	return wg.Wait, nil
}
