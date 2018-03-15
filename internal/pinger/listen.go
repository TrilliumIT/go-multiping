package pinger

import (
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

func (p *Pinger) listen() (func() error, error) {
	var err error
	var wg sync.WaitGroup
	retF := func() error { return nil }

	p.Conn, err = icmp.ListenPacket(p.network, p.src)
	if err != nil {
		return retF, err
	}

	err = setPacketCon(p.Conn)
	if err != nil {
		return retF, err
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

	rCh := make(chan *recvMsg, p.pktChLen)
	for w := 0; w <= p.workers; w++ {
		wg.Add(1)
		swg.Add(1)
		go func() {
			defer wg.Done()
			swg.Done()
			for r := range rCh {
				p.processMessage(r)
			}
		}()
	}

	lwg := sync.WaitGroup{}
	for w := 0; w < p.workers; w++ {
		swg.Add(1)
		lwg.Add(1)
		go func() {
			defer lwg.Done()
			swg.Done()
			for {
				select {
				case <-p.stop:
					return
				default:
					r := &recvMsg{
						payload: make([]byte, p.expectedLen),
					}
					r.err = readPacket(p.Conn, r)
					r.recieved = time.Now()
					select {
					case rCh <- r:
					default:
						// workers blocked
						go p.processMessage(r)
					}
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		lwg.Wait()
		close(rCh)
	}()

	swg.Wait()
	return retF, nil
}
