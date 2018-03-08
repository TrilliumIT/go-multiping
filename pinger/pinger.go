package pinger

import (
	protoPinger "github.com/clinta/go-multiping/internal/pinger"
	"net"
)

type Pinger struct {
	v4Pinger *protoPinger.Pinger
	v6Pinger *protoPinger.Pinger
}

func NewPinger() *Pinger {
	return &Pinger{
		v4Pinger: protoPinger.New(4),
		v6Pinger: protoPinger.New(6),
	}
}

func (p *Pinger) getProtoPinger(ip net.IP) *protoPinger.Pinger {
	if ip != nil && ip.To4() != nil {
		return p.v4Pinger
	}
	return p.v6Pinger
}
