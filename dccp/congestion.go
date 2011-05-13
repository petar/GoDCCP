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

	// Conn calls PreWrite before a packet is sent to give CongestionControl
	// an opportunity to add options to an outgoing packet
	PreWrite(h *Header)

	// Conn calls PostRead after a packet has been accepted as valid 
	PostRead(h *Header)

	// Injection ...
}

const (
	CCID2 = 2 // TCP-like Congestion Control, RFC 4341
	CCID3 = 3 // TCP-Friendly Rate Control (TFRC), RFC 4342
	CCIDPlain = 8	// Simple constant-rate control for testing purposes
)
