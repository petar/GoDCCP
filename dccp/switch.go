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
	link  Link
	flows []*flow	// TODO: Lookups in a short array should be fine for now. Hashing?
	rest  chan connReadResult
}

// switchHeader{} is an internal data structure that carries a parsed switch packet,
// which contains a flow id and a generic DCCP header
type switchHeader struct {
	id     *ID
	header *GenericHeader
}

func newConnSwitch(link Link) *flowSwitch {
	swtch := &flowSwitch{ 
		link:  link, 
		flows: make([]*flow, 0),
		rest:  make(chan connReadResult),
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
		id, header, err := readSwitchHeader(buf)
		if err != nil {
			continue
		}
		// Is there a flow with this dest ID already?
		flow := swtch.findFlow(&id.Dest)
		if flow != nil {
			flow.ch <- switchHeader{id, header}
		} else {
			swtch.rest <- switchHeader{id, header}
		}
	}
	close(swtch.rest)
	swtch.Lock()
	for _, flow := range swtch.flows {
		close(flow.ch)
	}
	swtch.link = nil
	swtch.Unlock()
}

// ReadSwitchHeader() reads a header consisting of switch-specific flow information followed by a
// DCCP generic header
func readSwitchHeader(p []byte) (id *ID, header *GenericHeader, err os.Error) {
	id = &FlowID{}
	n, err := id.Read(p)
	if err != nil {
		return nil, nil, err
	}
	p = p[n:]
	header, err = ReadGenericHeader(p, id.SourceAddr, id.DestAddr, AnyProto, false)
	if err != nil {
		return nil, nil, err
	}
	return id, header, nil
}

// findFlow() checks if there already exists a flow with the given local IP
func (swtch *flowSwitch) findFlow(ip *IP) *flow {
	swtch.lk.Lock()
	defer swtch.lk.Unlock()

	for _, flow := range swtch.flows {
		if flow.id.Source.IP.Equals(ip) {
			return flow
		}
	}
	return nil
}

// addr@ is a textual representation of a flow address and port, e.g.
//   0011`2233`4455`6677`8899`aabb`ccdd`eeff:453
func (swtch *flowSwitch) Dial(addr string) (flow net.Conn, err os.Error) {
	swtch.lk.Lock()
	defer swtch.lk.Unlock()

	ch := make(chan switchHeader)
	flow = &flow{
		swtch: swtch,
		ch:    ch,
	}
	if _, err = flow.linkaddr.Parse(addr); err != nil {
		return nil, err
	}
	flow.id.Source.Init()

	swtch.flows = append(swtch.flows, flow)

	return flow, nil
}

XX

func (swtch *flowSwitch) delFlow(flowid *PhysicalFlowID) {
	swtch.lk.Lock()
	defer swtch.lk.Unlock()
	for i, flow := range swtch.flows {
		if flow.PhysicalFlowID == flowid {	// Pointer comparison is OK!
			l := len(swtch.flows)
			swtch.flows[i] = swtch.flows[l-1]
			swtch.flows = swtch.flows[:l-1]
			return
		}
	}
	panic("unreach")
}

func (swtch *flowSwitch) Write(buf []byte, flowid *PhysicalFlowID) os.Error {
	swtch.lk.Lock()
	link := swtch.link
	swtch.lk.Unlock()
	if link == nil {
		return os.EBADF
	}
	return link.Write(buf, flowid)
}

func (swtch *flowSwitch) Read() (buf []byte, flowid *PhysicalFlowID, err os.Error) {
	r := <-swtch.rest
	if r.buf == nil {
		err = os.EBADF
	}
	return r.buf, r.flowid, err
}

func (swtch *phsyicalSwitch) Close() os.Error {
	swtch.lk.Lock()
	link := swtch.link
	swtch.link = nil
	swtch.lk.Unlock()
	return link.Close()
}

func (swtch *flowSwitch) Now() int64 { return time.Now() }

// flow{} acts as a packet ReadWriteCloser{} for Conn.
type flow struct {
	linkaddr Addr
	id       ID
	swtch    *flowSwitch
	ch       chan switchHeader
}

func (flow *flow) Write(buf []byte) os.Error {
	return flow.swtch.Write(buf, flow.FlowID)
}

func (flow *flow) Read() (buf []byte, err os.Error) {
	buf = <-flow.ch
	if buf == nil {
		err = os.EBADF
	}
	return 
}

func (flow *flow) Close() os.Error {
	flow.swtch.delFlow(flow)
	return nil
}

