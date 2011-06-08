// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
)

func (c *Conn) shortID() string {
	var w bytes.Buffer
	c.Lock()
	isServer := c.socket.IsServer()
	c.Unlock()
	p := c.hc.LocalLabel().Bytes()
	if isServer {
		fmt.Fprintf(&w, "S·%2x", p[0])
	} else {
		fmt.Fprintf(&w, "C·%2x", p[0])
	}
	return string(w.Bytes())
}

func (c *Conn) logState() {
	c.Lock()
	state := c.socket.GetState()
	c.Unlock()
	fmt.Printf("%s—%s\n", c.shortID(), StateString(state))
}

func (c *Conn) logReadHeader(h *Header) {
	fmt.Printf("%s/R %s\n", c.shortID(), h.String())
}

func (c *Conn) logWriteHeader(h *Header) {
	fmt.Printf("%s/W %s\n", c.shortID(), h.String())
}
