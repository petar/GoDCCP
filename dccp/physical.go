// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"io"
	"os"
)

// Physical{} is an abstract interface to a physical connection-less packet layer which sends and
// receives packets
type Physical interface {
	Read() (buf []byte, addr PhysicalAddr, err os.Error)	// Receive next packet of data
	Write(buf []byte, addr PhysicalAddr) os.Error		// Send a packet of data
	io.Closer
}

// PhysicalAddr{} is a general-purpose address type
type PhysicalAddr []byte

var ZeroPhysicalAddr = PhysicalAddr{}
