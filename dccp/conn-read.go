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

func (c *Conn) state() {
}

func (c *Conn) readLoop() {
	for {
		h, err := e.readHeader()
		if err != nil {
			continue // drop packets that are unsupported
		}
		c.slk.Lock()
		state = c.socket.GetState()
		s.slk.Unlock()
		switch state {
		case CLOSED:
			err = c.processCLOSED(h)
		case LISTEN:
			err = c.processLISTEN(h)
		case REQUEST:
			err = c.processREQUEST(h)
		case RESPOND:
			err = c.processRESPOND(h)
		case PARTOPEN:
			err = c.processPARTOPEN(h)
		case OPEN:
			err = c.processOPEN(h)
		case CLOSEREQ:
			err = c.processCLOSEREQ(h)
		case CLOSING:
			err = c.processCLOSING(h)
		case TIMEWAIT:
			err = c.processTIMEWAIT(h)
		default:
			panic("invalid dccp socket state")
		}
		// XXX: Decide if it is time to end the loop
		???
	}
}
