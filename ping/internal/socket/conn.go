package socket

import (
	"math/rand"
	"net"

	"github.com/TrilliumIT/go-multiping/ping/internal/conn"
	"github.com/TrilliumIT/go-multiping/ping/internal/endpointmap"
	"github.com/TrilliumIT/go-multiping/ping/internal/ping"
	"github.com/TrilliumIT/go-multiping/ping/internal/timeoutmap"
)

func (s *Socket) Add(dst *net.IPAddr, handle func(*ping.Ping, error)) (int, error) {
	conn, em, tm := s.getStuff(dst.IP)
	return s.add(conn, em, tm, dst, handle)
}

func (s *Socket) add(
	conn *conn.Conn, em *endpointmap.Map, tm *timeoutmap.Map,
	dst *net.IPAddr, handle func(*ping.Ping, error),
) (int, error) {
	var id, sl int
	var err error
	s.l.Lock()
	defer s.l.Unlock()
	for {
		id = rand.Intn(1<<16-2) + 1
		_, sl, err = em.Add(dst.IP, id, handle)
		if err == endpointmap.ErrAlreadyExists {
			continue
		}
		if sl == 1 {
			err = conn.Run(s.Workers)
		}
		break
	}
	return id, err
}

func (s *Socket) Del(dst *net.IPAddr, id int) error {
	conn, em, tm := s.getStuff(dst.IP)
	return s.del(conn, em, tm, dst, id)
}

func (s *Socket) del(
	conn *conn.Conn, em *endpointmap.Map, tm *timeoutmap.Map,
	dst *net.IPAddr, id int) error {
	s.l.Lock()
	defer s.l.Unlock()
	_, sl, err := em.Pop(dst.IP, id)
	if err != nil {
		return err
	}
	if sl == 0 {
		err = conn.Stop()
	}
	return err
}

// SendPing sends the ping, in the process it sets the sent time
// This object will be held in the sequencemap until the reply is recieved
// or it times out, at which point it will be handled. The handled object
// will be the same as the sent ping but with the additional information from
// having been recieved.
func (s *Socket) SendPing(p *ping.Ping) error {
	conn, em, tm := s.getStuff(p.Dst.IP)
	sm, ok := em.Get(p.Dst.IP, p.ID)
	if !ok {
		return endpointmap.ErrDoesNotExist
	}

	_, err := sm.Add(p)
	if err != nil {
		return err
	}
	err = conn.Send(p)
	if err != nil {
		_, _, _ = sm.Pop(p.Seq)
		return err
	}
	tm.Add(p.Dst.IP, p.ID, p.Seq, p.TimeOutTime())
	return nil
}
