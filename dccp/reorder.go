// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// WARN: reorderBuffer applied to all DCCP traffic may introduce significant
// delay in interactive sections like the initial handshake e.g.
// Not in use. Implementation incomplete until XXX addressed.

// reorderBuffer consumes headers and outputs them in sequence number order. 
// The output headers are guaranteed to have increasing sequence numbers.
type reorderBuffer struct {
	headers   []*Header
	lastIssue int64      // Sequence number of past packet released for reading
}

// Init prepares the reorderBuffer for new use
func (t *reorderBuffer) Init(size int) {
	t.headers = make([]*Header, size)
	t.lastIssue = 0
}

func (t *reorderBuffer) PushPop(h *Header) *Header {
	// XXX: Must guarantee strictly ascending order
	var popSeqNo int64 = dccp.SEQNOMAX + 1
	var popIndex int
	for i, g := range t.headers {
		if g == nil {
			t.headers[i] = h
			return nil
		}
		// XXX: This must employ circular comparison
		if g.SeqNo < popSeqNo {
			popIndex = i
			popSeqNo = g.SeqNo
		}
	}
	r := t.headers[popIndex]
	t.headers[popIndex] = h
	return r
}
