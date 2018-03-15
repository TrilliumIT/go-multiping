package pinger

import (
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
)

func (d *Dst) callbackWorker() {
	if d.callBack == nil {
		for range d.cbCh {
		}
	}
	for pkt := range d.cbCh {
		d.cbWg.Add(1)
		d.callBack(pkt.p, pkt.err)
		d.cbWg.Done()
	}
}

type pktWErr struct {
	p   *ping.Ping
	err error
}

func (d *Dst) runCallback(p *ping.Ping, err error) {
	d.cbWg.Add(1)
	pkt := &pktWErr{p, err}
	select {
	case d.cbCh <- pkt:
	default:
		d.cbWg.Add(1)
		go func() {
			if d.callBack != nil {
				d.callBack(p, err)
			}
			d.cbWg.Done()
		}()
	}
	d.cbWg.Done()
}

func (d *Dst) afterReply(rp *ping.Ping) {
	rCh := make(chan *ping.Ping)
	defer close(rCh)
	d.pktCh <- &pkt{rp, false, rCh}
	p := <-rCh
	if p == nil {
		// dupliate packet, was not in pending map
		return
	}
	p.Sent = rp.Sent // more accurate timestamp from body of recieved packet
	p.TTL = rp.TTL
	p.Len = rp.Len
	if rp.Src != nil { // src is the actual source on recieved packet, rather than wildcard
		p.Src = rp.Src
	}
	p.Recieved = rp.Recieved
	if !p.Sent.Add(d.timeout).Before(p.Recieved) {
		d.runCallback(p, nil)
		return
	}
	d.afterTimeout(p)
}

func (d *Dst) beforeSend(p *ping.Ping) {
	d.pktCh <- &pkt{p, true, nil}
}

func (d *Dst) afterSend(p *ping.Ping) {
	if d.callBackOnSend {
		d.runCallback(p, nil)
	}
}

func (d *Dst) afterSendError(p *ping.Ping, err error) error {
	d.pktCh <- &pkt{p, false, nil}
	if d.reSend {
		d.runCallback(p, err)
		return nil
	}
	return err
}

func (d *Dst) afterTimeout(p *ping.Ping) {
	d.runCallback(p, nil)
}

func (d *Dst) runSend() {
	d.cbWg.Add(1)
	go func() {
		defer d.cbWg.Done()
		t := time.NewTimer(d.timeout)
		if !t.Stop() {
			<-t.C
		}
		pending := make(map[uint16]*ping.Ping)
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
}

type pkt struct {
	p   *ping.Ping
	a   bool
	rCh chan<- *ping.Ping
}

func (d *Dst) processPkt(pending map[uint16]*ping.Ping, p *pkt, t *time.Timer) {
	if p.a {
		pending[uint16(p.p.Seq)] = p.p
		if len(pending) == 1 {
			resetTimer(t, time.Now(), d.timeout)
		}
	} else {
		if p.rCh != nil {
			p.rCh <- pending[uint16(p.p.Seq)]
		}
		delete(pending, uint16(p.p.Seq))
	}
	if len(pending) == 0 {
		stopTimer(t)
	}
}

func (d *Dst) processTimeout(pending map[uint16]*ping.Ping, t *time.Timer, n time.Time) {
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
