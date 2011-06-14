// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"log"
	"os"
)

// Step 2, Section 8.5: Check ports and process TIMEWAIT state
func (c *Conn) step2_ProcessTIMEWAIT(h *Header) os.Error {
	if c.socket.GetState() != TIMEWAIT {
		return nil
	}
	if h.Type != Reset {
		// In TIMEWAIT, the conn keeps responding with Reset until
		// TIMEWAIT ends as scheduled by gotoTIMEWAIT
		c.inject(c.generateAbnormalReset(ResetNoConnection, h))
	}
	return ErrDrop
}

// Step 3, Section 8.5: Process LISTEN state
func (c *Conn) step3_ProcessLISTEN(h *Header) os.Error {
	if c.socket.GetState() != LISTEN {
		return nil
	}
	if h.Type == Request {
		c.gotoRESPOND(h.ServiceCode, h.SeqNo)
		return nil
	}
	// For forward compatibility, if we receive a non-Request packet
	// we respond with with a Reset (unless the received packet was a Reset)
	// without aborting the connection.
	if h.Type != Reset {
		c.inject(c.generateAbnormalReset(ResetNoConnection, h))
	}
	return ErrDrop
}

// Step 4, Section 8.5: Prepare sequence numbers in REQUEST
func (c *Conn) step4_PrepSeqNoREQUEST(h *Header) os.Error {
	if c.socket.GetState() != REQUEST {
		return nil
	}
	inAckWindow := c.socket.InAckWindow(h.AckNo)
	if (h.Type == Response || h.Type == Reset) && inAckWindow {
		c.socket.SetISR(h.SeqNo)
		c.PlaceSeqAck(h)
		return nil
	}
	// For forward compatibility, even though the client expects only Response
	// packets in REQUEST mode, it responds to other packets with a ResetPacketError
	// and does not abort the connection.
	c.inject(c.generateReset(ResetPacketError))
	return ErrDrop
}

