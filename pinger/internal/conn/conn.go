package socket

import (
	"net"
	"sync"
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
)

type Conn struct {
	l       sync.RWMutex
	running bool
	wg      sync.WaitGroup
	cancel  func()
	conn    net.PacketConn
}

func New() *Conn {
	return &Conn{
		cancel: func() {},
	}
}

func (c *Conn) run(workers, buffer int) {
	c.l.Lock()
	if c.running {
		c.l.Unlock()
		return
	}

	// TODO run
	// TODO set cancel
	c.l.Unlock()
}

func (c *Conn) Stop() {
	c.cancel()
	c.wg.Wait()
}

func (c *Conn) Send(p *ping.Ping, workers, buffer int) error {
	c.l.RLock()
	if !c.running {
		c.l.RUnlock()
		c.run(workers, buffer)
		return c.Send(p)
	}

	p.Sent = time.Now()
	b, err := p.ToICMPMsg()
	if err != nil {
		return err
	}
	p.Len, err = c.conn.WriteTo(b, p.Dst)
	c.l.RUnlock()

	return err
}
