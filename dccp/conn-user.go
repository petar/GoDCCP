// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

func (c *Conn) WriteBlock(b []byte) os.Error {
	XXX // prepare header
	c.writeChan <- pkt
	c.Lock()
	defer c.Unlock()
	if c.socket.GetState() == OPEN {
		return nil
	}
	return os.EBADF
}

// ReadBlock blocks until the next packet of application data is received.
// It returns a non-nil error only if the connection has been closed.
func (c *Conn) ReadBlock() (b []byte, err os.Error) {
	b, ok := <-c.readChan
	if !ok {
		// The connection has been closed
		return nil, os.EBADF
	}
	return b, nil
}

// Close closes the connection, Section 8.3
func (c *Conn) Close() os.Error {
	c.Lock()
	defer c.Unlock()
	// Check if connection already closed
	state := c.socket.GetState()
	if state == CLOSEREQ || state == CLOSING || state == TIMEWAIT {
		return nil
	}
	if state != OPEN {
		return os.EBADF
	}
	// Transition to CLOSING
	c.teardown()
	c.inject(c.generateClose())
	c.gotoCLOSING()
	return nil
}