// Step 5, Section 8.5: Prepare sequence numbers for Sync
func (c *Conn) step5_PrepSeqNoForSync(h *Header) os.Error {
	if h.Type != Sync && h.Type != SyncAck {
		return nil
	}
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

	swl, swh := c.socket.GetSWLH()
	awl, awh := c.socket.GetAWLH()
	lswl, lawl := swl, awl

	gsr := c.socket.GetGSR()
	gar := c.socket.GetGAR()
	if h.Type == CloseReq || h.Type == Close || h.Type == Reset {
		lswl, lawl = gsr+1, gar
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
		var g *Header = c.generateSync()
		if h.Type == Reset {
			// Send Sync packet acknowledging S.GSR
			g.AckNo = gsr
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
	isServer := c.socket.IsServer()
	state := c.socket.GetState()
	osr := c.socket.GetOSR()
	if (isServer && h.Type == CloseReq) ||
		(isServer && h.Type == Response) ||
		(!isServer && h.Type == Request) ||
		(state >= OPEN && h.Type == Request && h.SeqNo >= osr) ||
		(state >= OPEN && h.Type == Response && h.SeqNo >= osr) ||
		(state == RESPOND && h.Type == Data) {
		g := c.generateSync()
		g.AckNo = h.SeqNo
		c.inject(g)
		return ErrDrop
	}
	return nil
}

// Step 8, Section 8.5: Process options and mark acknowledgeable
// Section 7.4: A received packet becomes acknowledgeable when Step 8 is reached.
func (c *Conn) step8_OptionsAndMarkAckbl(h *Header) os.Error {
	// TODO: Implement a connection reset if OnRead returns ErrReset
	// For now CongestionControl cannot advise a reset, just an ErrDrop
	defer c.syncWithCongestionControl()
	if err := c.scc.OnRead(h.Type, h.X, h.SeqNo, h.Options); err != nil {
		return ErrDrop
	}
	if err := c.rcc.OnRead(h.Type, h.X, h.SeqNo, h.CCVal, h.Options); err != nil {
		return ErrDrop
	}
	return nil
}

// Step 9, Section 8.5: Process Reset
func (c *Conn) step9_ProcessReset(h *Header) os.Error {
	if h.Type != Reset {
		return nil
	}
	c.teardownUser()
	c.gotoTIMEWAIT()
	return ErrDrop
}

// Step 10, Section 8.5: Process REQUEST state (second part)
func (c *Conn) step10_ProcessREQUEST2(h *Header) os.Error {
	if c.socket.GetState() != REQUEST {
		return nil
	}
	c.gotoPARTOPEN()

	return nil
}

// Step 11, Section 8.5: Process RESPOND state
func (c *Conn) step11_ProcessRESPOND(h *Header) os.Error {
	if c.socket.GetState() != RESPOND {
		return nil
	}
	if h.Type == Request {
		if c.socket.GetGSR() != h.SeqNo {
			log.Panic("GSR != h.SeqNo")
		}
		serviceCode := c.socket.GetServiceCode()
		if h.ServiceCode != serviceCode {
			return ErrDrop
		}
		c.inject(c.generateResponse(serviceCode))
	} else {
		if h.Type != Ack && h.Type != DataAck {
			log.Printf("entering OPEN on a non-Ack, non-DataAck packet")
		}
		c.gotoOPEN(h.SeqNo)
	}
	return nil
}

// Step 12, Section 8.5: Process PARTOPEN state
func (c *Conn) step12_ProcessPARTOPEN(h *Header) os.Error {
	if c.socket.GetState() != PARTOPEN {
		return nil
	}
	if h.Type == Response {
		c.inject(c.generateAck())
		// XXX: This is a deviation from the RFC. The Sync packet necessitates a SyncAck
		// response, which moves the client from PARTOPEN to OPEN in the lack of DataAck
		// packets sent from the server to the client.
		c.inject(c.generateSync())
		return nil
	}
	if h.Type != Response && h.Type != Reset && h.Type != Sync {
		c.gotoOPEN(h.SeqNo)
		return nil
	}
	return nil
}

// Step 13, Section 8.5: Process CloseReq
func (c *Conn) step13_ProcessCloseReq(h *Header) os.Error {
	if h.Type == CloseReq && c.socket.GetState() < CLOSEREQ {
		c.inject(c.generateClose())
		c.gotoCLOSING()
	}
	return nil
}

// Step 14, Section 8.5: Process Close
func (c *Conn) step14_ProcessClose(h *Header) os.Error {
	if h.Type != Close {
		return nil
	}
	c.teardownUser()
	c.gotoCLOSED()
	c.inject(c.generateReset(ResetClosed))
	c.teardownWriteLoop()
	return ErrDrop
}

// Step 15, Section 8.5: Process Sync
func (c *Conn) step15_ProcessSync(h *Header) os.Error {
	if h.Type == Sync {
		c.inject(c.generateSyncAck(h))
	}
	return nil
}

// Step 16, Section 8.5: Process Data
func (c *Conn) step16_ProcessData(h *Header) os.Error {
	// At this point any application data on P can be passed to the
	// application, except that the application MUST NOT receive data from
	// more than one Request or Response

	// REMARK: For now, we accept data only on Data* packets
	if h.Type != Data && h.Type != DataAck {
		return nil
	}

	// DCCP-Data, DCCP-DataAck, and DCCP-Ack packets received in CLOSEREQ or
	// CLOSING states MAY be either processed or ignored.

	// Drop data packets if application does not read them fast enough
	c.readAppLk.Lock()
	if c.readApp != nil {
		if len(c.readApp) < cap(c.readApp) {
			c.readApp <- h.Data
		} else {
			log.Printf("dropping recvd packet, slow app\n")
		}
	}
	c.readAppLk.Unlock()

	return nil
}
