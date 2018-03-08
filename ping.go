package ping

import (
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

func (p *protoPinger) listen() error {
	var wg sync.WaitGroup
	conn, err := icmp.ListenPacket(p.network, p.src)
	if err != nil {
		return err
	}

	// ipv4 listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-p.stop:
				return
			default:
				if conn.IPv4PacketConn() == nil {
					return
				}
				r := &recvMsg{}
				b := []byte{}
				var err error
				r.payloadLen, r.v4cm, _, err = conn.IPv4PacketConn().ReadFrom(b)
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
				if conn.IPv6PacketConn() == nil {
					return
				}
				r := &recvMsg{}
				var err error
				r.payloadLen, r.v6cm, _, err = conn.IPv6PacketConn().ReadFrom(r.payload)
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
