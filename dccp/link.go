// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"time"
)

// Note that Link objects return Go package (net, os, and syscall) errors, whereas
// SegmentConn and HeaderConn (being internal to DCCP) return DCCP errors.
// The driver type that implements access to a concrete communication link would be what
// bridges from Go errors to DCCP errors. E.g. this happans in type flow.

// Link is an abstract interface to a physical connection-less packet layer which sends and
// receives packets
type Link interface {

	// Returns the Maximum Transmission Unit
	// Writes smaller than this are guaranteed to be sent whole
	GetMTU() int

	// ReadFrom receives the next packet of data. It returns net.Error errors. So in the event
	// of a timeout e.g. it will return a net.Error with Timeout() true, rather than an
	// ErrTimeout. The latter is used internally to DCCP in types that implement HeaderConn and
	// SegmentConn.
	ReadFrom(buf []byte) (n int, addr net.Addr, err error)

	// WriteTo sends a packet of data
	WriteTo(buf []byte, addr net.Addr) (n int, err error)

	// SetReadDeadline has the same meaning as net.Conn.SetReadDeadline
	SetReadDeadline(t time.Time) error

	// Close terminates the link gracefully
	Close() error
}
