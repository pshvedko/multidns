package multidns

import (
	"net"
	"time"
)

const DefaultTimeout = 2 * time.Second

func LookupIPv4(host string, via net.IP) (ips []net.IP, err error) {
	var c net.Conn
	c, err = net.DialUDP("udp", nil, &net.UDPAddr{IP: via, Port: 53})
	if err != nil {
		return
	}
	defer c.Close()
	err = c.SetDeadline(time.Now().Add(DefaultTimeout))
	if err != nil {
		return
	}
	a := Message{}
	m := Message{}
	m.SetID(0)
	m.SetType(MessageQuery)
	m.SetRecursionDesired(true)
	m.AddQuery(NewQuery(NewDomain(host), TypeA, ClassIN))
	_, err = m.SendUDP(c)
	if err != nil {
		return
	}
	err = a.ReadUDP(c)
	if err != nil {
		return
	}
	ips = a.A()
	return
}

func LookupIPv6(host string, via net.IP) (ips []net.IP, err error) {
	var c net.Conn
	c, err = net.DialUDP("udp", nil, &net.UDPAddr{IP: via, Port: 53})
	if err != nil {
		return
	}
	defer c.Close()
	err = c.SetDeadline(time.Now().Add(DefaultTimeout))
	if err != nil {
		return
	}
	a := Message{}
	m := Message{}
	m.SetID(0)
	m.SetType(MessageQuery)
	m.SetRecursionDesired(true)
	m.AddQuery(NewQuery(NewDomain(host), TypeAAAA, ClassIN))
	_, err = m.SendUDP(c)
	if err != nil {
		return
	}
	err = a.ReadUDP(c)
	if err != nil {
		return
	}
	ips = a.A()
	return
}

func LookupIP(host string, via net.IP) (ips []net.IP, err error) {
	var c net.Conn
	c, err = net.DialUDP("udp", nil, &net.UDPAddr{IP: via, Port: 53})
	if err != nil {
		return
	}
	defer c.Close()
	err = c.SetDeadline(time.Now().Add(DefaultTimeout))
	if err != nil {
		return
	}
	a := Message{}
	m := Message{}
	m.SetID(0)
	m.SetType(MessageQuery)
	m.SetRecursionDesired(true)
	m.AddQuery(NewQuery(NewDomain(host), TypeA, ClassIN))
	_, err = m.SendUDP(c)
	if err != nil {
		return
	}
	err = a.ReadUDP(c)
	if err != nil {
		return
	}
	ips = a.A()
	//if cname := a.CNAME(); len(cname) > 0 {
	//	m.Question[0] = NewQuery(NewDomain(cname[0]), TypeAAAA, ClassIN)
	//} else {
	m.Question[0] = NewQuery(NewDomain(host), TypeAAAA, ClassIN)
	//}
	m.ID++
	_, err = m.SendUDP(c)
	if err != nil {
		return
	}
	err = a.ReadUDP(c)
	if err != nil {
		return
	}
	ips = append(ips, a.A()...)
	return
}
