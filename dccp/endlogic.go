// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"rand"
)

// Endpoint logic for a single Half-connection
type Endpoint struct {
	ISS uint64	// Initial Sequence number Sent
	ISR uint64	// Initial Sequence number Received
	GSS uint64	// Greatest Sequence number Sent
	GSR uint64	// Greatest Sequence number Received (sent as AckNo back to other endpoint)
	GAR uint64	// Greatest Acknowledgement number Received

	SWBF uint64	// Sequence Window/B Feature
	SWAF uint64	// Sequence Window/A Feature

}

func pickInitialSeqNo() uint64 { return uint64(rand.Int63()) & 0xffffff }

func maxu64(x,y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}
// XXX: In theory, long lived connections may wrap around the AckNo/SeqNo space
// in which case maxu64() should not be used below. This will never happen however
// if we are using 48-bit numbers exclusively

// SWL and SWH
func (e *Endpoint) SeqNoWindowLowAndHigh() (SWL uint64, SWH uint64) {
	return maxu64(e.GSR + 1 - e.SWBF/4, e.ISR), e.GSR + (3*e.SWBF)/4
}

// AWL and AWH
func (e *Endpoint) AckNoWindowLowAndHigh() (AWL uint64, AWH uint64) {
	return maxu64(e.GSS + 1 - e.SWAF, e.ISS), e.GSS
}

func (e *Endpoint) IsSeqNoValid(seqno uint64) bool {
	swl, swh := e.SeqNoWindowLowHigh()
	return swl <= seqno && seqno <= swh
}

func (e *Endpoint) IsAckNoValid(ackno uint64) bool {
	awl, awh := e.AckNoWindowLowHigh()
	return awl <= ackno && ackno <= awh
}
