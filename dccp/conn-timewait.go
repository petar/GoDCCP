// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// If endpoint is in TIMEWAIT, it must perform a Reset sequence
// Implements the second half of Step 2, Section 8.5
func (c *Conn) readInTIMEWAIT(h *Header) os.Error {
	if h.Type == Reset {
		return
	}
	var seqno uint64 = 0
	if h.HasAckNo() {
		seqno = h.AckNo+1
	}
	?
	g := NewResetHeader(ResetNoConnection, c.id.SourcePort, c.id.DestPort, seqno, h.SeqNo)
	hdr, err := g.Write(c.id.SourceAddr, c.id.DestAddr, AnyProto, false)
	if err != nil {
		return err
	}
	return c.inject(hdr)
}
