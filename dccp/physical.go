// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
	"sync"
)

// Physical{} is an abstract interface to a physical packet layer which
// sends and receives packets and supports a notion of time
type Physical interface {

	Send(buf []byte, flowid *FlowID) os.Error		// Send a block of data
	Receive() (buf []byte, flowid *FlowID, err os.Error)	// Receive next block of data
	Close() os.Error
}

type FlowID struct {
	SourceAddr, DestAddr	[]byte
	SourcePort, DestPort	uint16
}

// physicalSwitch{} helps multiplex the physical layer among 
// a listener type and connection types
// 
type physicalSwitch struct {
	phy    Physical
	flows  []*physicalFlow	// TODO: Lookups in a short array should be fine for now. Hashing?
	rest   chan receiveResult
	lk     sync.Mutex
}
type receiveResult struct {
	buf    []byte
	flowid *FlowID
}

func newPhysicalSwitch(phy Physical) *physicalSwitch {
	phsw := &physicalSwitch{ 
		phy:   phy, 
		flows: make([]*physicalFlow),
		rest:  make(chan receiveResult),
	}
	go phsw.readLoop()
	return phsw
}

func (phsw *physocalSwitch) readLoop() {
	for {
		phsw.lk.Lock()
		phy := phsw.phy
		phsw.lk.Unlock()
		if phy == nil {
			break
		}
		buf, flowid, err := phy.Receive()
		if err != nil {
			break
		}
		phfl := phsw.findFlow(flowid)
		if phfl != nil {
			phfl.ch <- buf
		} else {
			phsw.rest <- receiveResult{buf, flowid}
		}
	}
	close(phsw.rest)
	phsw.lk.Lock()
	for _, phfl := range phsw.flows {
		close(phsw.ch)
	}
	phsw.phy = nil
	phsw.lk.Unlock()
}

func (phsw *physicalSwitch) findFlow(flowid *FlowID) *physicalFlow {
	phsw.lk.Lock()
	defer phsw.lk.Unlock()
	for _, phfl := range phsw.flows {
		if phfl.FlowID == flowid {	// Pointer comparison is OK!
			return phfl
		}
	}
	return nil
}

func (phsw *physicalSwitch) MakeFlow(flowid *FlowID) *physicalFlow {
	phsw.lk.Lock()
	defer phsw.lk.Unlock()
	ch := make(chan []buf)
	phsw.flows = append(phsw.flows, flowid)
	return &physicalFlow{
		FlowID: flowid,
		phy:    phsw,
		ch:     ch,
	}
}

func (phsw *physicalSwitch) delFlow(flowid *FlowID) {
	phsw.lk.Lock()
	defer phsw.lk.Unlock()
	for i, phfl := range phsw.flows {
		if phfl.FlowID == flowid {	// Pointer comparison is OK!
			l := len(phsw.flows)
			phsw.flows[i] = phsw.flows[l-1]
			phsw.flows = phsw.flows[:l-1]
			return
		}
	}
	panic("unreach")
}

func (phsw *physicalSwitch) Send(buf []byte, flowid *FlowID) os.Error {
	phsw.lk.Lock()
	phy := phsw.phy
	phsw.lk.Unlock()
	if phy == nil {
		return os.EBADF
	}
	return phy.Send(buf, flowid)
}

func (phsw *physicalSwitch) Receive() (buf []byte, flowid *FlowID, err os.Error) {
	r := <-phsw.rest
	if r.buf == nil {
		err = os.EBADF
	}
	return r.buf, r.flowid, err
}

func (phsw *phsyicalSwitch) Close() os.Error {
	phsw.lk.Lock()
	phy := phsw.phy
	phsw.phy = nil
	phsw.lk.Unlock()
	return phy.Close()
}

func (phsw *physicalSwitch) Now() int64 { return time.Now() }

// physicalFlow{} is 
type physicalFlow struct {
	*FlowID
	phsw *physicalSwitch
	ch   chan []byte
}

func (phfl *physicalFlow) Send(buf []byte) os.Error {
	return phfl.phsw.Send(buf, phfl.FlowID)
}

func (phfl *physicalFlow) Receive() (buf []byte, err os.Error) {
	buf = <-phfl.ch
	if buf == nil {
		err = os.EBADF
	}
	return 
}

func (phfl *physicalFlow) Now() int64 { return phfl.phsw.Now() }

func (phfl *physicalFlow) Close() os.Error {
	phfl.phsw.delFlow(phfl)
	return nil
}
