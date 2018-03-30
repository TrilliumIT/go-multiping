package socket

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"time"

	"github.com/TrilliumIT/go-multiping/ping/internal/conn"
	"github.com/TrilliumIT/go-multiping/ping/internal/endpointmap"
	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
	"github.com/TrilliumIT/go-multiping/ping/internal/timeoutmap"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (s *Socket) Add(dst *net.IPAddr, h func(*ping.Ping, error)) (ping.Id, error) {
	conn, em, tm, _, setCancel := s.getConnMaps(dst.IP)
	return s.add(conn, em, tm, setCancel, dst, h)
}

var ErrNoIDs = errors.New("no avaliable icmp IDs")
var ErrTimedOut = errors.New("timed out")

func (s *Socket) add(
	conn *conn.Conn, em *endpointmap.Map, tm *timeoutmap.Map, setCancel func(func()),
	dst *net.IPAddr, h func(*ping.Ping, error),
) (ping.Id, error) {
	var id int
	var sl int
	var err error
	s.l.Lock()
	defer s.l.Unlock()
	startId := rand.Intn(1<<16 - 1)
	for id = startId; id < startId+1<<16-1; id++ {
		_, sl, err = em.Add(dst.IP, ping.Id(id), h)
		if err == endpointmap.ErrAlreadyExists {
			continue
		}
		if sl == 1 {
			err = conn.Run(s.Workers)
			if err != nil {
				ctx, cancel := context.WithCancel(context.Background())
				setCancel(cancel)
				go func() {
					for ip, id, seq, _ := tm.Next(ctx); ip != nil; ip, id, seq, _ = tm.Next(ctx) {
						handle(em, tm, &ping.Ping{Dst: &net.IPAddr{IP: ip}, ID: id, Seq: seq}, ErrTimedOut)
					}
				}()
			}
		}
		return ping.Id(id), err
	}
	return 0, ErrNoIDs
}

func (s *Socket) Del(dst net.IP, id ping.Id) error {
	conn, em, tm, cancel, _ := s.getConnMaps(dst)
	return s.del(conn, em, tm, cancel, dst, id)
}

func (s *Socket) del(
	conn *conn.Conn, em *endpointmap.Map, tm *timeoutmap.Map, cancel func(),
	dst net.IP, id ping.Id) error {
	s.l.Lock()
	defer s.l.Unlock()
	sm, sl, err := em.Pop(dst, id)
	if err != nil {
		return err
	}
	sm.Close()
	if sl == 0 {
		err = conn.Stop()
		cancel()
	}
	return err
}

func (s *Socket) Drain(dst net.IP, id ping.Id) {
	conn, em, tm, cancel, _ := s.getConnMaps(dst)
	s.drain(conn, em, tm, cancel, dst, id)
}

func (s *Socket) drain(
	conn *conn.Conn, em *endpointmap.Map, tm *timeoutmap.Map, cancel func(),
	dst net.IP, id ping.Id) {
	s.l.Lock()
	sm, _, _ := em.Get(dst, id)
	if sm == nil {
		s.l.Unlock()
		return
	}
	s.l.Unlock()
	sm.Drain()
	return
}

// SendPing sends the ping, in the process it sets the sent time
// This object will be held in the sequencemap until the reply is recieved
// or it times out, at which point it will be handled. The handled object
// will be the same as the sent ping but with the additional information from
// having been recieved.
func (s *Socket) SendPing(p *ping.Ping) {
	conn, em, tm, _, _ := s.getConnMaps(p.Dst.IP)
	sm, ok, _ := em.Get(p.Dst.IP, p.ID)
	if !ok {
		return
	}

	var sl int
	sl = sm.Add(p)
	if sl == 0 {
		// Sending was closed
		return
	}
	dst, id, seq, to := p.Dst.IP, p.ID, p.Seq, p.TimeOut
	if to > 0 {
		tm.Add(dst, id, seq, time.Now().Add(2*to))
	}
	tot, err := conn.Send(p)
	if err != nil {
		tm.Del(dst, id, seq)
		if rp, _, err2 := sm.Pop(seq); err2 == nil {
			sm.Handle(rp, err)
		}
		return
	}
	if to > 0 {
		// update timeout with accurate timeout time
		tm.Update(dst, id, seq, tot)
	}
	return
}