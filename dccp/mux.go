// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
	"sync"
)

// XXX: Drop packets going to recently closed flows. Combine with
// XXX: Keepalive mechanism. Close connections that are idle for a while

// mux{} helps multiplex the connection-less physical packets among 
// multiple logical connections based on their logical flow ID.
type mux struct {
	sync.Mutex
	link       Link
	flows      []*flow	// TODO: Lookups in a short array should be fine for now. Hashing?
	acceptChan chan *flow
}

// muxHeader{} is an internal data structure that carries a parsed switch packet,
// which contains a flow id and a generic DCCP header
type muxHeader struct {
	Pair   *FlowPair
	Cargo  []byte
}

func newConnSwitch(link Link) *mux {
	m := &mux{ 
		link:       link, 
		flows:      make([]*flow, 0),
		acceptChan: make(chan *flow),
	}
	go m.loop()
	return m
}

func (m *mux) loop() {
	for {
		m.Lock()
		link := m.link
		m.Unlock()
		if link == nil {
			break
		}
		buf, addr, err := link.Read()
		if err != nil {
			break
		}
		pair, cargo, err := readSwitchHeader(buf)
		if err != nil {
			continue
		}
		flow := m.findFlow(&pair.Dest)
		if flow != nil {
			flow = m.acceptFlow(pair, cargo, addr)
		}
		flow.ch <- muxHeader{pair, cargo}
	}
	close(m.acceptChan)
	m.Lock()
	for _, flow := range m.flows {
		close(flow.ch)
	}
	m.link = nil
	m.Unlock()
}

func (m *mux) acceptFlow(pair *FlowPair, cargo []byte, remote *LinkAddr) *flow {
	m.Lock()
	defer m.Unlock()

	ch := make(chan muxHeader)
	p := &FlowPair{}
	p.Source.Choose()
	p.Dest = pair.Source
	f = newFlow(remote, m, ch, pair)
	m.flows = append(m.flows, f)

	return f
}

// ReadSwitchHeader() reads a header consisting of switch-specific flow information followed by data
func readSwitchHeader(p []byte) (pair *FlowPair, cargo []byte, err os.Error) {
	pair = &FlowPair{}
	n, err := pair.Read(p)
	if err != nil {
		return nil, nil, err
	}
	cargo = p[n:]
	return pair, cargo, nil
}

// Accept() returns the first incoming flow request
func (m *mux) Accept() (c net.Conn, err os.Error) {
	f, closed := <-m.acceptChan
	if closed {
		return nil, os.EBADF
	}
	return f
}

// findFlow() checks if there already exists a flow with the given local key
func (m *mux) findFlow(key *Label) *flow {
	m.lk.Lock()
	defer m.lk.Unlock()

	for _, flow := range m.flows {
		if flow.pair.Source.Equals(key) {
			return flow
		}
	}
	return nil
}

// addr@ is a textual representation of a link-level address, e.g.
//   0011`2233`4455`6677`8899`aabb`ccdd`eeff:453
func (m *mux) Dial(addr string) (flow net.Conn, err os.Error) {
	a := &LinkAddr{}
	if _, err = a.Parse(addr); err != nil {
		return nil, err
	}
	return m.DialAddr(a)
}

func (m *mux) DialAddr(remote *LinkAddr) (flow net.Conn, err os.Error) {
	m.Lock()
	defer m.Unlock()

	ch := make(chan muxHeader)
	pair := &FlowPair{}
	pair.Source.Choose()
	f = newFlow(remote, m, ch, pair)
	// XXX: fill-in pair.Dest when a packet arrives; update flow entry; lock on flow entry everywhere
	// or lookup flows by eother source or dest address
	m.flows = append(m.flows, f)

	return f, nil
}

// delFlow() removes the flow with the specified local key from the data structure
func (m *mux) delFlow(key *Label) {
	m.Lock()
	defer m.Unlock()

	for i, f := range m.flows {
		if f.pair.Source.Equals(key) {
			l := len(m.flows)
			m.flows[i] = m.flows[l-1]
			m.flows = m.flows[:l-1]
			return
		}
	}
	panic("unreach")
}

func (m *mux) write(pair *FlowPair, q []byte, remote *LinkAddr) (n int, err os.Error) {
	p := *pair
	p.Flip()
	buf := make([]byte, p.Len() + len(q))
	p.Write(buf)
	copy(buf[p.Len():], q)
	
	m.Lock()
	link := m.link
	m.Unlock()
	if link == nil {
		return 0, os.EBADF
	}

	n, err = link.Write(buf, remote)
	return max(0, n-p.Len()), err
}

func max(i,j int) int {
	if i > j {
		return i
	}
	return j
}

func (m *phsyicalSwitch) Close() os.Error {
	m.Lock()
	link := m.link
	m.link = nil
	m.Unlock()
	return link.Close()
}

// Now() returns the current time in nanoseconds
func (m *mux) Now() int64 { return time.Now() }
