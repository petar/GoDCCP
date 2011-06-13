// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// This is an approximate upper bound on the size of options that are
// allowed on a Data or DataAck packet. See isOptionValidForType.
const maxDataOptionSize = 24

// GetMTU() returns the maximum size of an application-level data block that can be passed
// to WriteBlock This is an informative number. Packets are sent anyway, but they may be
// dropped by the link layer or a router.
func (c *Conn) GetMTU() int {
	c.Lock()
	defer c.Unlock()
	c.updateSocketLink()
	return int(c.socket.GetMPS()) - maxDataOptionSize - getFixedHeaderSize(DataAck, true)
}

// WriteBlock blocks until the slice b is sent.
func (c *Conn) WriteBlock(b []byte) os.Error {
	c.writeDataLk.Lock()
	defer c.writeDataLk.Unlock()
	if c.writeData == nil {
		return os.EBADF
	}
	c.writeData <- b
	return nil
}

// ReadBlock blocks until the next packet of application data is received.
// It returns a non-nil error only if the connection has been closed.
func (c *Conn) ReadBlock() (b []byte, err os.Error) {
	b, ok := <-c.readApp
	if !ok {
		// The connection has been closed
		return nil, os.EBADF
	}
	return b, nil
}

// Close closes the connection, Section 8.3
func (c *Conn) Close() os.Error {
	c.Lock()
	state := c.socket.GetState()
	c.Unlock()
	if state == CLOSED || state == CLOSEREQ || state == CLOSING || state == TIMEWAIT {
		return nil
	}
	if state != OPEN {
		return os.EBADF
	}
	// Transition to CLOSING
	c.Lock()
	c.inject(c.generateClose())
	c.gotoCLOSING()
	c.Unlock()
	return nil
}

func (c *Conn) LocalLabel() Bytes { return c.hc.LocalLabel() }

func (c *Conn) RemoteLabel() Bytes { return c.hc.RemoteLabel() }

func (c *Conn) SetReadTimeout(nsec int64) os.Error {
	panic("un")
}
