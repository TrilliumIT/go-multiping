package listener

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"

	"github.com/TrilliumIT/go-multiping/internal/listenmap/internal/messages"
	"github.com/TrilliumIT/go-multiping/ping"
)

// Listener is a listener for one protocol, ipv4 or ipv6
type Listener struct {
	proto int
	l     sync.RWMutex
	dead  chan struct{}
	ipWg  sync.WaitGroup
	conn  *icmp.PacketConn
	Props *messages.Props
}

// New returns a new listener
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

// Running is if the listener is currently listening for packets
func (l *Listener) Running() bool {
	l.l.RLock()
	r := l.usRunning()
	l.l.RUnlock()
	return r
}

// ErrNotRunning is returned if send is requested and listener is not running
var ErrNotRunning = errors.New("listener not running")

// Send sends a packet using this connectiong
func (l *Listener) Send(p *ping.Ping, dst net.Addr) error {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.usRunning() {
		return ErrNotRunning
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
	return err
}

func (l *Listener) usRunning() bool {
	select {
	case <-l.dead:
		return false
	default:
	}
	return true
}

// WgAdd adds a waitgroup entry to prevent this listener from shutting down
func (l *Listener) WgAdd(delta int) {
	l.l.RLock()
	l.ipWg.Add(delta)
	l.l.RUnlock()
}

// WgDone decremetns the wait group
func (l *Listener) WgDone() {
	l.ipWg.Done()
}

// Run starts this listener running
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
	proc := getProcFunc(ctx, workers, buffer, &wWg)

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
			proc(&procMsg{ctx, r, getCb})
		}
	}()
	return nil
}
