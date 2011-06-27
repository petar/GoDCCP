// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"github.com/petar/GoDCCP/dccp"
)

type receiver struct {
	dccp.Mutex
	rttReceiver
	receiveRate
	lossEvents
}

// GetID() returns the CCID of this congestion control algorithm
func (r *receiver) GetID() byte { return dccp.CCID3 }

// Open tells the Congestion Control that the connection has entered
// OPEN or PARTOPEN state and that the CC can now kick in.
func (r *receiver) Open() {
	r.rttReceiver.Init()
	r.receiveRate.Init()
	r.lossEvents.Init()
	?
}

// Conn calls OnWrite before a packet is sent to give CongestionControl
// an opportunity to add CCVal and options to an outgoing packet
// NOTE: If the CC is not active, OnWrite MUST return nil.
func (r *receiver) OnWrite(htype byte, x bool, seqno int64) (options []*dccp.Option) {
	r.Lock()
	defer r.Unlock()
	?
}

// Conn calls OnRead after a packet has been accepted and validated
// If OnRead returns ErrDrop, the packet will be dropped and no further processing
// will occur. 
// NOTE: If the CC is not active, OnRead MUST return nil.
func (r *receiver) OnRead(htype byte, x bool, seqno int64, ccval byte, options []*dccp.Option) os.Error {
	r.Lock()
	defer r.Unlock()
	?
}

// OnIdle behaves identically to the same method of the HC-Sender CCID
func (r *receiver) OnIdle() os.Error {
	r.Lock()
	defer r.Unlock()
	?
}

// Close terminates the half-connection congestion control when it is not needed any longer
func (r *receiver) Close() {
	?
}
