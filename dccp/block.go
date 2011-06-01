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
	// byte size of the header's wire-format footprint.
	GetMTU() int

	ReadHeader() (h *Header, err os.Error)

	// WriteHeader can return ErrTooBig, if the wire-format of h exceeds the MTU
	WriteHeader(h *Header) (err os.Error)

	Close() os.Error
	SetReadTimeout(nsec int64) os.Error
}
