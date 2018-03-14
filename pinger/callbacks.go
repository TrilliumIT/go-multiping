package pinger

import (
	"time"

	"github.com/clinta/go-multiping/packet"
)

func (d *Dst) afterReply(p *packet.Packet) {
	d.pktCh <- &pkt{p, false}
	if !p.Sent.Add(d.timeout).Before(p.Recieved) {
		if d.onReply != nil {
			d.cbWg.Add(1)
			go func() {
				d.onReply(p)
				d.cbWg.Done()
			}()
		}
		return
	}
	d.afterTimeout(p)
}

func (d *Dst) beforeSend(p *packet.Packet) {
	d.pktCh <- &pkt{p, true}
}

func (d *Dst) afterSend(p *packet.Packet) {
	if d.onSend != nil {
		d.cbWg.Add(1)
		go func() {
			d.onSend(p)
			d.cbWg.Done()
		}()
	}
}

func (d *Dst) afterSendError(p *packet.Packet, err error) error {
	d.pktCh <- &pkt{p, false}
	if d.onSendError != nil {
		d.cbWg.Add(1)
		go func() {
			d.onSendError(p, err)
			d.cbWg.Done()
		}()
		return nil
	}
	return err
}

func (d *Dst) afterTimeout(p *packet.Packet) {
	if d.onTimeout != nil {
		d.cbWg.Add(1)
		go func() {
			d.onTimeout(p)
			d.cbWg.Done()
		}()
	}
}

func (d *Dst) runSend() {
	d.cbWg.Add(1)
	go func() {
		defer d.cbWg.Done()
		t := time.NewTimer(d.timeout)
		if !t.Stop() {
			<-t.C
		}
		pending := make(map[uint16]*packet.Packet)
		sendingTrigger := make(chan struct{}, 1)
		d.cbWg.Add(1)
		go func() {
			<-d.sending
			sendingTrigger <- struct{}{}
			d.cbWg.Done()
		}()
		for {
			select {
			case p := <-d.pktCh:
				d.processPkt(pending, p, t)
				continue
			case <-d.stop:
				return
			default:
			}

			select {
			case <-d.sending:
				if len(pending) == 0 {
					return
				}
			default:
			}

			select {
			case p := <-d.pktCh:
				d.processPkt(pending, p, t)
				continue
			case n := <-t.C:
				d.processTimeout(pending, t, n)
			case <-sendingTrigger:
				continue
			case <-d.stop:
				return
			}
		}
	}()

	return
}

type pkt struct {
	p *packet.Packet
	a bool
}

func (d *Dst) processPkt(pending map[uint16]*packet.Packet, p *pkt, t *time.Timer) {
	if p.a {
		pending[uint16(p.p.Seq)] = p.p
		if len(pending) == 1 {
			resetTimer(t, time.Now(), d.timeout)
		}
	} else {
		delete(pending, uint16(p.p.Seq))
	}
	if len(pending) == 0 {
		stopTimer(t)
	}
}

func (d *Dst) processTimeout(pending map[uint16]*packet.Packet, t *time.Timer, n time.Time) {
	var resetS time.Time
	for s, p := range pending {
		if p.Sent.Add(d.timeout).Before(n) {
			d.afterTimeout(p)
			delete(pending, s)
			continue
		}
		if resetS.IsZero() || resetS.After(p.Sent) {
			resetS = p.Sent
		}
	}

	if !resetS.IsZero() {
		resetTimer(t, resetS, d.timeout)
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
