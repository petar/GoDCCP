// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// generateAbnormalReset() generates a new out-of-sync Reset header, according to Section 8.3.1
func (c *Conn) generateAbnormalReset(resetCode byte, inResponseTo *Header) *Header {
	return c.TakeAbnormalSeqAck(NewResetHeader(resetCode), inResponseTo)
}

func (c *Conn) generateReset(resetCode byte) *Header {
	return c.TakeSeqAck(NewResetHeader(resetCode))
}

func (c *Conn) generateSync() *Header {
	return c.TakeSeqAck(NewSyncHeader())
}

func (c *Conn) generateSyncAck(inResponseTo *Header) *Header {
	g := c.TakeSeqAck(NewSyncAckHeader())
	if inResponseTo.Type != Sync {
		panic("SyncAck without a Sync")
	}
	g.AckNo = inResponseTo.SeqNo
	return g
}

func (c *Conn) generateRequest(serviceCode uint32) *Header {
	return c.TakeSeqAck(NewRequestHeader(serviceCode))
}

func (c *Conn) generateResponse(serviceCode uint32) *Header {
	return c.TakeSeqAck(NewResponseHeader(serviceCode))
}

func (c *Conn) generateClose() *Header {
	return c.TakeSeqAck(NewCloseHeader())
}

func (c *Conn) generateAck() *Header {
	return c.TakeSeqAck(NewAckHeader())
}

func (c *Conn) generateData(data []byte) *Header {
	return c.TakeSeqAck(NewDataHeader(data))
}

func (c *Conn) generateDataAck(data []byte) *Header {
	return c.TakeSeqAck(NewDataAckHeader(data))
}
