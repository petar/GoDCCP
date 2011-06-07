// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// PlaceSeqAck() updates the socket registers upon
// receiving a header from the other side.
func (c *Conn) PlaceSeqAck(h *Header) {
	c.AssertLocked()

	// Update GSR
	gsr := c.socket.GetGSR()
	c.socket.SetGSR(max64(gsr, h.SeqNo))

	// Update GAR
	if h.HasAckNo() {
		gar := c.socket.GetGAR()
		c.socket.SetGAR(max64(gar, h.AckNo))
	}
}

func (c *Conn) TakeSeqAck(h *Header) *Header {
	c.AssertLocked()

	h.SeqNo = max64(c.socket.GetISS(), c.socket.GetGSS() + 1)
	c.socket.SetGSS(h.SeqNo)
	h.AckNo = c.socket.GetGSR()

	return h
}

func (c *Conn) TakeAbnormalSeqAck(h, inResponseTo *Header) *Header {
	c.AssertLocked()

	if inResponseTo.HasAckNo() {
		h.SeqNo = inResponseTo.AckNo + 1
	} else {
		h.SeqNo = 0
	}
	h.AckNo = inResponseTo.SeqNo
	return h
}
