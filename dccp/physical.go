// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"io"
	"os"
	"sync"
)

type Reader interface {
	Read() (buf []byte, flowid *PhysicalFlowID, err os.Error)	// Receive next packet of data
}

type Writer interface {
	Write(buf []byte, flowid *PhysicalFlowID) os.Error		// Send a packet of data
}

// Physical{} is an abstract interface to a physical packet layer which
// sends and receives packets and supports a notion of time
type Physical interface {
	Reader
	Writer
	io.Closer
}

// PhysicalFlowID{} contains IP address/port identifiers of a source/destination pair.
type PhysicalFlowID struct {
	SourceAddr, DestAddr	[]byte
	SourcePort, DestPort	uint16
}
var zeroPhyFlowID = PhysicalFlowId{
	SourceAddr: make([]byte, 4),
	DestAddr: make([]byte, 4),
}

// physicalSwitch is presently not used, since we've opted for multiplexing
// packet flows on logical bases not physical basis. However, it may be useful
// to others realizations of DCCP.

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
	flowid *PhysicalFlowID
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

func (phsw *physicalSwitch) readLoop() {
	for {
		phsw.lk.Lock()
		phy := phsw.phy
		phsw.lk.Unlock()
		if phy == nil {
			break
		}
		buf, flowid, err := phy.Read()
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

func (phsw *physicalSwitch) findFlow(flowid *PhysicalFlowID) *physicalFlow {
	phsw.lk.Lock()
	defer phsw.lk.Unlock()
	for _, phfl := range phsw.flows {
		if phfl.PhysicalFlowID == flowid {	// Pointer comparison is OK!
			return phfl
		}
	}
	return nil
}

func (phsw *physicalSwitch) MakeFlow(flowid *PhysicalFlowID) *physicalFlow {
	phsw.lk.Lock()
	defer phsw.lk.Unlock()
	ch := make(chan []buf)
	phsw.flows = append(phsw.flows, flowid)
	return &physicalFlow{
		PhysicalFlowID: flowid,
		phy:    phsw,
		ch:     ch,
	}
}

func (phsw *physicalSwitch) delFlow(flowid *PhysicalFlowID) {
	phsw.lk.Lock()
	defer phsw.lk.Unlock()
	for i, phfl := range phsw.flows {
		if phfl.PhysicalFlowID == flowid {	// Pointer comparison is OK!
			l := len(phsw.flows)
			phsw.flows[i] = phsw.flows[l-1]
			phsw.flows = phsw.flows[:l-1]
			return
		}
	}
	panic("unreach")
}

func (phsw *physicalSwitch) Write(buf []byte, flowid *PhysicalFlowID) os.Error {
	phsw.lk.Lock()
	phy := phsw.phy
	phsw.lk.Unlock()
	if phy == nil {
		return os.EBADF
	}
	return phy.Write(buf, flowid)
}

func (phsw *physicalSwitch) Read() (buf []byte, flowid *PhysicalFlowID, err os.Error) {
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
	*PhysicalFlowID
	phsw *physicalSwitch
	ch   chan []byte
}

func (phfl *physicalFlow) Write(buf []byte) os.Error {
	return phfl.phsw.Write(buf, phfl.PhysicalFlowID)
}

func (phfl *physicalFlow) Read() (buf []byte, err os.Error) {
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
