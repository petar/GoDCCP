// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// generateAbnormalReset() generates a new out-of-sync Reset header, according to Section 8.3.1
func (c *Conn) generateAbnormalReset(resetCode byte, inResponseTo *Header) *Header {
	return c.TakeAbnormalSeqAck(NewResetHeader(resetCode, c.id.SourcePort, c.id.DestPort),
		inResponseTo)
}

func (c *Conn) generateReset(resetCode byte) *Header {
	return c.TakeSeqAck(NewResetHeader(resetCode, c.id.SourcePort, c.id.DestPort))
}

func (c *Conn) generateSync() *Header {
	return c.TakeSeqAck(NewSyncHeader(c.id.SourcePort, c.id.DestPort))
}

func (c *Conn) generateSyncAck(inResponseTo *Header) *Header {
	g := c.TakeSeqAck(NewSyncAckHeader(c.id.SourcePort, c.id.DestPort))
	if inResponseTo.Type != Sync {
		panic("SyncAck without a Sync")
	}
	g.AckNo = inResponseTo.SeqNo
	return g
}

func (c *Conn) generateResponse(serviceCode uint32) *Header {
	return c.TakeSeqAck(NewResponseHeader(serviceCode, c.id.SourcePort, c.id.DestPort))
}

func (c *Conn) generateClose() *Header {
	return c.TakeSeqAck(NewCloseHeader(c.id.SourcePort, c.id.DestPort))
}

func (c *Conn) generateAck() *Header {
	return c.TakeSeqAck(NewAckHeader(c.id.SourcePort, c.id.DestPort))
}

func (c *Conn) generateData(data []byte) *Header {
	return c.TakeSeqAck(NewDataHeader(data, c.id.SourcePort, c.id.DestPort))
}

func (c *Conn) generateDataAck(data []byte) *Header {
	return c.TakeSeqAck(NewDataAckHeader(data, c.id.SourcePort, c.id.DestPort))
}
