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
	OnWrite(htype byte, x bool, seqno int64) (ccval byte, options []*Option)

	// Conn calls OnRead after a packet has been accepted an validated
	// If OnRead returns ErrDrop, the packet will be dropped and no further processing
	// will occur. If if it returns ErrReset, the connection will be reset.
	// TODO: ErrReset behavior is not implemented. ErrReset should wrap the Reset Code to be
	// used.
	OnRead(htype byte, x bool, seqno int64, options []*Option) os.Error

	// Strobe blocks until a new packet can be sent without violating the
	// congestion control rate limit
	Strobe() os.Error

	// ??

	// Close terminates the half-connection congestion control when it is not needed any longer
	Close() os.Error
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
	OnWrite(htype byte, x bool, seqno int64) (options []*Option)

	// Conn calls OnRead after a packet has been accepted and validated
	// If OnRead returns ErrDrop, the packet will be dropped and no further processing
	// will occur. If if it returns ErrReset, the connection will be reset.
	// TODO: ErrReset behavior is not implemented. ErrReset should wrap the Reset Code to be
	// used.
	OnRead(htype byte, x bool, seqno int64, ccval byte, options []*Option) os.Error

	// ??

	// Close terminates the half-connection congestion control when it is not needed any longer
	Close() os.Error
}

type NewSenderCongestionControlFunc func() SenderCongestionControl
type NewReceiverCongestionControlFunc func() ReceiverCongestionControl

const (
	CCID2      = 2 // TCP-like Congestion Control, RFC 4341
	CCID3      = 3 // TCP-Friendly Rate Control (TFRC), RFC 4342
)

//  ---> Sender Congestion Control Activator

func newActivatorForSenderCongestionControl(scc SenderCongestionControl) SenderCongestionControl {
	return &senderCongestionControlActivator{ phase: ACTIVATOR_INIT, SenderCongestionControl: scc }
}

type senderCongestionControlActivator struct {
	Mutex
	phase byte
	SenderCongestionControl
}
const (
	ACTIVATOR_INIT = iota
	ACTIVATOR_OPEN
	ACTIVATOR_CLOSED
)

func (sa *senderCongestionControlActivator) Open() {
	sa.Lock()
	defer sa.Unlock()
	if sa.phase != ACTIVATOR_INIT {
		return
	}
	sa.phase = ACTIVATOR_OPEN
	sa.SenderCongestionControl.Open()
}

func (sa *senderCongestionControlActivator) OnWrite(htype byte, x bool, seqno int64) (ccval byte, options []*Option) {
	sa.Lock()
	defer sa.Unlock()
	if sa.phase == ACTIVATOR_OPEN {
		return sa.SenderCongestionControl.OnWrite(htype, x, seqno)
	}
	return 0, nil
}

func (sa *senderCongestionControlActivator) OnRead(htype byte, x bool, seqno int64, options []*Option) os.Error {
	sa.Lock()
	defer sa.Unlock()
	if sa.phase == ACTIVATOR_OPEN {
		return sa.SenderCongestionControl.OnRead(htype, x, seqno, options)
	}
	return nil
}

func (sa *senderCongestionControlActivator) Strobe() os.Error {
	sa.Lock()
	defer sa.Unlock()
	if sa.phase == ACTIVATOR_OPEN {
		return sa.SenderCongestionControl.Strobe()
	}
	return nil
}

func (sa *senderCongestionControlActivator) Close() os.Error {
	sa.Lock()
	defer sa.Unlock()
	if sa.phase != ACTIVATOR_OPEN {
		return nil
	}
	sa.phase = ACTIVATOR_CLOSED
	return sa.SenderCongestionControl.Close()
}

//  ---> Receiver Congestion Control Activator

func newActivatorForReceiverCongestionControl(rcc ReceiverCongestionControl) ReceiverCongestionControl {
	return &receiverCongestionControlActivator{ phase: ACTIVATOR_INIT, ReceiverCongestionControl: rcc }
}

type receiverCongestionControlActivator struct {
	Mutex
	phase byte
	ReceiverCongestionControl
}

func (ra *receiverCongestionControlActivator) Open() {
	ra.Lock()
	defer ra.Unlock()
	if ra.phase != ACTIVATOR_INIT {
		return
	}
	ra.phase = ACTIVATOR_OPEN
	ra.ReceiverCongestionControl.Open()
}

func (ra *receiverCongestionControlActivator) OnWrite(htype byte, x bool, seqno int64) (options []*Option) {
	ra.Lock()
	defer ra.Unlock()
	if ra.phase == ACTIVATOR_OPEN {
		return ra.ReceiverCongestionControl.OnWrite(htype, x, seqno)
	}
	return nil
}

func (ra *receiverCongestionControlActivator) OnRead(htype byte, x bool, seqno int64, ccval byte, options []*Option) os.Error {
	ra.Lock()
	defer ra.Unlock()
	if ra.phase == ACTIVATOR_OPEN {
		return ra.ReceiverCongestionControl.OnRead(htype, x, seqno, ccval, options)
	}
	return nil
}

func (ra *receiverCongestionControlActivator) Close() os.Error {
	ra.Lock()
	defer ra.Unlock()
	if ra.phase != ACTIVATOR_OPEN {
		return nil
	}
	ra.phase = ACTIVATOR_CLOSED
	return ra.ReceiverCongestionControl.Close()
}
