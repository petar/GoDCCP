// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
	"sync"
)

// connSwitch{} helps multiplex the physical layer among 
// multiple connections based on logical flow ID.
type connSwitch struct {
	phy    Physical
	flows  []*connFlow	// TODO: Lookups in a short array should be fine for now. Hashing?
	rest   chan connReadResult
	lk     sync.Mutex
}
type connReadResult struct {
	buf    []byte
	flowid *PhysicalFlowID
}

func newConnSwitch(phy Physical) *connSwitch {
	cswtch := &connSwitch{ 
		phy:   phy, 
		flows: make([]*connFlow),
		rest:  make(chan connReadResult),
	}
	go cswtch.loop()
	return cswtch
}

func (cswtch *connSwitch) loop() {
	for {
		cswtch.lk.Lock()
		phy := cswtch.phy
		cswtch.lk.Unlock()
		if phy == nil {
			break
		}
		buf, flowid, err := phy.Read()
		if err != nil {
			break
		}
		cflow := cswtch.findFlow(flowid)
		if cflow != nil {
			cflow.ch <- buf
		} else {
			cswtch.rest <- connReadResult{buf, flowid}
		}
	}
	close(cswtch.rest)
	cswtch.lk.Lock()
	for _, cflow := range cswtch.flows {
		close(cswtch.ch)
	}
	cswtch.phy = nil
	cswtch.lk.Unlock()
}

func (cswtch *connSwitch) findFlow(flowid *PhysicalFlowID) *physicalFlow {
	cswtch.lk.Lock()
	defer cswtch.lk.Unlock()
	for _, cflow := range cswtch.flows {
		if cflow.PhysicalFlowID == flowid {	// Pointer comparison is OK!
			return cflow
		}
	}
	return nil
}

func (cswtch *connSwitch) MakeFlow(flowid *PhysicalFlowID) *physicalFlow {
	cswtch.lk.Lock()
	defer cswtch.lk.Unlock()
	ch := make(chan []buf)
	cswtch.flows = append(cswtch.flows, flowid)
	return &physicalFlow{
		PhysicalFlowID: flowid,
		phy:    cswtch,
		ch:     ch,
	}
}

func (cswtch *connSwitch) delFlow(flowid *PhysicalFlowID) {
	cswtch.lk.Lock()
	defer cswtch.lk.Unlock()
	for i, cflow := range cswtch.flows {
		if cflow.PhysicalFlowID == flowid {	// Pointer comparison is OK!
			l := len(cswtch.flows)
			cswtch.flows[i] = cswtch.flows[l-1]
			cswtch.flows = cswtch.flows[:l-1]
			return
		}
	}
	panic("unreach")
}

func (cswtch *connSwitch) Write(buf []byte, flowid *PhysicalFlowID) os.Error {
	cswtch.lk.Lock()
	phy := cswtch.phy
	cswtch.lk.Unlock()
	if phy == nil {
		return os.EBADF
	}
	return phy.Write(buf, flowid)
}

func (cswtch *connSwitch) Read() (buf []byte, flowid *PhysicalFlowID, err os.Error) {
	r := <-cswtch.rest
	if r.buf == nil {
		err = os.EBADF
	}
	return r.buf, r.flowid, err
}

func (cswtch *phsyicalSwitch) Close() os.Error {
	cswtch.lk.Lock()
	phy := cswtch.phy
	cswtch.phy = nil
	cswtch.lk.Unlock()
	return phy.Close()
}

func (cswtch *connSwitch) Now() int64 { return time.Now() }

// connFlow{} acts as a packet ReadWriteCloser{} for Conn.
type connFlow struct {
	*FlowID
	cswtch *listener
	ch     chan []byte
}

func (cflow *connFlow) Write(buf []byte) os.Error {
	return cflow.cswtch.Write(buf, cflow.FlowID)
}

func (cflow *connFlow) Read() (buf []byte, err os.Error) {
	buf = <-cflow.ch
	if buf == nil {
		err = os.EBADF
	}
	return 
}

func (cflow *connFlow) Now() int64 { return cflow.cswtch.Now() }

func (cflow *connFlow) Close() os.Error {
	cflow.cswtch.delFlow(cflow)
	return nil
}

