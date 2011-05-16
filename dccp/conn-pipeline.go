// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

func (c *Conn) readHeader() (h *Header, err os.Error) {
	h, err = c.hc.ReadHeader()
	if err != nil {
		return nil, err
	}
	// We don't support non-extended (short) SeqNo's 
	if !h.X {
		return nil, ErrUnsupported
	}
	return h, nil
}

// XXX: Maybe this loop can lock on socket on behalf of all functions called inside of it.
// XXX: See if calls from one step to another mess with the global call sequence in here
func (c *Conn) readLoop() {
	for {
		h, err := e.readHeader()
		if err != nil {
			continue // drop packets that are unsupported. Forward compatibility
		}

		c.slk.Lock()
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
		...
	Done:
		c.slk.Unlock()
		// XXX: Decide if it is time to end the loop
		???
	}
}
