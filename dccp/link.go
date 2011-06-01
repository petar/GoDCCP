// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"io"
	"net"
	"os"
)

// Link{} is an abstract interface to a physical connection-less packet layer which sends and
// receives packets
type Link interface {
	GetMTU() int	// Writes smaller than this are guaranteed to be sent whole
	ReadFrom(buf []byte) (n int, addr net.Addr, err os.Error)	// Receive next packet of data
	WriteTo(buf []byte, addr net.Addr) (n int, err os.Error)	// Send a packet of data. Partial writes allowed
	io.Closer
}
