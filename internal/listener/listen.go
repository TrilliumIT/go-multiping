package listener

import (
	"context"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"

	"github.com/TrilliumIT/go-multiping/internal/messages"
	"github.com/TrilliumIT/go-multiping/internal/process"
	"github.com/TrilliumIT/go-multiping/ping"
)

type Listener struct {
	proto int
	l     sync.RWMutex
	dead  chan struct{}
	wg    sync.WaitGroup
	ctx   context.Context
	conn  *icmp.PacketConn
	props *messages.Props
}

func New(p int) *Listener {
	l := &Listener{proto: p}
	switch p {
	case 4:
		l.props = messages.V4Props
	case 6:
		l.props = messages.V6Props
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

func (l *Listener) usRunning() bool {
	select {
	case <-l.dead:
		return false
	default:
	}
	return true
}

func (l *Listener) WgAdd(delta int) {
	l.wg.Add(delta)
}

func (l *Listener) WgDone() {
	l.wg.Done()
}

func (l *Listener) Run(getCb func(net.IP, int) func(context.Context, *ping.Ping)) error {
	l.l.Lock()
	defer l.l.Unlock()
	if l.usRunning() {
		return nil
	}

	l.dead = make(chan struct{})
	var err error
	l.conn, err = icmp.ListenPacket(l.props.Network, l.props.Src)
	if err != nil {
		return err
	}
	err = setPacketCon(l.conn)
	if err != nil {
		_ = l.conn.Close()
		return err
	}

	var cancel func()
	// this is not inheriting a context. Each ip has a context, which will decrement the waitgroup when it's done.
	l.ctx, cancel = context.WithCancel(context.Background())
	go func() {
		l.wg.Wait()
		l.l.Lock()
		_ = l.conn.Close()
		cancel()
		<-l.dead
		l.l.Unlock()
	}()

	go func() {
		defer close(l.dead)
		for {
			select {
			case <-l.ctx.Done():
				return
			default:
			}
			r := &messages.RecvMsg{
				Payload: make([]byte, l.props.ExpectedLen),
			}
			err := readPacket(l.conn, r)
			if err != nil {
				continue
			}
			r.Recieved = time.Now()
			go process.ProcessMessage(l.ctx, r, getCb)
		}
	}()
	return nil
}
