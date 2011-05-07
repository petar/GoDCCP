// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// If socket is in LISTEN, 
// Implements Step 3, Section 8.5
func (c *Conn) processLISTEN(h *Header) os.Error {
	if h.Type == Reset {
		return ErrDrop
	}
	if h.Type != Request {
		return c.inject(c.newAbnormalReset(h))
	}
	c.slk.Lock()
	c.socket.SetState(RESPOND)
	iss := c.socket.ChooseISS()
	c.socket.SetGAR(iss)

	c.socket.SetISR(h.SeqNo)
	c.socket.SetGSR(h.SeqNo)

	c.slk.Unlock()
	return c.processRESPOND(h)
}
