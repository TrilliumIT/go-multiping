package pinger

import (
	"sync"
	"time"

	"github.com/clinta/go-multiping/packet"
)

func wrapCallbacks(
	onReply func(*packet.Packet),
	onSend func(*packet.Packet),
	onSendError func(*packet.Packet, error),
	onTimeout func(*packet.Packet),
	stop <-chan struct{},
	sending <-chan struct{},
	wg *sync.WaitGroup,
	timeout time.Duration,
	interval time.Duration,
) (
	func(*packet.Packet), // onReply
	func(*packet.Packet), // onSend
	func(*packet.Packet, error), // onSendError
) {
	buf := 2 * (timeout.Nanoseconds() / interval.Nanoseconds())
	if buf < 2 {
		buf = 2
	}
	pktCh := make(chan *pkt, buf)
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTimer(timeout)
		if !t.Stop() {
			<-t.C
		}
		pending := make(map[uint16]*packet.Packet)
		sendingTrigger := make(chan struct{}, 1)
		wg.Add(1)
		go func() {
			<-sending
			sendingTrigger <- struct{}{}
			wg.Done()
		}()
		for {
			select {
			case p := <-pktCh:
				processPkt(pending, p, t, timeout)
				continue
			case <-stop:
				return
			default:
			}

			select {
			case <-sending:
				if len(pending) == 0 {
					return
				}
			default:
			}

			select {
			case p := <-pktCh:
				processPkt(pending, p, t, timeout)
				continue
			case n := <-t.C:
				processTimeout(pending, t, timeout, onTimeout, n, wg)
			case <-sendingTrigger:
				continue
			case <-stop:
				return
			}
		}
	}()

	rOnSend := func(p *packet.Packet) {
		if onSend != nil {
			wg.Add(1)
			go func() {
				onSend(p)
				wg.Done()
			}()
		}
		pktCh <- &pkt{sent: p}
	}

	var rOnSendError func(*packet.Packet, error)

	if onSendError != nil {
		rOnSendError = func(p *packet.Packet, err error) {
			if onSendError != nil {
				wg.Add(1)
				go func() {
					onSendError(p, err)
					wg.Done()
				}()
			}
			pktCh <- &pkt{err: p}
		}
	}

	rOnReply := func(p *packet.Packet) {
		if p.Sent.Add(timeout).Before(p.Recieved) {
			if onTimeout != nil {
				wg.Add(1)
				go func() {
					onTimeout(p)
					wg.Done()
				}()
			}
		} else {
			if onReply != nil {
				wg.Add(1)
				go func() {
					onReply(p)
					wg.Done()
				}()
			}
		}
		pktCh <- &pkt{recv: p}
	}

	//return onReply, rOnSend, rOnSendError
	return rOnReply, rOnSend, rOnSendError
}

type pkt struct {
	sent *packet.Packet
	recv *packet.Packet
	err  *packet.Packet
}

func processPkt(pending map[uint16]*packet.Packet, p *pkt, t *time.Timer, timeout time.Duration) {
	if p.sent != nil {
		pending[uint16(p.sent.Seq)] = p.sent
		if len(pending) == 1 {
			resetTimer(t, p.sent.Sent, timeout)
		}
	}
	if p.recv != nil {
		delete(pending, uint16(p.recv.Seq))
	}
	if p.err != nil {
		delete(pending, uint16(p.err.Seq))
	}
	if len(pending) == 0 {
		stopTimer(t)
	}
}

func resetTimer(t *time.Timer, s time.Time, d time.Duration) {
	stopTimer(t)
	rd := time.Until(s.Add(d))
	if rd < time.Nanosecond {
		rd = time.Nanosecond
	}
	t.Reset(rd)
}

func stopTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

func processTimeout(pending map[uint16]*packet.Packet, t *time.Timer, timeout time.Duration, onTimeout func(*packet.Packet), n time.Time, wg *sync.WaitGroup) {
	var resetS time.Time
	for s, p := range pending {
		if p.Sent.Add(timeout).Before(n) {
			if onTimeout != nil {
				wg.Add(1)
				go func(p *packet.Packet) {
					defer wg.Done()
					onTimeout(p)
				}(p)
			}
			delete(pending, s)
			continue
		}
		if resetS.IsZero() || resetS.After(p.Sent) {
			resetS = p.Sent
		}
	}

	if !resetS.IsZero() {
		resetTimer(t, resetS, timeout)
	}
}
