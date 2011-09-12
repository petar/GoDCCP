// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"os"
)

// Bytes is a type that has an equivalent representation as a byte slice
// We use it for addresses, since net.Addr does not requires such representation
type Bytes interface {
	Bytes() []byte
}

// A BlockConn is an I/O facility that explicitly reads/writes data in the form
// of indivisible blocks of data. 
type BlockConn interface {
	// GetMTU returns th he largest allowable block size (for read and write). The MTU may vary.
	GetMTU() int

	ReadBlock() (block []byte, err os.Error)

	// If the user attempts to write a block that is too big, an ErrTooBig is returned
	// and the block is not sent.
	WriteBlock(block []byte) (err os.Error)

	LocalLabel() Bytes

	RemoteLabel() Bytes

	SetReadTimeout(nsec int64) os.Error

	Close() os.Error
}

// BlockDialListener represents a type that can accept and dial lossy packet connections
type BlockDialListener interface {
	Accept() (c BlockConn, err os.Error)
	Dial(addr net.Addr) (c BlockConn, err os.Error)
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

// NewHeaderOverBlockConn creates a HeaderConn on top of a BlockConn
func NewHeaderOverBlockConn(bc BlockConn) HeaderConn {
	return &headerOverBlock{ bc: bc }
}

type headerOverBlock struct {
	bc   BlockConn
}

func (hob *headerOverBlock) GetMTU() int { return hob.bc.GetMTU() }

// Since a BlockConn already has the notion of a flow, both ReadHeader
// and WriteHeader pass zero labels for the Source and Dest IPs
// to the DCCP header's read and write functions.

func (hob *headerOverBlock) ReadHeader() (h *Header, err os.Error) {
	p, err := hob.bc.ReadBlock()
	if err != nil {
		return nil, err
	}
	return ReadHeader(p, labelZero.Bytes(), labelZero.Bytes(), AnyProto, false)
}

func (hob *headerOverBlock) WriteHeader(h *Header) (err os.Error) {
	p, err := h.Write(labelZero.Bytes(), labelZero.Bytes(), AnyProto, false)
	if err != nil {
		return err
	}
	return hob.bc.WriteBlock(p)
}

func (hob *headerOverBlock) LocalLabel() Bytes { return hob.bc.LocalLabel() }

func (hob *headerOverBlock) RemoteLabel() Bytes { return hob.bc.RemoteLabel() }

func (hob *headerOverBlock) SetReadTimeout(nsec int64) os.Error { return hob.bc.SetReadTimeout(nsec) }

func (hob *headerOverBlock) Close() os.Error { return hob.bc.Close() }
