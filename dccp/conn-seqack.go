// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// PlaceSeqAck() updates the socket registers upon
// receiving a header from the other side.
func (c *Conn) PlaceSeqAck(h *Header) {
	c.slk.Lock()
	defer c.slk.Unlock()

	// Update GSR
	gsr := c.socket.GetGSR()
	c.socket.SetGSR(maxu64(gsr, h.SeqNo))

	// Update GAR
	if h.HasAckNo() {
		gar := c.socket.GetGAR()
		c.socket.SetGAR(maxu64(gar, h.AckNo))
	}
}

func (c *Conn) TakeSeqAck(h *Header) *Header {
	c.slk.Lock()
	defer c.slk.Unlock()

	seqno := c.socket.GetGSS() + 1
	c.socket.SetGSS(seqno)
	ackno := c.socket.GetGSR()

	h.SeqNo = seqno
	h.AckNo = ackno

	return h
}

func (c *Conn) TakeAbnormalSeqAck(h, inResponseTo *Header) *Header {
	h.SeqNo = 0
	if inResponseTo.HasAckNo() {
		h.SeqNo = inResponseTo.AckNo+1
	}
	h.AckNo = inResponseTo.AckNo
	return h
}
