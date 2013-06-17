// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "net"

// Bytes is a type that has an equivalent representation as a byte slice
// We use it for addresses, since net.Addr does not require such representation
type Bytes interface {
	Bytes() []byte
}

// A SegmentConn is an I/O facility that explicitly reads/writes data in the form
// of indivisible blocks of data. Implementors of this interface MUST only return
// i/o errors defined in the dccp package (ErrEOF, ErrBad, ErrTimeout, etc.)
type SegmentConn interface {
	// GetMTU returns th he largest allowable block size (for read and write). The MTU may vary.
	GetMTU() int

	// Read returns an ErrTimeout in the event of a timeout. See SetReadExpire.
	Read() (block []byte, err error)

	// If the user attempts to write a block that is too big, an ErrTooBig is returned
	// and the block is not sent.
	Write(block []byte) (err error)

	// SetReadExpire sets the expiration time for any blocked calls to Read
	// as a time represented in nanoseconds from now. It's semantics are similar to that
	// of net.Conn.SetReadDeadline except that the deadline is specified in time from now,
	// rather than absolute time. Also note that Read is expected to return 
	// an ErrTimeout in the event of timeouts.
	SetReadExpire(nsec int64) error

	Close() error
}

// SegmentDialAccepter represents a type that can accept and dial lossy packet connections
type SegmentDialAccepter interface {
	Accept() (c SegmentConn, err error)
	Dial(addr net.Addr) (c SegmentConn, err error)
	Close() error
}

// Implementors of this interface MUST only return i/o errors defined in the dccp package (ErrEOF,
// ErrBad, ErrTimeout, etc.)
type HeaderConn interface {
	// GetMTU() returns the Maximum Transmission Unit size. This is the maximum
	// byte size of the header and app data wire-format footprint.
	GetMTU() int

	// Read returns ErrTimeout in the event of timeout. See SetReadExpire.
	Read() (h *Header, err error)

	// Write can return ErrTooBig, if the wire-format of h exceeds the MTU
	Write(h *Header) (err error)

	// SetReadExpire behaves similarly to SegmentConn.SetReadExpire
	SetReadExpire(nsec int64) error

	Close() error
}

// —————
// NewHeaderConn creates a HeaderConn on top of a SegmentConn
func NewHeaderConn(bc SegmentConn) HeaderConn {
	return &headerConn{bc: bc}
}

type headerConn struct {
	bc SegmentConn
}

func (hc *headerConn) GetMTU() int {
	return hc.bc.GetMTU()
}

// Since a SegmentConn already has the notion of a flow, both Read
// and Write pass zero labels for the Source and Dest IPs
// to the DCCP header's read and write functions.

func (hc *headerConn) Read() (h *Header, err error) {
	p, err := hc.bc.Read()
	if err != nil {
		return nil, err
	}
	return ReadHeader(p, LabelZero.Bytes(), LabelZero.Bytes(), AnyProto, false)
}

func (hc *headerConn) Write(h *Header) (err error) {
	p, err := h.Write(LabelZero.Bytes(), LabelZero.Bytes(), AnyProto, false)
	if err != nil {
		return err
	}
	return hc.bc.Write(p)
}

func (hc *headerConn) SetReadExpire(nsec int64) error {
	return hc.bc.SetReadExpire(nsec)
}

func (hc *headerConn) Close() error {
	return hc.bc.Close()
}
