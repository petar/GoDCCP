// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// CongestionControl abstracts away the congestion control logic of a 
// DCCP connection.
type CongestionControl interface {

	// GetID() returns the CCID of this congestion control algorithm
	GetID()

	// GetCCMPS returns the Congestion Control Maximum Packet Size, CCMPS. Generally, PMTU <= CCMPS
	GetCCMPS() uint32 

	// GetRTT returns the Round-Trip Time as measured by this CCID
	GetRTT() int64

	// GetSeqWinA returns the Sequence Window/A Feature, see Section 7.5.1
	GetSeqWinA() uint64

	// GetSeqWinB returns the Sequence Window/B Feature, see Section 7.5.1
	GetSeqWinB() uint64 

	// Conn calls OnWrite before a packet is sent to give CongestionControl
	// an opportunity to add CCVal and options to an outgoing packet
	OnWrite(htype byte, x bool, seqno uint64) (ccval byte, options []Option)

	// Conn calls OnRead after a packet has been accepted an validated
	// If OnRead returns ErrDrop, the packet will be dropped and no further processing
	// will occur. If if it returns ErrReset, the connection will be reset.
	// TODO: ErrReset behavior is not implemented. ErrReset should wrap the Reset Code to be
	// used.
	OnRead(htype byte, x bool, seqno uint64, ccval byte, options []Option) os.Error

	// Strobe blocks until a new packet can be sent without violating the
	// congestion control rate limit
	Strobe()
}

const (
	CCID2 = 2 // TCP-like Congestion Control, RFC 4341
	CCID3 = 3 // TCP-Friendly Rate Control (TFRC), RFC 4342
	CCID_PETAR = 7 // Simple constant-rate control for testing purposes
)
