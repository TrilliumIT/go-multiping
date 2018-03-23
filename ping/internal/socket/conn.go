package socket

import (
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

func (s *Socket) Add(dst *net.IPAddr, handle func(*ping.Ping, error)) (int, error) {
	conn, em, tm := s.getStuff(dst.IP)
	return s.add(conn, em, tm, dst, handle)
}

var ErrNoIDs = errors.New("no avaliable icmp IDs")

func (s *Socket) add(
	conn *conn.Conn, em *endpointmap.Map, tm *timeoutmap.Map,
	dst *net.IPAddr, handle func(*ping.Ping, error),
) (int, error) {
	var id int
	var sl int
	var err error
	s.l.Lock()
	defer s.l.Unlock()
	startId := rand.Intn(1<<16 - 1)
	for id = startId; id < startId+1<<16-1; id++ {
		_, sl, err = em.Add(dst.IP, uint16(id), handle)
		if err == endpointmap.ErrAlreadyExists {
			continue
		}
		if sl == 1 {
			err = conn.Run(s.Workers)
		}
		return int(uint16(id)), err
	}
	return 0, ErrNoIDs
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
	_, sl, err := em.Pop(dst.IP, uint16(id))
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
func (s *Socket) SendPing(p *ping.Ping) (int, error) {
	conn, em, tm := s.getStuff(p.Dst.IP)
	sm, ok := em.Get(p.Dst.IP, uint16(p.ID))
	if !ok {
		return 0, endpointmap.ErrDoesNotExist
	}

	_, count := sm.Add(p)
	err := conn.Send(p)
	if err != nil {
		_, _, _ = sm.Pop(p.Seq)
		return count, err
	}
	tm.Add(p.Dst.IP, p.ID, p.Seq, p.TimeOutTime())
	return count, nil
}
