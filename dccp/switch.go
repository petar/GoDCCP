// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
	"sync"
)

// flowSwitch{} helps multiplex the connection-less physical packets among 
// multiple logical connections based on their logical flow ID.
type flowSwitch struct {
	sync.Mutex
	link       Link
	flows      []*flow	// TODO: Lookups in a short array should be fine for now. Hashing?
	acceptChan chan *flow
}

// switchHeader{} is an internal data structure that carries a parsed switch packet,
// which contains a flow id and a generic DCCP header
type switchHeader struct {
	pair   *FlowPair
	cargo  []byte
}

func newConnSwitch(link Link) *flowSwitch {
	swtch := &flowSwitch{ 
		link:       link, 
		flows:      make([]*flow, 0),
		acceptChan: make(chan *flow),
	}
	go swtch.loop()
	return swtch
}

func (swtch *flowSwitch) loop() {
	for {
		swtch.Lock()
		link := swtch.link
		swtch.Unlock()
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
		flow := swtch.findFlow(&pair.Dest)
		if flow != nil {
			flow = swtch.acceptFlow(pair, cargo, addr)
		}
		flow.ch <- switchHeader{pair, cargo}
	}
	close(swtch.acceptChan)
	swtch.Lock()
	for _, flow := range swtch.flows {
		close(flow.ch)
	}
	swtch.link = nil
	swtch.Unlock()
}

func (swtch *flowSwitch) acceptFlow(pair *FlowPair, cargo []byte, linkaddr *LinkAddr) *flow {
	swtch.Lock()
	defer swtch.Unlock()

	ch := make(chan switchHeader)
	flow = &flow{
		linkaddr: *linkaddr,
		swtch:    swtch,
		ch:       ch,
	}
	flow.pair.Source.Choose()
	swtch.flows = append(swtch.flows, flow)

	return flow
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
func (swtch *flowSwitch) Accept() (c net.Conn, err os.Error) {
	f, closed := <-swtch.acceptChan
	if closed {
		return nil, os.EBADF
	}
	return f
}

// findFlow() checks if there already exists a flow with the given local key
func (swtch *flowSwitch) findFlow(key *FlowKey) *flow {
	swtch.lk.Lock()
	defer swtch.lk.Unlock()

	for _, flow := range swtch.flows {
		if flow.pair.Source.Equals(key) {
			return flow
		}
	}
	return nil
}

// addr@ is a textual representation of a link-level address, e.g.
//   0011`2233`4455`6677`8899`aabb`ccdd`eeff:453
func (swtch *flowSwitch) Dial(addr string) (flow net.Conn, err os.Error) {
	a := &LinkAddr{}
	if _, err = a.Parse(addr); err != nil {
		return nil, err
	}
	return swtch.DialAddr(a)
}

func (swtch *flowSwitch) DialAddr(addr *LinkAddr) (flow net.Conn, err os.Error) {
	swtch.Lock()
	defer swtch.Unlock()

	ch := make(chan switchHeader)
	flow = &flow{
		linkaddr: *addr,
		swtch:    swtch,
		ch:       ch,
	}
	flow.pair.Source.Choose()
	swtch.flows = append(swtch.flows, flow)

	return flow, nil
}

// delFlow() removes the flow with the specified local key from the data structure
func (swtch *flowSwitch) delFlow(key *FlowKey) {
	swtch.Lock()
	defer swtch.Unlock()

	for i, flow := range swtch.flows {
		if flow.pair.Source.Equals(key) {
			l := len(swtch.flows)
			swtch.flows[i] = swtch.flows[l-1]
			swtch.flows = swtch.flows[:l-1]
			return
		}
	}
	panic("unreach")
}

func (swtch *phsyicalSwitch) Close() os.Error {
	swtch.lk.Lock()
	link := swtch.link
	swtch.link = nil
	swtch.lk.Unlock()
	return link.Close()
}

// Now() returns the current time in nanoseconds
func (swtch *flowSwitch) Now() int64 { return time.Now() }
