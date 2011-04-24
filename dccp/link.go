// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"io"
	"os"
)

// Link{} is an abstract interface to a physical connection-less packet layer which sends and
// receives packets
type Link interface {
	Read() (buf []byte, ip IP, err os.Error)	// Receive next packet of data
	Write(buf []byte, ip IP) os.Error		// Send a packet of data
	io.Closer
}
