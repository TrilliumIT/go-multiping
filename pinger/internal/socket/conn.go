package socket

import (
	"net"
	"sync"
	"time"

	"github.com/TrilliumIT/go-multiping/ping"
)

type conn struct {
	l      sync.RWMutex
	r      bool
	wg     sync.WaitGroup
	cancel func()
	conn   net.PacketConn
}

func (c *conn) run() {
	c.l.Lock()
	if c.r {
		c.l.Unlock()
		return
	}

	// TODO run
	// TODO set cancel
	c.l.Unlock()
}

func (c *conn) send(p *ping.Ping) error {
	c.l.RLock()
	if !c.r {
		c.l.RUnlock()
		c.run()
		return c.send(p)
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
