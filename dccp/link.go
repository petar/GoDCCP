// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "net"

// Link is an abstract interface to a physical connection-less packet layer which sends and
// receives packets
type Link interface {

	// Returns the Maximum Transmission Unit
	// Writes smaller than this are guaranteed to be sent whole
	GetMTU() int

	// ReadFrom receives the next packet of data
	ReadFrom(buf []byte) (n int, addr net.Addr, err error)

	// WriteTo sends a packet of data
	WriteTo(buf []byte, addr net.Addr) (n int, err error)

	// SetReadTimeout has the same meaning as net.Conn.SetReadTimeout
	SetReadTimeout(nsec int64) error

	// Close terminates the link gracefully
	Close() error
}
