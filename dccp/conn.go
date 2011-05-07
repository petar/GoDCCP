// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
)

// Conn 
type Conn struct {
	socket
	hc headerConn
}

type socket struct {
	sync.Mutex
	RTT		uint64	// Round-trip time

	ISS		uint64	// Initial Sequence number Sent
	ISR		uint64	// Initial Sequence number Received
	GSS		uint64	// Greatest Sequence number Sent
				//	The greatest SeqNo of a packet sent by this endpoint
	GSR		uint64	// Greatest Sequence number Received (consequently, sent as AckNo back)
				//	The greatest SeqNo of a packet received by the other endpoint
	GAR		uint64	// Greatest Acknowledgement number Received

	OSR		uint64	// First OPEN Sequence number Received

	SWBF		uint64	// Sequence Window/B Feature
	SWAF		uint64	// Sequence Window/A Feature

	MPS		uint32	// Maximum Packet Size
				// The MPS is influenced by the
				// maximum packet size allowed by the current congestion control
				// mechanism (CCMPS), the maximum packet size supported by the path's
				// links (PMTU, the Path Maximum Transmission Unit) [RFC1191], and the
				// lengths of the IP and DCCP headers.

	State		int
	ServiceCode	uint32
}
