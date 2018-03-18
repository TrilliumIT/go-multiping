package pinger

import (
	"context"

	"github.com/TrilliumIT/go-multiping/internal/listenMap"
)

func init() {
	connCreated = make(chan struct{})
}

// Conn is a raw socket connection (one for ipv4 and one for ipv6). Connections are only actively listening when there are active pings going on
// Conn must be created via NewConn or NewConnWithContext
type Conn struct {
	ctx context.Context
	lm  *listenMap.ListenMap
}

var conn *Conn
var connCreated chan struct{}

// DefaultConn is the default global conn used by the pinger package
func DefaultConn() *Conn {
	select {
	case <-connCreated:
		return conn
	default:
	}
	conn = NewConn()
	close(connCreated)
	return conn
}

// NewConn returns a new Conn
func NewConn() *Conn {
	return NewConnWithContext(context.Background())
}

// NewConnWithContext returns a new Conn with the given context. When ctx ic canceled, all active pings and listeners are stopped.
func NewConnWithContext(ctx context.Context) *Conn {
	return &Conn{
		ctx: ctx,
		lm:  listenMap.NewListenMap(ctx),
	}
}
