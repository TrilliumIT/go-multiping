package pinger

import (
	"time"

	"github.com/clinta/go-multiping/packet"
)

func wrapCallbacks(
	onReply func(*packet.Packet),
	onSend func(*packet.SentPacket),
	onSendError func(*packet.SentPacket, error),
	onTimeout func(*packet.SentPacket),
	stop <-chan struct{},
	timeout time.Duration,
	interval time.Duration,
) (
	func(*packet.Packet),
	func(*packet.SentPacket),
	func(*packet.SentPacket, error),
) {
	if onTimeout == nil {
		return onReply, onSend, onSendError
	}

	buf := 2 * (timeout.Nanoseconds() / interval.Nanoseconds())
	if buf < 2 {
		buf = 2
	}
	pktCh := make(chan *pkt, buf)
	go func() {
		t := time.NewTimer(timeout)
		if !t.Stop() {
			<-t.C
		}
		pending := make(map[int]*packet.SentPacket)
		for {
			select {
			case p := <-pktCh:
				processPkt(pending, p, t, timeout, onTimeout)
				continue
			case <-stop:
				return
			default:
			}

			select {
			case p := <-pktCh:
				processPkt(pending, p, t, timeout, onTimeout)
				continue
			case n := <-t.C:
				processTimeout(pending, t, timeout, onTimeout, n)
			case <-stop:
				return
			}
		}
	}()

	rOnSend := func(p *packet.SentPacket) {
		if onSend != nil {
			go onSend(p)
		}
		pktCh <- &pkt{sent: p}
	}

	var rOnSendError func(*packet.SentPacket, error)

	if onSendError != nil {
		rOnSendError = func(p *packet.SentPacket, err error) {
			if onSendError != nil {
				go onSendError(p, err)
			}
			pktCh <- &pkt{err: p}
		}
	}

	rOnReply := func(p *packet.Packet) {
		if p.Sent.Add(timeout).Before(p.Recieved) {
			if onTimeout != nil {
				go onTimeout(p.ToSentPacket())
			}
		} else {
			if onReply != nil {
				go onReply(p)
			}
		}
		pktCh <- &pkt{recv: p}
	}

	return rOnReply, rOnSend, rOnSendError
}

type pkt struct {
	sent *packet.SentPacket
	recv *packet.Packet
	err  *packet.SentPacket
}

func processPkt(pending map[int]*packet.SentPacket, p *pkt, t *time.Timer, timeout time.Duration, onTimeout func(*packet.SentPacket)) {
	if p.sent != nil {
		pending[p.sent.Seq] = p.sent
		if len(pending) == 1 {
			resetTimer(t, p.sent.Sent, timeout)
		}
	}
	if p.recv != nil {
		delete(pending, p.recv.Seq)
	}
	if p.err != nil {
		delete(pending, p.err.Seq)
	}
	if len(pending) == 0 {
		stopTimer(t)
	}
}

func resetTimer(t *time.Timer, s time.Time, d time.Duration) {
	stopTimer(t)
	rd := s.Add(d).Sub(time.Now())
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

func processTimeout(pending map[int]*packet.SentPacket, t *time.Timer, timeout time.Duration, onTimeout func(*packet.SentPacket), n time.Time) {
	var resetS time.Time
	for s, p := range pending {
		if p.Sent.Add(timeout).Before(n) {
			if onTimeout != nil {
				go onTimeout(p)
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
