package pinger

import (
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/listenmap"
)

// Conn is a raw socket connection (one for ipv4 and one for ipv6). Connections are only actively listening when there are active pings going on
// Conn must be created via NewConn
type Conn struct {
	lm *listenmap.ListenMap
}

// NewConn returns a new Conn
func NewConn() *Conn {
	c := &Conn{
		lm: listenmap.NewListenMap(),
	}
	c.lm.SetWorkers(-1)
	return c
}

// SetWorkers sets a number of workers to process incoming packets and distribute them to the appropriate handlers
// <-1 disables workers entirely and synchronously distributes the incoming packet before listening for a new one. This is the default.
// -1 enables automatic worker allocation. Any time packet processing would block, a new worker is started. Workers remain active until all active pings stop.
// 0 disalbes workers, causing each incoming packet to start a new goroutine to handle it. This can cause incoming packets to be missed.
// A postive number pre-allocates a set number of workers
func (c *Conn) SetWorkers(n int) {
	c.lm.SetWorkers(n)
}

// SetBuffer sets the buffer size for incoming packets being sent to workers
// At zero there is no buffer, and if the workers are not ready to process
// no new packets will be recieved until it is.
// This change will not take effect until all running pings on Conn are stopped
func (c *Conn) SetBuffer(n int) {
	c.lm.SetBuffer(n)
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
