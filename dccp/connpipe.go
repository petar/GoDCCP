// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"log"
	"os"
)

func (c *Conn) readHeader() (h *Header, err os.Error) {
	h, err = c.hc.ReadHeader()
	if err != nil {
		if err != ErrTimeout {
			log.Printf("dropping\n")
		}
		return nil, err
	}
	// We don't support non-extended (short) SeqNo's 
	if !h.X {
		return nil, ErrUnsupported
	}
	return h, nil
}

func (c *Conn) readLoop() {
	if err := c.hc.SetReadTimeout(MSL); err != nil {
		log.Printf("SetReadTimeout failed")
		c.kill()
		return
	}
	for {
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		if state == CLOSED {
			break
		}
		h, err := c.readHeader()
		if err != nil {
			_, protoError := err.(ProtoError)
			if protoError || err == os.EAGAIN {
				// Drop packets that are unsupported or if there is timeout. 
				// Intended for forward compatibility.
				continue
			} else {
				// Die if the underlying link is broken
				c.kill()
				return
			}
		}
		c.logReadHeader(h)

		c.Lock()
		c.updateSocketCongestionControl()
		if c.step2_ProcessTIMEWAIT(h) != nil {
			goto Done
		}
		if c.step3_ProcessLISTEN(h) != nil {
			goto Done
		}
		if c.step4_PrepSeqNoREQUEST(h) != nil {
			goto Done
		}
		if c.step5_PrepSeqNoForSync(h) != nil {
			goto Done
		}
		if c.step6_CheckSeqNo(h) != nil {
			goto Done
		}
		if c.step7_CheckUnexpectedTypes(h) != nil {
			goto Done
		}
		if c.step8_OptionsAndMarkAckbl(h) != nil {
			goto Done
		}
		if c.step9_ProcessReset(h) != nil {
			goto Done
		}
		if c.step10_ProcessREQUEST2(h) != nil {
			goto Done
		}
		if c.step11_ProcessRESPOND(h) != nil {
			goto Done
		}
		if c.step12_ProcessPARTOPEN(h) != nil {
			goto Done
		}
		if c.step13_ProcessCloseReq(h) != nil {
			goto Done
		}
		if c.step14_ProcessClose(h) != nil {
			goto Done
		}
		if c.step15_ProcessSync(h) != nil {
			goto Done
		}
		if c.step16_ProcessData(h) != nil {
			goto Done
		}
	Done:
		c.Unlock()
	}
}

func (c *Conn) updateSocketCongestionControl() {
	c.AssertLocked()
	swaf, swbf := c.cc.GetSWABF()
	c.socket.SetSWABF(swaf, swbf)
	c.socket.SetRTT(c.cc.GetRTT())
	c.socket.SetCCMPS(c.cc.GetCCMPS())
}

func (c *Conn) updateSocketLink() {
	c.AssertLocked()
	c.socket.SetPMTU(int32(c.hc.GetMTU()))
}
