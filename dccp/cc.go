// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// Regarding options and Half-Connection CCIDs (from Section 10.3):
//
// Any packet may contain information meant for either half-connection,
// so CCID-specific option types, feature numbers, and Reset Codes
// explicitly signal the half-connection to which they apply.
//
// o  Option numbers 128 through 191 are for options sent from the
//    HC-Sender to the HC-Receiver; option numbers 192 through 255 are
//    for options sent from the HC-Receiver to the HC-Sender.
//
// o  Reset Codes 128 through 191 indicate that the HC-Sender reset the
//    connection (most likely because of some problem with
//    acknowledgements sent by the HC-Receiver).  Reset Codes 192
//    through 255 indicate that the HC-Receiver reset the connection
//    (most likely because of some problem with data packets sent by the
//    HC-Sender).
//
// o  Finally, feature numbers 128 through 191 are used for features
//    located at the HC-Sender; feature numbers 192 through 255 are for
//    features located at the HC-Receiver.  Since Change L and Confirm L
//    options for a feature are sent by the feature location, we know
//    that any Change L(128) option was sent by the HC-Sender, while any
//    Change L(192) option was sent by the HC-Receiver.  Similarly,
//    Change R(128) options are sent by the HC-Receiver, while Change
//    R(192) options are sent by the HC-Sender.


// SenderCongestionControl specifies the interface for the congestion control logic of a DCCP
// sender (aka Half-Connection Sender CCID)
type SenderCongestionControl interface {

	// The congestion control is considered active between a call to Open and a call to Close or
	// an internal event that forces closure (like a reset event).

	// GetID() returns the CCID of this congestion control algorithm
	GetID() byte

	// GetCCMPS returns the Congestion Control Maximum Packet Size, CCMPS. Generally, PMTU <= CCMPS
	GetCCMPS() int32

	// GetRTT returns the Round-Trip Time as measured by this CCID
	GetRTT() int64

	// Open tells the Congestion Control that the connection has entered
	// OPEN or PARTOPEN state and that the CC can now kick in. Before the
	// call to Open and after the call to Close, the Strobe function is
	// expected to return immediately.
	Open()

	// Conn calls OnWrite before a packet is sent to give CongestionControl
	// an opportunity to add CCVal and options to an outgoing packet
	// NOTE: If the CC is not active, OnWrite should return 0, nil.
	OnWrite(htype byte, x bool, seqno int64) (ccval byte, options []*Option)

	// Conn calls OnRead after a packet has been accepted and validated
	// If OnRead returns ErrDrop, the packet will be dropped and no further processing
	// will occur. If OnRead returns ResetError, the connection will be reset.
	// NOTE: If the CC is not active, OnRead MUST return nil.
	OnRead(htype byte, x bool, seqno int64, options []*Option) os.Error

	// Strobe blocks until a new packet can be sent without violating the
	// congestion control rate limit. 
	// NOTE: If the CC is not active, Strobe MUST return immediately.
	Strobe()

	// OnIdle is called periodically, giving the CC a chance to:
        // (a) Request a connection reset by returning a CongestionReset, or
	// (b) Request the injection of an Ack packet by returning a CongestionAck
	// NOTE: If the CC is not active, OnIdle MUST to return nil.
	OnIdle() os.Error

	// Close terminates the half-connection congestion control when it is not needed any longer
	Close()
}

// ReceiverCongestionControl specifies the interface for the congestion control logic of a DCCP
// receiver (aka Half-Connection Receiver CCID)
type ReceiverCongestionControl interface {

	// GetID() returns the CCID of this congestion control algorithm
	GetID() byte

	// Open tells the Congestion Control that the connection has entered
	// OPEN or PARTOPEN state and that the CC can now kick in.
	Open()

	// Conn calls OnWrite before a packet is sent to give CongestionControl
	// an opportunity to add CCVal and options to an outgoing packet
	// NOTE: If the CC is not active, OnWrite MUST return nil.
	OnWrite(htype byte, x bool, seqno int64) (options []*Option)

	// Conn calls OnRead after a packet has been accepted and validated
	// If OnRead returns ErrDrop, the packet will be dropped and no further processing
	// will occur. 
	// NOTE: If the CC is not active, OnRead MUST return nil.
	OnRead(htype byte, x bool, seqno int64, ccval byte, options []*Option) os.Error

	// OnIdle behaves identically to the same method of the HC-Sender CCID
	OnIdle() os.Error

	// Close terminates the half-connection congestion control when it is not needed any longer
	Close()
}

