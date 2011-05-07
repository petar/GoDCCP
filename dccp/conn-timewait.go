// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// newAbnormalReset() generates a new Reset header, according to Section 8.3.1
func (c *Conn) newAbnormalReset(h *Header) *Header {
	var seqno uint64 = 0
	if h.HasAckNo() {
		seqno = h.AckNo+1
	}
	return NewResetHeader(ResetNoConnection, c.id.SourcePort, c.id.DestPort, seqno, h.SeqNo)
}

// If socket is in TIMEWAIT, it must perform a Reset sequence.
// Implements the second half of Step 2, Section 8.5
func (c *Conn) processTIMEWAIT(h *Header) os.Error {
	if h.Type == Reset {
		return ErrDrop
	}
	return c.inject(c.newAbnormalReset(h))
}
