package multidns

import (
	"net"
	"sync"
)

type Resolver struct {
	Local *net.UDPAddr
	co    *net.UDPConn
	mu    sync.Mutex
	po    sync.Pool
}

func (r *Resolver) pop() []byte {
	p := r.po.Get()
	switch b := p.(type) {
	case []byte:
		return b
	default:
		return make([]byte, 512)
	}
}

func (r *Resolver) put(b []byte) {
	r.po.Put(b)
}

func (r *Resolver) send(m *Message) (err error) {

	return
}
