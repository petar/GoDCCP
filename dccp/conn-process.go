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

	return c.step11_ProcessRESPOND(h)
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
			return c.step10_ProcessREQUEST2(h)
		case Reset:
			return c.step9_ProcessReset(h)
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
		c.inject(g)
		return ErrDrop
	}
	panic("unreach")
}

// Step 7, Section 8.5: Check for unexpected packet types
func (c *Conn) step7_CheckUnexpectedTypes(h *Header) os.Error {
	c.slk.Lock()
	isServer := c.socket.IsServer()
	isClient := !isServer
	state := c.socket.GetState()
	osr := c.socket.GetOSR()
	c.slk.Unlock()
	if (isServer && h.Type == CloseReq)
		|| (isServer && h.Type == Response)
		|| (isClient && h.Type == Request)
		|| (state >= OPEN && h.Type == Request && h.SeqNo >= osr)
		|| (state >= OPEN && h.Type == Response && h.SeqNo >= osr)
		|| (state == RESPOND && h.Type == Data) {
			g := c.newSync()
			g.AckNo = h.SeqNo	
			c.inject(g)
			return ErrDrop
		}
	return nil
}

// Step 8, Section 8.5: Process options and mark acknowledgeable
func (c *Conn) step8_OptionsAndMarkAckbl(h *Header) os.Error {
	// We don't support any options yet

	// Mark packet as acknowledgeable (in Ack Vector terms, Received or Received ECN Marked)
	XX Not implemented yet
}

// Step 9, Section 8.5: Process Reset
func (c *Conn) step9_ProcessReset(h *Header) os.Error {
	
	X?X c.teardown()
	panic("¿i?")

	c.slk.Lock()
	c.socket.SetState(TIMEWAIT)
	c.slk.Unlock()

	go func() {
		time.Sleep(2*MSL)
		c.kill()
	}()
	return ErrDrop
}

func (c *Conn) kill() {
	panic("¿i?")
}

// Step 10, Section 8.5: Process REQUEST state (second part)
func (c *Conn) step10_ProcessREQUEST2(h *Header) os.Error {

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

	return c.step12_ProcessPARTOPEN(h)
}

// abort() resets the connection with Reset Code 2, "Aborted"
func (c *Conn) abort() os.Error {
	?
}

// newResponse() generates a new Response header
func (c *Conn) newResponse(serviceCode uint32) *Header { 
	return c.TakeSeqAck(NewResponseHeader(serviceCode, c.id.SourcePort, c.id.DestPort))
}

// Step 11, Section 8.5: Process RESPOND state
func (c *Conn) step11_ProcessRESPOND(h *Header) os.Error {
	c.slk.Lock()
	defer c.slk.Unlock()

	if c.socket.GetState() == RESPOND {
		if h.Type == Request {
			// Send Response, possibly containing Init Cookie
			// (But we don't support Init Cookies yet.)
			if c.socket.GetGSR() != h.SeqNo {
				log.Panic("DCCP RESPOND: GSR != h.SeqNo\n")
			}
			XXX should we also be saving this ServiceCode to the socket state ??
			c.inject(c.newResponse(h.ServiceCode))
			// If Init Cookie was sent,
			//    Destroy S and return
			// (Again, Init Cookies not supported.)
		} else {
			c.socket.SetOSR(h.SeqNo)
			c.socket.SetState(OPEN)
		}
	}
	return nil
}

func (c *Conn) newAck() *Header {
	return c.TakeSeqAck(NewAckHeader(c.id.SourcePort, c.id.DestPort))
}

// Step 12, Section 8.5: Process PARTOPEN state
func (c *Conn) step12_ProcessPARTOPEN(h *Header) os.Error {
	c.slk.Lock()
	defer c.slk.Unlock()
	if c.socket.GetState() != PARTOPEN {
		return nil
	}

	if h.Type == Response {
		c.inject(c.newAck())
		return nil
	}
	// Otherwise,
	// (Section 8.1.5) The client leaves the PARTOPEN state for OPEN when it
	// receives a valid packet other than DCCP-Response, DCCP-Reset, or
	// DCCP-Sync from the server.
	if h.Type != Response && h.Type != Reset && h.Type != Sync {
		c.slk.Lock()
		c.socket.SetOSR(h.SeqNo)
		c.socket.SetState(OPEN)
		c.slk.Unlock()
		return nil
	}
	log.Printf("Step12: unexpected packet type: %d\n", h.Type)
	return ErrDrop
}

// Step 13, Section 8.5: Process CloseReq
func (c *Conn) step13_ProcessCloseReq(h *Header) os.Error {
	c.slk.Lock()
	defer c.slk.Unlock()

	if h.Type == CloseReq && c.socket.GetState() < CLOSEREQ {
		// Generate Close
		??
		c.socket.SetState(CLOSING)
		// Set CLOSING timer
		??
	}

	return nil
}

// Step 14, Section 8.5: Process Close
func (c *Conn) step14_ProcessClose(h *Header) os.Error {
	if h.Type == Close {
		// Generate Reset(Closed)
		??
		// Tear down connection
		??
		return ErrDrop
	}
	return nil
}

// Step 15, Section 8.5: Process Sync
func (c *Conn) step15_ProcessSync(h *Header) os.Error {
	if h.Type == Sync {
		// Generate SyncAck
		??
	}
	return nil
}

// Step 16, Section 8.5: Process Data
func (c *Conn) step16_ProcessData(h *Header) os.Error {
	// At this point any application data on P can be passed to the
	// application, except that the application MUST NOT receive data from
	// more than one Request or Response
	??
}
