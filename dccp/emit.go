// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"runtime/debug"
)

func (c *Conn) emitCatchSeqNo(h *Header, seqNos ...int64) {
	if h == nil { 
		return
	}
	for _, seqNo := range seqNos {
		if h.SeqNo == seqNo {
			c.logger.EC(1, "conn", "Catch", 
				fmt.Sprintf("Caught SeqNo=%d: %s\n%s", seqNo, h.String(), string(debug.Stack())), 
				h)
			break
		}
	}
}

func (c *Conn) emitSetState() {
	c.AssertLocked()
	c.logger.SetState(c.socket.GetState())
}
