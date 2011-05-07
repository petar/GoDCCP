// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

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

func (c *Conn) readLoop() {
	for {
		h, err := e.readPacket()
		if err != nil {
			continue XX // no continue
		}
		?
	}
}
