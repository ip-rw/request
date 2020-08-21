package request

import (
	"errors"
	"sync"
	"sync/atomic"
)

// ErrServersNotExists is the error that servers dose not exists
var ErrServersNotExists = errors.New("servers dose not exist")

type RoundRobin struct {
	*sync.Mutex
	addrs []*WrapTCPDialer
	next  uint32
}

func NewRR() *RoundRobin {
	return &RoundRobin{
		addrs: []*WrapTCPDialer{},
	}
}
// Next returns next address
func (r *RoundRobin) Add(toadd *WrapTCPDialer) bool {
	//r.Lock()
	//defer r.Unlock()
	seen := false
	for _, addr := range r.addrs {
		if addr.LocalAddr.String() == toadd.LocalAddr.String() {
			seen = true
			break
		}
	}
	if !seen {
		r.addrs = append(r.addrs, toadd)
		return true
	}
	return false
}

func (r *RoundRobin) Next() *WrapTCPDialer	 {
	n := atomic.AddUint32(&r.next, 1)
	return r.addrs[(int(n)-1)%len(r.addrs)]
}
