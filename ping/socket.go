package ping

import (
	"sync"

	"github.com/TrilliumIT/go-multiping/ping/internal/socket"
)

// Socket is a raw socket connection (one for ipv4 and one for ipv6). Sockets are only actively listening when there are one or more open connections
// Sockets must be created via NewSocket
type Socket struct {
	s *socket.Socket
}

// NewSocket returns a new Socket
func NewSocket() *Socket {
	s := &Socket{
		s: socket.New(),
	}
	return s
}

// SetWorkers sets a number of workers to process incoming packets and run handlers
// This change will only take effect once all open connections are closed
func (s *Socket) SetWorkers(n int) {
	s.s.Workers = n
}

var dSocket *Socket
var dSocketLock sync.RWMutex

// DefaultConn is the default global conn used by the pinger package
func DefaultSocket() *Socket {
	dSocketLock.RLock()
	if dSocket != nil {
		dSocketLock.RUnlock()
		return dSocket
	}

	dSocketLock.RUnlock()
	dSocketLock.Lock()
	if dSocket != nil {
		dSocketLock.Unlock()
		return dSocket
	}
	dSocket = NewSocket()
	dSocketLock.Unlock()
	return dSocket
}
