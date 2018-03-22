package conn

import (
	"context"
)

func (c *Conn) runWorkers(ctx context.Context, workers int) {
	if workers < -1 {
		go c.singleWorker(ctx)
		return
	}

	panic("not implemented")
}

func ctxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}

func (c *Conn) singleWorker(ctx context.Context) {
	for {
		p, err := c.conn.read()
		if ctxDone(ctx) {
			return
		}
		c.handler(p, err)
	}
}
