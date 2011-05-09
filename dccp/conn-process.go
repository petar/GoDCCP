// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"log"
	"os"
	"time"
)

// newAbnormalReset() generates a new out-of-sync Reset header, according to Section 8.3.1
func (c *Conn) newAbnormalReset(resetCode uint32, h *Header) *Header {
	return c.TakeAbnormalSeqAck(NewResetHeader(resetCode, c.id.SourcePort, c.id.DestPort), h)
}

// newReset() generates a new Reset header
func (c *Conn) newReset(resetCode uint32) *Header {
	return c.TakeSeqAck(NewResetHeader(resetCode, c.id.SourcePort, c.id.DestPort))
}

// newSync() generates a new Sync header
func (c *Conn) newSync() *Header { 
	return c.TakeSeqAck(NewSyncHeader(c.id.SourcePort, c.id.DestPort))
}

// If socket is in TIMEWAIT, it must perform a Reset sequence.
// Implements the second half of Step 2, Section 8.5
func (c *Conn) processTIMEWAIT(h *Header) os.Error {
	if h.Type == Reset {
		return nil
	}
	return c.inject(c.newAbnormalReset(ResetNoConnection, h))
}

// If socket is in LISTEN, 
// Implements Step 3, Section 8.5
func (c *Conn) processLISTEN(h *Header) os.Error {
	if h.Type == Reset {
		return ErrDrop
	}
	if h.Type != Request {
		return c.inject(c.newAbnormalReset(ResetNoConnection, h))
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

// If socket is in REQUEST
// Implements Step 4, Section 8.5
func (c *Conn) processREQUEST(h *Header) os.Error {
	c.slk.Lock()
	inAckWindow := c.socket.InAckWindow(h.AckNo)
	c.slk.Unlock()
	if (h.Type == Response || h.Type == Reset) && inAckWindow {
		c.slk.Lock()
		c.socket.SetISR(h.SeqNo)
		c.slk.Unlock()
		c.PlaceSeqAck(h)

		switch h.Type {
		case Response:
			return c.processREQUEST2(h)
		case Reset:
			return c.processReset(h)
		}
		panic("unreach")
	}
	return c.inject(c.newReset(ResetPacketError))
}

// Step 5, Section 8.5: Prepare sequence numbers for Sync
func (c *Conn) step5_PrepSeqNoForSync(h *Header) os.Error {
	if h.Type != Sync && h.Type != SyncAck {
		return ErrDrop
	}

	c.slk.Lock()
	defer c.slk.Unlock()

	swl, _ := c.socket.GetSWLH()
	if c.socket.InAckWindow(h.AckNo) && h.SeqNo >= swl {
		c.socket.UpdateGSR(h.SeqNo)
		return nil
	}
	return ErrDrop
}

// Step 6, Section 8.5: Check sequence numbers
func (c *Conn) step6_CheckSeqNo(h *Header) os.Error {
	// We don't support short sequence numbers
	if !h.X {
		return ErrDrop
	}

	c.slk.Lock()
	defer c.slk.Unlock()

	swl, swh := c.socket.GetSWLH()
	awl, awh := c.socket.GetAWLH()
	lswl, lawl := swl, awl

	gsr := c.socket.GetGSR()
	gar := c.socket.GetGAR()
	if h.Type == CloseReq || h.Type == Close || h.Type == Reset {
		lswl, lawl := gsr + 1, gar
	}

	hasAckNo := h.HasAckNo()
	if (lswl <= h.SeqNo && h.SeqNo <= swh) && (!hasAckNo || (lawl <= h.AckNo && h.AckNo <= awh)) {
		c.socket.UpdateGSR(h.SeqNo)
		if h.Type != Sync {
			if hasAckNo {
				c.socket.UpdateGAR(h.AckNo)
			} else {
				log.Printf("Step6: expecting ack no\n")
			}
		}
		return nil
	} else {
		var g *Header = c.newSync()
		if h.Type == Reset {
			// Send Sync packet acknowledging S.GSR
		} else {
			// Send Sync packet acknowledging P.seqno
			g.AckNo = h.SeqNo	
		}
		defer c.inject(g) // Use defer to perform this after the 'defer c.slk.Unlock()'
		return ErrDrop
	}
	panic("unreach")
}

// Process Reset headers
// Implements Step 9, Section 8.5
func (c *Conn) processReset(h *Header) os.Error {
	panic("¿i?")

	// XXX c.teardown()

	c.slk.Lock()
	c.socket.SetState(TIMEWAIT)
	c.slk.Unlock()

	go func() {
		time.Sleep(2*MSL)
		c.kill()
	}()
	return nil
}

func (c *Conn) kill() {
	panic("¿i?")
}

// Implements Step 10, Section 8.5
func (c *Conn) processREQUEST2(h *Header) os.Error {

	// Move to PARTOPEN state
	c.slk.Lock()
	c.socket.SetState(PARTOPEN)
	c.slk.Unlock()
	
	// PARTOPEN means send an Ack, don't send Data packets, retransmit Acks
	// periodically, and always include any Init Cookie from the Response 
	// (Init Cookies are not supported yet.)

	// Start PARTOPEN timer, according to Section 8.1.5
	defer go func() {
		// The preferred mechanism would be a roughly 200-millisecond timer, set
		// every time a packet is transmitted in PARTOPEN.  If this timer goes
		// off and the client is still in PARTOPEN, the client generates another
		// DCCP-Ack and backs off the timer.  If the client remains in PARTOPEN
		// for more than 4MSL (8 minutes), it SHOULD reset the connection with
		// Reset Code 2, "Aborted".
		b := newBackOff(PARTOPEN_BACKOFF_FIRST, PARTOPEN_BACKOFF_MAX)
		for {
			e := b.Sleep()
			c.slk.Lock()
			state := c.socket.GetState()
			c.slk.Unlock()
			if state != PARTOPEN {
				break
			}
			if e != nil {
				c.abort()
				break
			}
			c.inject(c.newAck())
		}
	}()

	return c.processPARTOPEN(h)
}

// abort() resets the connection with Reset Code 2, "Aborted"
func (c *Conn) abort() os.Error {
	?
}

// If socket is in RESPOND, 
// Implements Step 11, Section 8.5
func (c *Conn) processRESPOND(h *Header) os.Error {
	panic("¿i?")
}

func (c *Conn) newAck() *Header {
	return c.TakeSeqAck(NewAckHeader(c.id.SourcePort, c.id.DestPort))
}

// If socket is in PARTOPEN
// Implements Step 12, Section 8.5
func (c *Conn) processPARTOPEN(h *Header) os.Error {
	if h.Type == Response {
		c.inject(c.newAck())
		return nil
	}
	// Otherwise,
	// The client leaves the PARTOPEN state for OPEN when it receives a
	// valid packet other than DCCP-Response, DCCP-Reset, or DCCP-Sync from
	// the server.
	if h.Type != Response && h.Type != Reset && h.Type != Sync {
		c.slk.Lock()
		c.socket.SetOSR(h.SeqNo)
		c.socket.SetState(OPEN)
		c.slk.Unlock()
		return nil
	}
	log.Printf("processPARTOPEN, unexpected packet type: %d\n", h.Type)
	return ErrDrop
}
