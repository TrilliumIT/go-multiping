package pinger

import (
	"sync"

	"github.com/TrilliumIT/go-multiping/internal/listenMap"
)

// Conn is a raw socket connection (one for ipv4 and one for ipv6). Connections are only actively listening when there are active pings going on
// Conn must be created via NewConn
type Conn struct {
	lm *listenMap.ListenMap
}

// NewConn returns a new Conn
func NewConn() *Conn {
	return &Conn{
		lm: listenMap.NewListenMap(),
	}
}

// SetWorkers sets a number of workers to process incoming packets
// By default each incoming packet fires off a new goroutine to process it and call the callback
// Setting workers changes this behavior so that n goroutines are already running and waiting to process packets.
// This may provide better performance than dynamic goroutines. But if the workers are filled this can lead to dropped ping responses.
// This change will not take effect until all running pings on Conn are stopped
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
