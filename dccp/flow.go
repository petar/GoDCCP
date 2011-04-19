// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// FlowID{} contains identifiers of the local and remote logical addresses.
// These are usually random numbers created by the handshake. They intentionally
// resemble an IPv6 address/port pair in order to be fit DCCP's checksum routines
// which know how to handle IPv6 address/port end-point identifiers.
type FlowID struct {
	SourceID, DestID [FlowIDLength]byte
	SourcePort, DestPort uint16
}
const FlowIDLength = 16

var ZeroFlowID = FlowID{}

// Read reads the flow ID from the wire format
func (f *FlowID) Read(p []byte) (n int, err os.Error) {
	if len(p) < 2*(FlowIDLength+2) {
		return 0, os.NewError("flow header too short")
	}
	for i := 0; i < FlowIDLength; i++ {
		f.SourceID[i] = p[i]
		f.DestID[i] = p[FlowIDLength+i]
	}
	p = p[2*FlowIDLength:]
	f.SourcePort = decode2ByteUint(p[0:2])
	f.DestPort = decode2ByteUint(p[2:4])
	return 2*(FlowIDLength+2), nil
}

// Write writes the flow ID in wire format
func (f *FlowID) Write(p []byte) (n int, err os.Error) {
	if len(p) < 2*(FlowIDLength+2) {
		return 0, os.NewError("flow header can't fit")
	}
	for i, b := range f.SourceID {
		p[i] = b
	}
	for i, b := range f.DestID {
		p[FlowIDLength+i] = b
	}
	p = p[2*FlowIDLength:]
	encode2ByteUint(f.SourcePort, p[0:2])
	encode2ByteUint(f.DestPort, p[2:4])
	return 2*(FlowIDLength+2), nil
}
