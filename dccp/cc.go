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

	// Start tells the Congestion Control that it is being put in use.
	// This method is handy since CC is generally time-sensitive, and so having
	// an indication of "start" allows the CC to distinguish between its creation
	// time and the time when it actually starts being utilized.
	Start()

	// GetID() returns the CCID of this congestion control algorithm
	GetID() byte

	// GetCCMPS returns the Congestion Control Maximum Packet Size, CCMPS. Generally, PMTU <= CCMPS
	GetCCMPS() int32

	// GetRTT returns the Round-Trip Time as measured by this CCID
	GetRTT() int64

	// Conn calls OnWrite before a packet is sent to give CongestionControl
	// an opportunity to add CCVal and options to an outgoing packet
	OnWrite(htype byte, x bool, seqno int64) (ccval byte, options []Option)

	// Conn calls OnRead after a packet has been accepted an validated
	// If OnRead returns ErrDrop, the packet will be dropped and no further processing
	// will occur. If if it returns ErrReset, the connection will be reset.
	// TODO: ErrReset behavior is not implemented. ErrReset should wrap the Reset Code to be
	// used.
	OnRead(htype byte, x bool, seqno int64, options []Option) os.Error

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

	// Start tells the Congestion Control that it is being put in use.
	// This method is handy since CC is generally time-sensitive, and so having
	// an indication of "start" allows the CC to distinguish between its creation
	// time and the time when it actually starts being utilized.
	Start()

	// GetID() returns the CCID of this congestion control algorithm
	GetID() byte

	// Conn calls OnWrite before a packet is sent to give CongestionControl
	// an opportunity to add CCVal and options to an outgoing packet
	OnWrite(htype byte, x bool, seqno int64) (options []Option)

	// Conn calls OnRead after a packet has been accepted and validated
	// If OnRead returns ErrDrop, the packet will be dropped and no further processing
	// will occur. If if it returns ErrReset, the connection will be reset.
	// TODO: ErrReset behavior is not implemented. ErrReset should wrap the Reset Code to be
	// used.
	OnRead(htype byte, x bool, seqno int64, ccval byte, options []Option) os.Error

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
