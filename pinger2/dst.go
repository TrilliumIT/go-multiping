package pinger

import (
	"context"
)

type Dst struct {
	ctx    context.Context
	cancel func()
}

func (d *Dst) Cancel() {
	d.cancel()
}
