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
	MaxBlockLen() int	// The largest allowable block size (for read and write)
	ReadBlock() (block []byte, err os.Error)
	WriteBlock(block []byte) (err os.Error)
	Close() os.Error
}

type HeaderConn interface {
	ReadHeader() (h *Header, err os.Error)
	WriteHeader(h *Header) (err os.Error)
	Close() os.Error
}
