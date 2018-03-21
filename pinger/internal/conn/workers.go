package conn

import (
	"context"
)

func (c *Conn) runWorkers(
	ctx context.Context,
	workers, buffer int,
) {
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
		// TODO
		//r, err := c.readPacket()
		if ctxDone(ctx) {
			return
		}
		/*
			if err != nil {
				continue
			}
			processMessage(ctx, r, getCb)
		*/
	}
}
