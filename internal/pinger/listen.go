package pinger

import (
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

func (p *Pinger) listen() (func() (error, func()), error) {
	var err error
	var wg sync.WaitGroup
	retF := func() (error, func()) { return nil, func() {} }

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

	pwg := sync.WaitGroup{}
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
				pwg.Add(1)
				go func() {
					p.processMessage(r)
					pwg.Done()
				}()
			}
		}
	}()

	retF = func() (error, func()) {
		err := <-ech
		wg.Wait()
		return err, pwg.Wait
	}

	swg.Wait()
	return retF, nil
}
