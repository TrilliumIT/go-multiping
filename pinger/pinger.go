package pinger

import (
	"net"
	"sync"

	protoPinger "github.com/TrilliumIT/go-multiping/internal/pinger"
)

// Pinger holds a single listener for incoming ICMPs. (one for ipv4 one for ipv6 if necessary). It only holds the listener open while a ping is running.
// One Pinger can support multiple ongoing pings with a single listener.
type Pinger struct {
	v4Pinger *protoPinger.Pinger
	v6Pinger *protoPinger.Pinger
}

var globalPinger *Pinger
var globalPingerLock sync.Mutex

func getGlobalPinger() *Pinger {
	globalPingerLock.Lock()
	defer globalPingerLock.Unlock()
	if globalPinger != nil {
		return globalPinger
	}
	if globalPinger == nil {
		globalPinger = NewPinger()
	}
	return globalPinger
}

// NewPinger returns a new Pinger
func NewPinger() *Pinger {
	return NewPingerWorkers(2, 16)
}

// NewPingerWorkers returns a new Pinger with the specified number of workers and channel length
// This type of tuning shouldn't be necessary for most workloads
func NewPingerWorkers(workers, chanLen int) *Pinger {
	return &Pinger{
		v4Pinger: protoPinger.New(4, workers, chanLen),
		v6Pinger: protoPinger.New(6, workers, chanLen),
	}
}

func (p *Pinger) getProtoPinger(ip net.IP) *protoPinger.Pinger {
	if ip != nil && ip.To4() != nil {
		return p.v4Pinger
	}
	return p.v6Pinger
}
