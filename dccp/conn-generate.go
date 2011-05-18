// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// generateAbnormalReset() generates a new out-of-sync Reset header, according to Section 8.3.1
func (c *Conn) generateAbnormalReset(resetCode uint32, h *Header) *Header {
	return c.TakeAbnormalSeqAck(NewResetHeader(resetCode, c.id.SourcePort, c.id.DestPort), h)
}

// generateReset() generates a new Reset header
func (c *Conn) generateReset(resetCode uint32) *Header {
	return c.TakeSeqAck(NewResetHeader(resetCode, c.id.SourcePort, c.id.DestPort))
}

// generateSync() generates a new Sync header
func (c *Conn) generateSync() *Header { 
	return c.TakeSeqAck(NewSyncHeader(c.id.SourcePort, c.id.DestPort))
}

// generateSyncAck() generates a new SyncAck header
func (c *Conn) generateSync(inResponseTo *Header) *Header { 
	g := c.TakeSeqAck(NewSyncHeader(c.id.SourcePort, c.id.DestPort))
	if inResponseTo.Type != Sync {
		panic("SyncAck without a Sync")
	}
	g.AckNo = h.SeqNo
	return g
}

// generateResponse() generates a new Response header
func (c *Conn) generateResponse(serviceCode uint32) *Header { 
	return c.TakeSeqAck(NewResponseHeader(serviceCode, c.id.SourcePort, c.id.DestPort))
}

func (c *Conn) generateAck() *Header {
	return c.TakeSeqAck(NewAckHeader(c.id.SourcePort, c.id.DestPort))
}

func (c *Conn) generateClose() *Header {
	return c.TakeSeqAck(NewCloseHeader(c.id.SourcePort, c.id.DestPort))
}
