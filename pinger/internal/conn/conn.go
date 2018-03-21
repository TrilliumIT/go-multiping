package conn

import (
	"context"
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
	conn    conn
	handler func(*ping.Ping, error)
}

type conn interface {
	start() error
	writeTo([]byte, *net.IPAddr) (int, error)
	read() (*ping.Ping, error)
	close() error
}

func New(proto int) *Conn {
	return &Conn{
		cancel: func() {},
		conn:   newConn(proto),
	}
}

func (c *Conn) run(workers, buffer int) error {
	c.l.Lock()
	if c.running {
		c.l.Unlock()
		return nil
	}

	err := c.conn.start()
	if err != nil {
		_ = c.conn.close()
		c.l.Unlock()
		return err
	}
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(context.Background())
	c.runWorkers(ctx, workers, buffer)
	c.running = true
	c.l.Unlock()
	return nil
}

func (c *Conn) Stop() {
	c.l.Lock()
	c.cancel()
	// TODO is throwing packets still necessary?
	err := c.conn.close()
	if err != nil {
		// if this errs, these goroutines are stuck, nothing to be done, just die
		panic(err)
	}
	c.wg.Wait()
	c.running = false
	c.l.Unlock()
}

func (c *Conn) Send(p *ping.Ping, workers, buffer int) error {
	c.l.RLock()
	if !c.running {
		c.l.RUnlock()
		err := c.run(workers, buffer)
		if err != nil {
			return err
		}
		return c.Send(p, workers, buffer)
	}

	p.Sent = time.Now()
	b, err := p.ToICMPMsg()
	if err != nil {
		return err
	}
	p.Len, err = c.conn.writeTo(b, p.Dst)
	c.l.RUnlock()

	return err
}