type NewSenderCongestionControlFunc func() SenderCongestionControl
type NewReceiverCongestionControlFunc func() ReceiverCongestionControl

const (
	CCID2      = 2 // TCP-like Congestion Control, RFC 4341
	CCID3      = 3 // TCP-Friendly Rate Control (TFRC), RFC 4342
)

// ---> Sender CC actuator

// The actuator makes sure that the underlying Open/Close objects sees
// exactly one call to Open and one call to Close (after the call to Open).

func newSenderCCActuator(scc SenderCongestionControl) SenderCongestionControl {
	return &senderCCActuator{ 
		SenderCongestionControl: scc,
		phase:                   ACTUATOR_INIT,
		cls:                     make(chan int),
	}
}

const (
	ACTUATOR_INIT = iota
	ACTUATOR_OPEN
	ACTUATOR_CLOSED
)

type senderCCActuator struct {
	SenderCongestionControl
	Mutex
	phase byte
	cls   chan int
}

func (a *senderCCActuator) Open() {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_INIT {
		return
	}
	a.phase = ACTUATOR_OPEN
	a.SenderCongestionControl.Open()
}

func (a *senderCCActuator) Close() {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_OPEN {
		return
	}
	a.phase = ACTUATOR_CLOSED
	if a.cls != nil {
		close(a.cls)
		a.cls = nil
	}
	a.SenderCongestionControl.Close()
}

func (a *senderCCActuator) Strobe() {
	a.Lock()
	cls := a.cls
	if a.phase != ACTUATOR_OPEN || cls == nil {
		a.Unlock()
		return
	}
	a.Unlock()

	go func() {
		a.SenderCongestionControl.Strobe()
		a.Lock()
		defer a.Unlock()
		if a.cls != nil {
			a.cls <- 1
		}
	}()

	// This unblocks if (i) either Close is called or (ii) the underlying Strobe returns
	<-cls
}

func (a *senderCCActuator) OnWrite(htype byte, x bool, seqno int64) (ccval byte, options []*Option) {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_OPEN {
		return 0, nil
	}
	return a.SenderCongestionControl.OnWrite(htype, x, seqno)
}

func (a *senderCCActuator) OnRead(htype byte, x bool, seqno int64, options []*Option) os.Error {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_OPEN {
		return nil
	}
	return a.SenderCongestionControl.OnRead(htype, x, seqno, options)
}

func (a *senderCCActuator) OnIdle() os.Error {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_OPEN {
		return nil
	}
	return a.SenderCongestionControl.OnIdle()
}

// ---> Receiver CC actuator

func newReceiverCCActuator(rcc ReceiverCongestionControl) ReceiverCongestionControl {
	return &receiverCCActuator{ 
		ReceiverCongestionControl: rcc,
		phase:                     ACTUATOR_INIT,
	}
}

type receiverCCActuator struct {
	ReceiverCongestionControl
	Mutex
	phase byte
}

func (a *receiverCCActuator) Open() {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_INIT {
		return
	}
	a.phase = ACTUATOR_OPEN
	a.ReceiverCongestionControl.Open()
}

func (a *receiverCCActuator) Close() {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_OPEN {
		return
	}
	a.phase = ACTUATOR_CLOSED
	a.ReceiverCongestionControl.Close()
}
	
func (a *receiverCCActuator) OnWrite(htype byte, x bool, seqno int64) (options []*Option) {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_OPEN {
		return nil
	}
	return a.ReceiverCongestionControl.OnWrite(htype, x, seqno)
}

func (a *receiverCCActuator) OnRead(htype byte, x bool, seqno int64, ccval byte, options []*Option) os.Error {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_OPEN {
		return nil
	}
	return a.ReceiverCongestionControl.OnRead(htype, x, seqno, ccval, options)
}

func (a *receiverCCActuator) OnIdle() os.Error {
	a.Lock()
	defer a.Unlock()
	if a.phase != ACTUATOR_OPEN {
		return nil
	}
	return a.ReceiverCongestionControl.OnIdle()
}
