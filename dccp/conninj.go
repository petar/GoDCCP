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
	c.Lock()
	writeNonData := c.writeNonData
	writeData := c.writeData
	c.Unlock()

	// The presence of multiple loops below allows user calls to WriteBlock to
	// block in "writeNonData <-" while the connection moves into a state where
	// it accepts app data (in Loop_II)

	// This loop is active while in states prior to PARTOPEN (for clients) and OPEN (for servers)
	// In particular, app data is not accepted yet
	Loop_I:

	for {
		if c.cc.Strobe() != nil {
			panic("broken congestion control")
		}
		h, ok := <-writeNonData
		if !ok {
			// Closing writeNonData means that the Conn is done and dead
			goto Exit
		}
		// We'll allow nil headers, since they can be used to trigger unblock
		// from the above send operator and (without resulting into an actual
		// send) activate the state check after the "if" statement below
		if h != nil {
			err := c.hc.WriteHeader(h)
			// If the underlying layer is broken, abort
			if err != nil {
				c.abortQuietly()
				goto Exit
			}
		}
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		switch state {
		case OPEN, PARTOPEN:
			goto Loop_II
		case LISTEN, RESPOND, REQUEST:
			continue Loop_I
		case CLOSED:
			goto Exit
		default:
			goto Loop_III
		}
	}

	// This loop is active during PARTOPEN and OPEN states when app data is acceptable
	Loop_II:

	for {
		if c.cc.Strobe() != nil {
			panic("broken congestion control")
		}
		var h *Header
		var ok bool
		var appData []byte
		select {
		// Note that non-Data packets take precedence
		case h, ok = <-writeNonData:
			if !ok {
				// Closing writeNonData means that the Conn is done and dead
				goto Exit
			}
		case appData, ok = <-writeData:
			if !ok {
				// When writeData is closed, we transition to the 3rd loop,
				// which accepts only non-Data packets
				goto Loop_III
			}
			// By virtue of being in Loop_II (which implies we have been or are in OPEN
			// or PARTOPEN), we know that some packets of the other side have been
			// received, and so AckNo can be filled in meaningfully (below) in the
			// DataAck packet

			// We allow 0-length app data packets. No reason not to.
			// XXX: I am not sure if Header.Data == nil (rather than
			// Header.Data = []byte{}) would cause a problem in Header.Write
			// It should be that it doesn't. Must verify this.
			c.Lock()
			h = c.generateDataAck(appData)
			c.Unlock()
		}
		if h != nil {
			err := c.hc.WriteHeader(h)
			if err != nil {
				c.abortQuietly()
				goto Exit
			}
		}
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		switch state {
		case OPEN, PARTOPEN:
			continue Loop_II
		case LISTEN, RESPOND, REQUEST:
			panic("cannot return to OPEN or PARTOPEN")
		case CLOSED:
			goto Exit
		default:
			goto Loop_III
		}
	}

	// This loop is active in states after PARTOPEN and OPEN, when app data is not accepted
	Loop_III:

	for {
		if c.cc.Strobe() != nil {
			panic("broken congestion control")
		}
		h, ok := <-writeNonData
		if !ok {
			// Closing writeNonData means that the Conn is done and dead
			goto Exit
		}
		// We'll allow nil headers, since they can be used to trigger unblock
		// from the above send operator and (without resulting into an actual
		// send) activate the state check after the "if" statement below
		if h != nil {
			err := c.hc.WriteHeader(h)
			// If the underlying layer is broken, abort
			if err != nil {
				c.abortQuietly()
				goto Exit
			}
		}
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		switch state {
		case OPEN, PARTOPEN, LISTEN, RESPOND, REQUEST:
			panic("cannot return to prior state")
		case CLOSEREQ, CLOSING, TIMEWAIT:
			continue Loop_III
		default:
			goto Exit
		}
	}

	// Close the congestion control here when it won't be needed any longer
	Exit:
	c.cc.Close()
}
