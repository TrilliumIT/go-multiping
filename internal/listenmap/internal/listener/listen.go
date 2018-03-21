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
	conn  *icmp.PacketConn
	Props *messages.Props
	addL  chan *runParams
	delL  chan struct{}
	// try to lock for sending
	// if returned f is not nil, call f to unlock
	// if f is nil then i is zero and listener is not running
	lockL chan chan<- func()
}

// New returns a new listener
func New(p int) *Listener {
	l := &Listener{
		proto: p,
		addL:  make(chan *runParams),
		delL:  make(chan struct{}),
		lockL: make(chan chan<- func()),
	}
	switch p {
	case 4:
		l.Props = messages.V4Props
	case 6:
		l.Props = messages.V6Props
	}
	go func() {
		var err error
		var i int
		pause := make(chan struct{})
		unpause := func() { pause <- struct{}{} }
		cancelAndWait := func() {}
		for {
			select {
			case r := <-l.lockL:
				if i == 0 {
					r <- nil
					continue
				}
				r <- unpause
				<-pause
			case r := <-l.addL:
				i += 1
				if i == 1 {
					cancelAndWait, err = l.run(r.getCb, r.workers, r.buffer)
				}
				r.err <- err
				err = nil
			case <-l.delL:
				i -= 1
				if i < 0 {
					panic("listener decremented below 0")
				}
				if i == 0 {
					// shut her down
					cancelAndWait()
				}
			}
		}
	}()
	return l
}

// ErrNotRunning is returned if send is requested and listener is not running
var ErrNotRunning = errors.New("listener not running")

// Send sends a packet using this connectiong
func (l *Listener) Send(p *ping.Ping, dst net.Addr) error {
	c := make(chan func())
	l.lockL <- c
	unlock := <-c
	if unlock == nil {
		return ErrNotRunning
	}
	err := l.send(p, dst)
	unlock()
	return err
}

// ToICMPMsg returns a byte array ready to send on the wire
func (l *Listener) sendAngryPacket() {
	b, err := (&icmp.Message{
		Code: 0,
		Type: l.Props.RecvType,
		Body: &icmp.Echo{
			ID:  0,
			Seq: 0,
		},
	}).Marshal(nil)
	if err != nil {
		panic(err)
	}
	_, err = l.conn.WriteTo(b, l.Props.SrcAddr)
	if err != nil {
		panic(err)
	}
}

func (l *Listener) send(p *ping.Ping, dst net.Addr) error {
	p.Sent = time.Now()
	b, err := p.ToICMPMsg()
	if err != nil {
		return err
	}
	p.Len, err = l.conn.WriteTo(b, dst)
	return err
}

// Run either starts the listner, or adds another waiter to prevent it from stopping
func (l *Listener) Run(getCb func(net.IP, uint16) func(context.Context, *ping.Ping), workers int, buffer int) (func(), error) {
	done := func() { l.delL <- struct{}{} }
	eCh := make(chan error)
	l.addL <- &runParams{getCb, workers, buffer, eCh}
	return done, <-eCh
}

type runParams struct {
	getCb   func(net.IP, uint16) func(context.Context, *ping.Ping)
	workers int
	buffer  int
	err     chan<- error
}

func (l *Listener) run(getCb func(net.IP, uint16) func(context.Context, *ping.Ping), workers int, buffer int) (cancelAndWait func(), err error) {
	cancelAndWait = func() {}
	l.conn, err = icmp.ListenPacket(l.Props.Network, l.Props.Src)
	if err != nil {
		return cancelAndWait, err
	}
	err = setPacketCon(l.conn)
	if err != nil {
		_ = l.conn.Close()
		return cancelAndWait, err
	}

	// this is not inheriting a context. Each ip has a context, which will decrement the waitgroup when it's done.
	wCtx, wCancel := context.WithCancel(context.Background())

	// start workers
	proc, wWait := getProcFunc(wCtx, workers, buffer)

	pWg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	rWg := make(chan struct{})
	go func() {
		for {
			r := &messages.RecvMsg{
				Payload: make([]byte, l.Props.ExpectedLen),
			}
			err := readPacket(l.conn, r)
			if err != nil {
				continue
			}
			select {
			case <-ctx.Done():
				close(rWg)
				return
			default:
			}
			pWg.Add(1)
			r.Recieved = time.Now()
			proc(ctx, r, getCb, pWg.Done)
		}
	}()

	cancelAndWait = func() {
		cancel() // stop conection listener
		// Despite https://golang.org/pkg/net/#PacketConn claims
		// close does not actually cause reads to be unblocked.
		// This leads to nasty deadlocks.
		// Throw angry packets at the connection until it dies!
	angryPackets:
		for {
			select {
			case <-rWg: // The listern has returned
				break angryPackets
			default:
				l.sendAngryPacket()
			}
		}
		_ = l.conn.Close()
		pWg.Wait() // wait for packets to be distributed
		wCancel()  // stop workers
		wWait()    // wait for workers to stop
	}
	return cancelAndWait, nil
}
