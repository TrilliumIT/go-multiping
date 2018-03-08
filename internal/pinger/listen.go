package pinger

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (p *Pinger) createListener() error {
	var err error
	p.Conn, err = icmp.ListenPacket(p.network, p.src)
	if err != nil {
		return err
	}
	if p.Conn.IPv4PacketConn() != nil {
		err = p.Conn.IPv4PacketConn().SetControlMessage(ipv4.FlagDst|ipv4.FlagSrc|ipv4.FlagTTL, true)
		if err != nil {
			return err
		}
	}

	if p.Conn.IPv6PacketConn() != nil {
		err = p.Conn.IPv6PacketConn().SetControlMessage(ipv6.FlagDst|ipv6.FlagSrc|ipv6.FlagHopLimit, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pinger) listen() error {
	var wg sync.WaitGroup
	// ipv4 listener
	wg.Add(1)
	go func() {
		fmt.Println("Starting v4 listener")
		defer fmt.Println("quitting v4 listener")
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
				fmt.Println("Waiting on v4 packet")
				r.payloadLen, r.v4cm, _, err = p.Conn.IPv4PacketConn().ReadFrom(b)
				fmt.Println("v4 packet recieved")
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
		fmt.Println("Starting v6 listener")
		defer fmt.Println("quitting v6 listener")
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
