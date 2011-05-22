// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// CongestionControl abstracts away the congestion control logic of a 
// DCCP connection.
type CongestionControl interface {

	// Returns the Congestion Control Maximum Packet Size, CCMPS. Generally, PMTU <= CCMPS
	GetCCMPS() uint32 

	// Conn calls OnWrite before a packet is sent to give CongestionControl
	// an opportunity to add options to an outgoing packet
	OnWrite(h *Header)

	// Conn calls OnRead after a packet has been accepted an validated
	// If OnRead returns ErrDrop, the packet will be dropped and no further processing
	// will occur. If if it returns ErrReset, the connection will be reset.
	OnRead(h *Header) os.Error

	// Strobe blocks until a new packet can be sent without violating the
	// congestion control rate limit
	Strobe()
}

const (
	CCID2 = 2 // TCP-like Congestion Control, RFC 4341
	CCID3 = 3 // TCP-Friendly Rate Control (TFRC), RFC 4342
	CCIDPlain = 8	// Simple constant-rate control for testing purposes
)
