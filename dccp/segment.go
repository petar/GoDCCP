// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"os"
)

// Bytes is a type that has an equivalent representation as a byte slice
// We use it for addresses, since net.Addr does not require such representation
type Bytes interface {
	Bytes() []byte
}

// A SegmentConn is an I/O facility that explicitly reads/writes data in the form
// of indivisible blocks of data. 
type SegmentConn interface {
	// GetMTU returns th he largest allowable block size (for read and write). The MTU may vary.
	GetMTU() int

	ReadSegment() (block []byte, err os.Error)

	// If the user attempts to write a block that is too big, an ErrTooBig is returned
	// and the block is not sent.
	WriteSegment(block []byte) (err os.Error)

	LocalLabel() Bytes

	RemoteLabel() Bytes

	SetReadTimeout(nsec int64) os.Error

	Close() os.Error
}

// SegmentDialAccepter represents a type that can accept and dial lossy packet connections
type SegmentDialAccepter interface {
	Accept() (c SegmentConn, err os.Error)
	Dial(addr net.Addr) (c SegmentConn, err os.Error)
	Close() os.Error
}

type HeaderConn interface {
	// GetMTU() returns the Maximum Transmission Unit size. This is the maximum
	// byte size of the header and app data wire-format footprint.
	GetMTU() int

	// os.EAGAIN is returned in the event of timeout.
	ReadHeader() (h *Header, err os.Error)

	// WriteHeader can return ErrTooBig, if the wire-format of h exceeds the MTU
	WriteHeader(h *Header) (err os.Error)

	LocalLabel() Bytes

	RemoteLabel() Bytes

	SetReadTimeout(nsec int64) os.Error

	Close() os.Error
}

// —————
// NewHeaderConn creates a HeaderConn on top of a SegmentConn
func NewHeaderConn(bc SegmentConn) HeaderConn {
	return &headerConn{ bc: bc }
}

type headerConn struct {
	bc   SegmentConn
}

func (hob *headerConn) GetMTU() int { 
	return hob.bc.GetMTU() 
}

// Since a SegmentConn already has the notion of a flow, both ReadHeader
// and WriteHeader pass zero labels for the Source and Dest IPs
// to the DCCP header's read and write functions.

func (hob *headerConn) ReadHeader() (h *Header, err os.Error) {
	p, err := hob.bc.ReadSegment()
	if err != nil {
		return nil, err
	}
	return ReadHeader(p, LabelZero.Bytes(), LabelZero.Bytes(), AnyProto, false)
}

func (hob *headerConn) WriteHeader(h *Header) (err os.Error) {
	p, err := h.Write(LabelZero.Bytes(), LabelZero.Bytes(), AnyProto, false)
	if err != nil {
		return err
	}
	return hob.bc.WriteSegment(p)
}

func (hob *headerConn) LocalLabel() Bytes { 
	return hob.bc.LocalLabel() 
}

func (hob *headerConn) RemoteLabel() Bytes { 
	return hob.bc.RemoteLabel() 
}

func (hob *headerConn) SetReadTimeout(nsec int64) os.Error { 
	return hob.bc.SetReadTimeout(nsec) 
}

func (hob *headerConn) Close() os.Error { 
	return hob.bc.Close() 
}
