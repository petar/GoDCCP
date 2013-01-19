// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// Special case seq and ack numbers

// generateAbnormalReset() generates a new out-of-sync Reset header, according to Section 8.3.1
func (c *Conn) generateAbnormalReset(resetCode byte, inResponseTo *Header) *writeHeader {
	h := &writeHeader{}
	h.Header.InitResetHeader(resetCode)
	h.SeqAckType = seqAckAbnormal
	h.InResponseTo = inResponseTo
	return h
}

func (c *Conn) generateSyncAck(inResponseTo *Header) *writeHeader {
	h := &writeHeader{}
	h.Header.InitSyncAckHeader()
	h.SeqAckType = seqAckSyncAck
	h.InResponseTo = inResponseTo
	return h
}

// Common case seq and ack numbers

func (c *Conn) generateReset(resetCode byte) *writeHeader {
	h := &writeHeader{}
	h.Header.InitResetHeader(resetCode)
	h.SeqAckType = seqAckNormal
	return h
}

func (c *Conn) generateSync() *writeHeader {
	h := &writeHeader{}
	h.Header.InitSyncHeader()
	h.SeqAckType = seqAckNormal
	return h
}

func (c *Conn) generateRequest(serviceCode uint32) *writeHeader {
	h := &writeHeader{}
	h.Header.InitRequestHeader(serviceCode)
	h.SeqAckType = seqAckNormal
	return h
}

func (c *Conn) generateResponse(serviceCode uint32) *writeHeader {
	h := &writeHeader{}
	h.Header.InitResponseHeader(serviceCode)
	h.SeqAckType = seqAckNormal
	return h
}

func (c *Conn) generateClose() *writeHeader {
	h := &writeHeader{}
	h.Header.InitCloseHeader()
	h.SeqAckType = seqAckNormal
	return h
}

func (c *Conn) generateAck() *writeHeader {
	h := &writeHeader{}
	h.Header.InitAckHeader()
	h.SeqAckType = seqAckNormal
	return h
}

func (c *Conn) generateData(data []byte) *writeHeader {
	h := &writeHeader{}
	h.Header.InitDataHeader(data)
	h.SeqAckType = seqAckNormal
	return h
}

func (c *Conn) generateDataAck(data []byte) *writeHeader {
	h := &writeHeader{}
	h.Header.InitDataAckHeader(data)
	h.SeqAckType = seqAckNormal
	return h
}
