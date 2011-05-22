// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// inject() adds the packet h to the outgoing non-Data pipeline, without blocking.
// The pipeline is flushed continuously respecting the CongestionControl's
// rate-limiting policy.
// REMARK: inject() is called at most once (currently) from inside readLoop() and inside a
// lock on Conn, so it must not block, hence writeNonData has buffer space
func (c *Conn) inject(h *Header) {
	if h == nil {
		panic("injecting nil header")
	}
	c.writeNonData <- h
}

// writeLoop() sends headers incoming on the writeData and writeNonData channels, while
// giving priority to writeNonData. It continues to do so until writeNonData is closed.
func (c *Conn) writeLoop() {
	writeNonData := c.writeNonData
	writeData := c.writeData
	for {
		c.Strobe()
		var h *Header
		var ok bool
		select {
		case h, ok = <-writeNonData:
			if !ok {
				break
			}
		case h, ok = <-writeData:
			if !ok {
				writeData = make(chan *Header)
			} else {
				c.Lock()
				state := c.socket.GetState()
				c.Unlock()
				if state != OPEN {
					h = nil
				}
			}
		}
		if h == nil {
			continue
		}
		err := c.headerConn.WriteHeader(h)
		if err != nil {
			c.kill()
			break
		}
	}
}
