package listener

import (
	"context"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"

	"github.com/TrilliumIT/go-multiping/internal/listenMap/internal/messages"
	"github.com/TrilliumIT/go-multiping/ping"
)

type Listener struct {
	proto int
	l     sync.RWMutex
	dead  chan struct{}
	ipWg  sync.WaitGroup
	conn  *icmp.PacketConn
	Props *messages.Props
}

func New(p int) *Listener {
	l := &Listener{proto: p}
	switch p {
	case 4:
		l.Props = messages.V4Props
	case 6:
		l.Props = messages.V6Props
	}
	l.dead = make(chan struct{})
	close(l.dead)
	return l
}

func (l *Listener) Running() bool {
	l.l.RLock()
	r := l.usRunning()
	l.l.RUnlock()
	return r
}

type ErrNotRunning struct{}

func (e *ErrNotRunning) Error() string {
	return "listener not running"
}

func (l *Listener) Send(p *ping.Ping, dst net.Addr) error {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.usRunning() {
		return &ErrNotRunning{}
	}
	l.ipWg.Add(1)
	defer l.ipWg.Done()
	// TODO the rest of the send
	p.Sent = time.Now()
	b, err := p.ToICMPMsg()
	if err != nil {
		return err
	}
	p.Len, err = l.conn.WriteTo(b, dst)
	return nil
}

func (l *Listener) usRunning() bool {
	select {
	case <-l.dead:
		return false
	default:
	}
	return true
}

func (l *Listener) WgAdd(delta int) {
	l.l.RLock()
	l.ipWg.Add(delta)
	l.l.RUnlock()
}

func (l *Listener) WgDone() {
	l.ipWg.Done()
}

func (l *Listener) Run(getCb func(net.IP, uint16) func(context.Context, *ping.Ping), workers int, buffer int) error {
	l.l.Lock()
	defer l.l.Unlock()
	if l.usRunning() {
		return nil
	}

	l.dead = make(chan struct{})
	var err error
	l.conn, err = icmp.ListenPacket(l.Props.Network, l.Props.Src)
	if err != nil {
		return err
	}
	err = setPacketCon(l.conn)
	if err != nil {
		_ = l.conn.Close()
		return err
	}

	// this is not inheriting a context. Each ip has a context, which will decrement the waitgroup when it's done.
	ctx, cancel := context.WithCancel(context.Background())
	wWg := sync.WaitGroup{}
	go func() {
		for {
			l.ipWg.Wait()
			l.l.Lock()
			wgC := make(chan struct{})
			go func() { l.ipWg.Wait(); close(wgC) }()
			select {
			case <-wgC:
			default:
				l.l.Unlock()
				continue
			}
			_ = l.conn.Close()
			cancel()
			wWg.Wait()
			close(l.dead)
			l.l.Unlock()
			break
		}
	}()

	// start workers
	wCh := make(chan *procMsg, buffer)
	proc := processMessage
	if workers == 0 {
		proc = func(p *procMsg) { go processMessage(p) }
		workers = 1
	}
	wWg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case p := <-wCh:
					proc(p)
				}
			}
		}()
	}

	wWg.Add(1)
	go func() {
		defer wWg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			r := &messages.RecvMsg{
				Payload: make([]byte, l.Props.ExpectedLen),
			}
			err := readPacket(l.conn, r)
			if err != nil {
				continue
			}
			r.Recieved = time.Now()
			wCh <- &procMsg{ctx, r, getCb}
		}
	}()
	return nil
}

type procMsg struct {
	ctx   context.Context
	r     *messages.RecvMsg
	getCb func(net.IP, uint16) func(context.Context, *ping.Ping)
}

func processMessage(pm *procMsg) {
	p := pm.r.ToPing()
	if p == nil {
		return
	}

	cb := pm.getCb(p.Dst, uint16(p.ID))
	if cb == nil {
		return
	}

	cb(pm.ctx, p)
}
