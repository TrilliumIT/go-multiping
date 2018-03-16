package pinger

import (
	"context"
	"encoding/binary"
	"net"

	"github.com/TrilliumIT/go-multiping/ping"
)

type index struct {
	ip net.IP
	id int
}

func (i *index) key() [18]byte {
	var r [18]byte
	copy(r[0:16], i.ip.To16())
	binary.LittleEndian.PutUint16(r[16:], uint16(i.id))
	return r
}

type setter struct {
	*index
	cb func(*ping.Ping)
}

type getter struct {
	*index
	cb chan<- func(*ping.Ping)
}

type manager struct {
	ctx   context.Context
	setCh chan *setter
	getCh chan *getter
	delCh chan *index
}

func (m *manager) set(ip net.IP, id int, cb func(*ping.Ping)) {
	set(m.ctx, m.setCh, ip, id, cb)
}

func set(ctx context.Context, set chan<- *setter, ip net.IP, id int, cb func(*ping.Ping)) {
	select {
	case set <- &setter{&index{ip, id}, cb}:
	case <-ctx.Done():
	}
}

func (m *manager) get(ip net.IP, id int) func(*ping.Ping) {
	return get(m.ctx, m.getCh, ip, id)
}

func get(ctx context.Context, get chan<- *getter, ip net.IP, id int) func(*ping.Ping) {
	c := make(chan func(*ping.Ping))

	select {
	case get <- &getter{&index{ip, id}, c}:
	case <-ctx.Done():
		return nil
	}

	select {
	case cb := <-c:
		return cb
	case <-ctx.Done():
	}
	return nil
}

func (m *manager) del(ip net.IP, id int) {
	del(m.ctx, m.delCh, ip, id)
}

func del(ctx context.Context, del chan<- *index, ip net.IP, id int) {
	select {
	case del <- &index{ip, id}:
	case <-ctx.Done():
	}
}

func (m *manager) run() {
	run(m.ctx, m.setCh, m.getCh, m.delCh)
}

func run(ctx context.Context, set <-chan *setter, get <-chan *getter, del <-chan *index) {
	callbacks := make(map[[18]byte]func(*ping.Ping))
	for {
		select {
		case s := <-set:
			callbacks[s.key()] = s.cb
		case d := <-del:
			delete(callbacks, d.key())
		case g := <-get:
			select {
			case g.cb <- callbacks[g.key()]:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
