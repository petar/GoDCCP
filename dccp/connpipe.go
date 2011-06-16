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

// How often we exit from a blocking call to readHeader, 1 sec in nanoseconds
const READ_TIMEOUT = 1e9 

func (c *Conn) readLoop() {
	for {
		c.Lock()
		state := c.socket.GetState()
		rtt := c.socket.GetRTT()
		c.Unlock()
		if state == CLOSED {
			break
		}

		// Adjust read timeout
		if err := c.hc.SetReadTimeout(5 * rtt); err != nil {
			log.Printf("SetReadTimeout failed")
			c.abortQuietly()
			return
		}

		// Read next header
		h, err := c.readHeader()
		if err != nil {
			_, ok := err.(ProtoError)
			if ok {
				// Drop packets that are unsupported. Intended for forward compatibility.
				continue
			} else if err == os.EAGAIN {
				// In the even of timeout, poll the congestion controls
				c.pollCongestionControl()
				continue
			} else {
				// Die if the underlying link is broken
				c.abortQuietly()
				return
			}
		}
		c.logReadHeader(h)

		c.Lock()
		c.syncWithCongestionControl()
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

func (c *Conn) pollCongestionControl() {
	if e := c.scc.OnIdle(); e != nil {
		if re, ok := e.(CongestionReset); ok {
			c.abortWith(re.ResetCode())
			return
		}
		if e == CongestionAck {
			c.Lock()
			c.inject(c.generateAck())
			c.Unlock()
			return
		}
		log.Printf("unknown sender cc idle event")
	}
	if e := c.rcc.OnIdle(); e != nil {
		if re, ok := e.(CongestionReset); ok {
			c.abortWith(re.ResetCode())
			return
		}
		if e == CongestionAck {
			c.Lock()
			c.inject(c.generateAck())
			c.Unlock()
			return
		}
		log.Printf("unknown receiver cc idle event")
	}
}

func (c *Conn) syncWithCongestionControl() {
	c.AssertLocked()
	c.socket.SetRTT(c.scc.GetRTT())
	c.socket.SetCCMPS(c.scc.GetCCMPS())
}

func (c *Conn) syncWithLink() {
	c.AssertLocked()
	c.socket.SetPMTU(int32(c.hc.GetMTU()))
}
