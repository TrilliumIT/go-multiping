package pinger

import (
	"context"
	"math/rand"
	"net"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/listenmap"
	"github.com/TrilliumIT/go-multiping/ping"
	"github.com/TrilliumIT/go-multiping/pinger/internal/pending"
	"github.com/TrilliumIT/go-multiping/pinger/internal/ticker"
)

func (c *Conn) pingWithTicker(ctx context.Context, tick ticker.Ticker, pktWg *sync.WaitGroup, host string, hf HandleFunc, conf *PingConf) error {
	pm := pending.NewMap()
	phf := func(p *ping.Ping, err error) { hf(ctx, p, err) }
	var id, seq uint16
	var dst *net.IPAddr
	sent := -1 // number of packets attempted to be sent
	var lCancel func()
	var err error
	wCtx, wCancel := context.WithCancel(ctx)
	defer wCancel()
	proc, wWait := getProcFunc(wCtx, conf.Workers, conf.Buffer, pm, phf, pktWg)
	tick.Ready()
	for {
		sent++
		if conf.Count > 0 && sent >= conf.Count {
			break
		}

		select {
		case <-ctx.Done():
			run(lCancel)
			return nil
		case <-tick.C():
		}

		pktWg.Add(1)
		tick.Ready()

		if id == 0 {
			id = uint16(rand.Intn(1<<16-2) + 1)
		}
		seq = uint16(sent)

		p := &pending.Ping{P: &ping.Ping{
			Host:    host,
			ID:      int(id),
			Seq:     int(seq),
			TimeOut: conf.Timeout,
		}}
		p.Lock()
		var pCtx context.Context
		pCtx, p.Cancel = context.WithCancel(ctx)

		if dst == nil || (conf.ReResolveEvery != 0 && sent%conf.ReResolveEvery == 0) {
			var changed bool
			dst, changed, p.Err = resolve(dst, host)
			if p.Err != nil {
				run(p.Unlock, p.Cancel)
				if conf.RetryOnResolveError {
					proc(&procPing{p, pCtx})
					pktWg.Done()
					continue
				}
				run(pktWg.Done, lCancel)
				return p.Err
			}

			if changed {
				run(lCancel) // cancel the current listener
				lCancel = nil
			}
		}
		p.P.Dst = dst.IP
		p.P.Src = c.lm.SrcAddr(dst.IP)

		if lCancel == nil { // we don't have a listner yet for this dst
			// Register with listenmap
			lCancel, err = addListener(ctx, c.lm, dst.IP, id, pm.OnRecv)
			if err != nil {
				// we need to call done here, because we're not calling wait on this error. Add errors that arent ErrAlreadyExists are a returnable problem
				run(p.Unlock, p.Cancel, pktWg.Done, lCancel)
				lCancel = nil
				if err == listenmap.ErrAlreadyExists { // we already have this listener registered
					id = 0 // try a different id
					continue
				}
				return err
			}
		}

		if opp, ok := pm.Add(p); ok {
			// we've looped seq and this old pending packet is still hanging around, cancel it
			opp.SetError(ErrSeqWrapped)
			opp.Cancel()
		}

		p.Err = c.lm.Send(p.P, dst)

		if p.Err != nil {
			run(p.Unlock, p.Cancel)
			if conf.RetryOnSendError {
				proc(&procPing{p, pCtx})
				pktWg.Done()
				continue
			}
			run(pktWg.Done, lCancel)
			return p.Err
		}

		if conf.Timeout > 0 {
			// we're not running wait yet, so nothing is waiting on this ctx, we're replacing it with one with a timeout now
			// but canceling is a good idea to release resources from the previous ctx
			run(p.Cancel)
			pCtx, p.Cancel = context.WithTimeout(ctx, conf.Timeout)
		}

		run(p.Unlock)
		proc(&procPing{p, pCtx})
		pktWg.Done()
	}

	wWait()
	wCancel()
	pktWg.Wait()
	run(lCancel)
	return nil
}

func run(f ...func()) {
	for _, ff := range f {
		if ff != nil {
			ff()
		}
	}
}
