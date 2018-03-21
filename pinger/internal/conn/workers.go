package socket

import (
	"context"
)

func startWorkers(
	ctx context.Context,
	workers int,
	add, done func(),
) {
	if workers < -1 {
	}

	panic("not implemented")
}
