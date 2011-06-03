// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// A BlockConn is an I/O facility that explicitly reads/writes data in the form
// of indivisible blocks of data. 
type BlockConn interface {
	// GetMTU returns th he largest allowable block size (for read and write). The MTU may vary.
	GetMTU() int

	ReadBlock() (block []byte, err os.Error)

	// If the user attempts to write a block that is too big, an ErrTooBig is returned
	// and the block is not sent.
	WriteBlock(block []byte) (err os.Error)

	Close() os.Error
	SetReadTimeout(nsec int64) os.Error
}

type HeaderConn interface {
	// GetMTU() returns the Maximum Transmission Unit size. This is the maximum
	// byte size of the header and app data wire-format footprint.
	GetMTU() int

	// os.EAGAIN is returned in the event of timeout.
	ReadHeader() (h *Header, err os.Error)

	// WriteHeader can return ErrTooBig, if the wire-format of h exceeds the MTU
	WriteHeader(h *Header) (err os.Error)

	Close() os.Error
	SetReadTimeout(nsec int64) os.Error
}

// NewHeaderOverBlockConn creates a HeaderConn on top of a BlockConn
func NewHeaderOverBlockConn(bc BlockConn, localIP, remoteIP []byte) HeaderConn {
	return &headerOverBlock{
		localIP:  localIP,
		remoteIP: remoteIP,
		bc:       bc,
	}
}

type headerOverBlock struct {
	localIP, remoteIP []byte
	bc BlockConn
}


func (hob *headerOverBlock) GetMTU() int { return hob.bc.GetMTU() }

func (hob *headerOverBlock) ReadHeader() (h *Header, err os.Error) {
	p, err := hob.bc.ReadBlock()
	if err != nil {
		return nil, err
	}
	return ReadHeader(p, hob.remoteIP, hob.localIP, AnyProto, false)
}

func (hob *headerOverBlock) WriteHeader(h *Header) (err os.Error) {
	p, err := h.Write(hob.localIP, hob.remoteIP, AnyProto, false)
	if err != nil {
		return err
	}
	return hob.bc.WriteBlock(p)
}

func (hob *headerOverBlock) Close() os.Error { return hob.bc.Close() }

func (hob *headerOverBlock) SetReadTimeout(nsec int64) os.Error { return hob.bc.SetReadTimeout(nsec) }
