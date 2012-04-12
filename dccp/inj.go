// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

func (c *Conn) writeCCID(h *Header) *Header {
	now := c.run.Nanoseconds()
	// HC-Sender CCID
	ccval, sropts := c.scc.OnWrite(&PreHeader{Type: h.Type, X: h.X, SeqNo: h.SeqNo, AckNo: h.AckNo, Time: now})
	if !validateCCIDSenderToReceiver(sropts) {
		panic("sender congestion control writes disallowed options")
	}
	h.CCVal = ccval
	// HC-Receiver CCID
	rsopts := c.rcc.OnWrite(&PreHeader{Type: h.Type, X: h.X, SeqNo: h.SeqNo, AckNo: h.AckNo, Time: now})
	if !validateCCIDReceiverToSender(rsopts) {
		panic("receiver congestion control writes disallowed options")
	}
	// TODO: Also check option compatibility with respect to packet type (Data vs. other)
	h.Options = append(h.Options, append(sropts, rsopts...)...)
	return h
}

// inject adds the packet h to the outgoing non-Data pipeline, without blocking.  The
// pipeline is flushed continuously respecting the CongestionControl's rate-limiting policy.
//
// inject is called at most once (currently) from inside readLoop and inside a lock
// on Conn, so it must not block, hence writeNonData has buffer space
func (c *Conn) inject(h *Header) {
	c.writeNonDataLk.Lock()
	defer c.writeNonDataLk.Unlock()

	if c.writeNonData == nil {
		return
	}

	// Catch outgoing non-Data packets for debug purposes here
	// c.emitCatchSeqNo(h, 161019, 161020, 161021)

	// Dropping a nil is OK, since it happens only if there are other packets in the queue
	if len(c.writeNonData) < cap(c.writeNonData) {
		c.writeNonData <- h
	} else {
		c.amb.E(EventDrop, "Slow strobe", h)
	}
}

func (c *Conn) write(h *Header) error {
	c.scc.Strobe()
	return c.hc.WriteHeader(h)
}

// writeLoop() sends headers incoming on the writeData and writeNonData channels, while
// giving priority to writeNonData. It continues to do so until writeNonData is closed.
func (c *Conn) writeLoop(writeNonData chan *Header, writeData chan []byte) {

	// The presence of multiple loops below allows user calls to WriteSegment to
	// block in "writeNonData <-" while the connection moves into a state where
	// it accepts app data (in _Loop_II)

	// This loop is active until state OPEN or PARTOPEN is observed, when a
	// transition to _Loop II_is made
	c.amb.E(EventInfo, "Write Loop I")
_Loop_I:

	for {
		h, ok := <-writeNonData
		if !ok {
			// Closing writeNonData means that the Conn is done and dead
			goto _Exit
		}
		// We'll allow nil headers, since they can be used to trigger unblock
		// from the above send operator and (without resulting into an actual
		// send) activate the state check after the "if" statement below
		if h != nil {
			// Tell the CCID about h right before it gets sent.
			// This way, the roundtrip measurements e.g. which are done inside CCID
			// will not be affected by the wait time incurred due to send rate considerations
			c.Lock()
			h = c.writeCCID(h)
			c.Unlock()
			c.amb.E(EventWrite, "Non-data to injection queue", h)
			err := c.write(h)
			// If the underlying layer is broken, abort
			if err != nil {
				c.abortQuietly()
				goto _Exit
			}
		}
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		switch state {
		case OPEN, PARTOPEN:
			goto _Loop_II
		}
		continue _Loop_I
	}

	// This loop is active until writeData is not closed
	c.amb.E(EventInfo, "Write Loop II")
_Loop_II:

	for {
		var h *Header
		var ok bool
		var appData []byte
		select {
		// Note that non-Data packets take precedence
		case h, ok = <-writeNonData:
			if !ok {
				// Closing writeNonData means that the Conn is done and dead
				goto _Exit
			}
		case appData, ok = <-writeData:
			if !ok {
				// When writeData is closed, we transition to the 3rd loop,
				// which accepts only non-Data packets
				goto _Loop_III
			}
			// By virtue of being in _Loop_II (which implies we have been or are in OPEN
			// or PARTOPEN), we know that some packets of the other side have been
			// received, and so AckNo can be filled in meaningfully (below) in the
			// DataAck packet

			// We allow 0-length app data packets. No reason not to.
			// XXX: I am not sure if Header.Data == nil (rather than
			// Header.Data = []byte{}) would cause a problem in Header.Write
			// It should be that it doesn't. Must verify this.
			c.Lock()
			h = c.generateDataAck(appData)
			h = c.writeCCID(h)
			c.Unlock()
			c.amb.E(EventWrite, "Data", h)
		}
		if h != nil {
			err := c.write(h)
			if err != nil {
				c.abortQuietly()
				goto _Exit
			}
		}
	}

	// This loop is active until writeNonData is not closed
	c.amb.E(EventInfo, "Write Loop III")
_Loop_III:

	for {
		h, ok := <-writeNonData
		if !ok {
			// Closing writeNonData means that the Conn is done and dead
			goto _Exit
		}
		// We'll allow nil headers, since they can be used to trigger unblock
		// from the above send operator
		if h != nil {
			c.Lock()
			h = c.writeCCID(h)
			c.Unlock()
			c.amb.E(EventWrite, "Non-data to injection queue", h)
			err := c.write(h)
			// If the underlying layer is broken, abort
			if err != nil {
				c.abortQuietly()
				goto _Exit
			}
		}
	}

_Exit:
	c.amb.E(EventInfo, "Write loop EXIT")
}
