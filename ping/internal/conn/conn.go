package conn

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
)

type Conn struct {
	l       sync.RWMutex
	running bool
	cancel  func()
	conn    conn
	handler func(*ping.Ping, error)
	// I had a waitgroup for workers, but it has been removed
	// There's no reason to delay a stop waiting for these to shutdown
	// If a new listen comes in, a new listener will be created
	// Just cancel the context and let them die on their own.
}

type conn interface {
	start() error
	writeTo([]byte, *net.IPAddr) (int, error)
	read() (*ping.Ping, error)
	close() error
}

func New(proto int, h func(*ping.Ping, error)) *Conn {
	return &Conn{
		cancel:  func() {},
		conn:    newConn(proto),
		handler: h,
	}
}

func (c *Conn) Run(workers int) error {
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
	c.runWorkers(ctx, workers)
	c.running = true
	c.l.Unlock()
	return nil
}

func (c *Conn) Stop() error {
	c.l.Lock()
	c.cancel()
	// TODO is throwing packets still necessary?
	err := c.conn.close()
	if err != nil {
		c.l.Unlock()
		return err
	}
	c.running = false
	c.l.Unlock()
	return nil
}

var ErrNotRunning = errors.New("not running")

func (c *Conn) Send(p *ping.Ping) error {
	c.l.RLock()
	if !c.running {
		c.l.RUnlock()
		return ErrNotRunning
	}

	var err error
	var b []byte
send:
	for {
		p.Sent = time.Now()
		b, err = p.ToICMPMsg()
		if err != nil {
			return err
		}
		p.Len = len(b)
		_, err = c.conn.writeTo(b, p.Dst)
		if err != nil {
			subErr := err
			for {
				switch subErr.(type) {
				case syscall.Errno:
					if subErr == syscall.ENOBUFS {
						continue send
					}
					break send
				case *net.OpError:
					subErr = subErr.(*net.OpError).Err
				case *os.SyscallError:
					subErr = subErr.(*os.SyscallError).Err
				default:
					break send
				}
			}
		}
		break
	}

	c.l.RUnlock()
	return err
}
