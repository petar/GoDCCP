// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// inject() adds the packet h to the outgoing non-Data pipeline, without blocking.
// The pipeline is flushed continuously respecting the CongestionControl's
// rate-limiting policy.
// REMARK: inject() is called at most once (currently) from inside readLoop() and inside a
// lock on Conn, so it must not block, hence writeNonData has buffer space
func (c *Conn) inject(h *Header) {
	if h == nil {
		panic("injecting nil header")
	}
	c.logWriteHeaderLocked(h)
	c.writeNonData <- h
}

// writeLoop() sends headers incoming on the writeData and writeNonData channels, while
// giving priority to writeNonData. It continues to do so until writeNonData is closed.
func (c *Conn) writeLoop() {
	writeNonData := c.writeNonData
	writeData := c.writeData
Loop:
	for {
		if c.cc.Strobe() != nil {
			panic("broken congestion control")
		}
		var h *Header
		var ok bool
		var appData []byte
		select {
		case h, ok = <-writeNonData:
			if !ok {
				// Closing writeNonData means that the Conn is done and dead
				break Loop
			}
		case appData, ok = <-writeData:
			if !ok {
				// Make a dummy channel that no-one sends to, so that the
				// select (above) would block until something comes on writeNonData.
				writeData = make(chan []byte)
			} else {
				c.Lock()
				state := c.socket.GetState()
				c.Unlock()
				// Having been in OPEN guarantees that AckNo can be filled in
				// meaningfully (below) in the DataAck packet
				if state != OPEN {
					appData = nil
				}
			}
			if len(appData) > 0 {
				c.Lock()
				h = c.generateDataAck(appData)
				c.Unlock()
			}
		}
		if h == nil {
			continue
		}
		err := c.hc.WriteHeader(h)
		if err != nil {
			c.abortQuietly()
			break Loop
		}
	}
	// Close the congestion control here when it won't be needed any longer
	c.cc.Close()
}
