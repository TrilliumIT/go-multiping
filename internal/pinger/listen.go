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
				err := readPacket(p.Conn, r)
				if err != nil {
					continue
				}
				r.recieved = time.Now()
				wg.Add(1)
				go func() {
					p.processMessage(r)
					wg.Done()
				}()
			}
		}
	}()

	retF = func() error {
		err := <-ech
		wg.Wait()
		return err
	}

	swg.Wait()
	return retF, nil
}
