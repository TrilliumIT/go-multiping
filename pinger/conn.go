package pinger

import (
	"context"
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/listenMap"
)

// Conn is a raw socket connection (one for ipv4 and one for ipv6). Connections are only actively listening when there are active pings going on
// Conn must be created via NewConn or NewConnWithContext
type Conn struct {
	lm *listenMap.ListenMap
}

var conn *Conn
var connLock sync.RWMutex

// DefaultConn is the default global conn used by the pinger package
func DefaultConn() *Conn {
	connLock.RLock()
	if conn != nil {
		connLock.RUnlock()
		return conn
	}

	connLock.RUnlock()
	connLock.Lock()
	if conn != nil {
		connLock.Unlock()
		return conn
	}
	conn = NewConn()
	connLock.Unlock()
	return conn
}

// NewConn returns a new Conn
func NewConn() *Conn {
	return NewConnWithContext(context.Background())
}

// NewConnWithContext returns a new Conn with the given context. When ctx ic canceled, all active pings and listeners are stopped.
func NewConnWithContext(ctx context.Context) *Conn {
	return &Conn{
		lm: listenMap.NewListenMap(ctx),
	}
}
