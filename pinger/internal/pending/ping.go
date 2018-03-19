package pending

import (
	"context"
	"errors"
	"sync"

	"github.com/TrilliumIT/go-multiping/ping"
)

// Ping is a pending ping waiting on a recieved packet
type Ping struct {
	Cancel func()
	P      *ping.Ping
	l      sync.Mutex
	Err    error
}

// Lock locks a ping from modification
func (p *Ping) Lock() {
	p.l.Lock()
}

// Unlock unlocks it
func (p *Ping) Unlock() {
	p.l.Unlock()
}

// UpdateFrom updates a ping with the recieved ping
func (p *Ping) UpdateFrom(p2 *ping.Ping) {
	p.Lock()
	p.P.UpdateFrom(p2)
	p.Unlock()
}

// SetError sets the error on the pending ping to be sent to the handler
func (p *Ping) SetError(err error) {
	p.Lock()
	p.Err = err
	p.Unlock()
}

// ErrTimedOut is the error returned when a packet times out. Then calls the handler
var ErrTimedOut = errors.New("ping timed out")

// Wait waits for the reply to be recieved, or ctx to time out.
func (p *Ping) Wait(ctx context.Context, pm *Map, cb func(*ping.Ping, error)) {
	<-ctx.Done()
	p.l.Lock()
	pm.Del(uint16(p.P.Seq))
	if ctx.Err() == context.DeadlineExceeded && p.Err == nil {
		p.Err = ErrTimedOut
	}

	cb(p.P, p.Err)
	p.l.Unlock()
}
