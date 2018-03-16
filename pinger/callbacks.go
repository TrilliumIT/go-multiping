package pinger

import (
	"fmt"
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
)

func (d *Dst) afterReply(p *ping.Ping) {
	d.cbWg.Add(1)
	defer d.cbWg.Done()
	if d.timeout > 0 {
		p.TimeOut = p.Sent.Add(d.timeout)
	}
	d.pktCh <- &pkt{p, false}
	if !p.Sent.Add(d.timeout).Before(p.Recieved) {
		if d.callBack != nil {
			d.cbWg.Add(1)
			go func() {
				d.callBack(p, nil)
				d.cbWg.Done()
			}()
		}
		return
	}
	d.afterTimeout(p)
}

func (d *Dst) beforeSend(p *ping.Ping) {
	d.pktCh <- &pkt{p, true}
}

func (d *Dst) afterSend(p *ping.Ping) {
	if d.callBackOnSend {
		d.cbWg.Add(1)
		go func() {
			d.callBack(p, nil)
			d.cbWg.Done()
		}()
	}
}

func (d *Dst) afterSendError(p *ping.Ping, err error) error {
	d.pktCh <- &pkt{p, false}
	if d.reSend {
		d.cbWg.Add(1)
		go func() {
			d.callBack(p, err)
			d.cbWg.Done()
		}()
		return nil
	}
	return err
}

func (d *Dst) afterTimeout(p *ping.Ping) {
	if d.callBack != nil {
		d.cbWg.Add(1)
		go func() {
			d.callBack(p, nil)
			d.cbWg.Done()
		}()
	}
}

func (d *Dst) runSend() {
	d.cbWg.Add(1)
	go func() {
		defer d.cbWg.Done()
		t := time.NewTimer(time.Hour)
		stopTimer(t)
		nextTimeout := time.Time{}
		pending := make(map[uint16]*ping.Ping)
		doneSending := false
		for {
			select {
			case p := <-d.pktCh:
				nextTimeout = d.processPkt(pending, p, t, nextTimeout)
				continue
			case <-d.stop:
				return
			default:
			}

			if doneSending {
				if len(pending) == 0 {
					return
				}

				fmt.Printf("len pending: %v\n", len(pending))
				fmt.Printf("next timeout: %v\n", nextTimeout)

				select {
				case p := <-d.pktCh:
					nextTimeout = d.processPkt(pending, p, t, nextTimeout)
				case n := <-t.C:
					nextTimeout = d.processTimeout(pending, t, n)
				case <-d.stop:
					return
				}
				continue
			}

			select {
			case p := <-d.pktCh:
				nextTimeout = d.processPkt(pending, p, t, nextTimeout)
			case n := <-t.C:
				nextTimeout = d.processTimeout(pending, t, n)
			case <-d.sending:
				doneSending = true
			case <-d.stop:
				return
			}
		}
	}()
}

type pkt struct {
	p *ping.Ping
	a bool
}

func (d *Dst) processPkt(pending map[uint16]*ping.Ping, p *pkt, t *time.Timer, nt time.Time) time.Time {
	if p.a {
		pending[uint16(p.p.Seq)] = p.p
	} else {
		delete(pending, uint16(p.p.Seq))
	}
	if len(pending) == 0 {
		stopTimer(t)
		nt = time.Time{}
		return nt
	}
	if nt.IsZero() || nt.After(time.Now().Add(d.timeout)) {
		nt = time.Now().Add(d.timeout)
		resetTimer(t, nt)
	}
	return nt
}

func (d *Dst) processTimeout(pending map[uint16]*ping.Ping, t *time.Timer, n time.Time) time.Time {
	var nextTimeout time.Time
	for s, p := range pending {
		if p.IsSending() {
			continue
		}
		if p.TimeOut.Before(n) {
			d.afterTimeout(p)
			delete(pending, s)
			continue
		}
		if nextTimeout.IsZero() || nextTimeout.After(p.TimeOut) {
			nextTimeout = p.TimeOut
		}
	}

	if !nextTimeout.IsZero() {
		resetTimer(t, nextTimeout)
	}
	return nextTimeout
}

func resetTimer(t *time.Timer, e time.Time) {
	stopTimer(t)
	d := time.Until(e)
	if d < time.Nanosecond {
		d = time.Nanosecond
	}
	t.Reset(d)
}

func stopTimer(t *time.Timer) {
	t.Stop()
	select {
	case <-t.C:
	default:
	}
}
