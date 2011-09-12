// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
)

func (c *Conn) shortID() string {
	c.AssertLocked()
	var w bytes.Buffer
	isServer := c.socket.IsServer()
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
	id := c.shortID()
	c.Unlock()
	fmt.Printf("%s—%s\n", id, StateString(state))
}

func (c *Conn) logReadHeader(h *Header) {
	c.Lock()
	state := c.socket.GetState()
	id := c.shortID()
	c.Unlock()
	fmt.Printf("%s/R/%s —— %s\n", id, StateString(state), h.String())
}

func (c *Conn) logWriteHeader(h *Header) {
	c.Lock()
	state := c.socket.GetState()
	id := c.shortID()
	c.Unlock()
	fmt.Printf("%s/W/%s —— %s\n", id, StateString(state), h.String())
}

func (c *Conn) logWriteHeaderLocked(h *Header) {
	c.AssertLocked()
	state := c.socket.GetState()
	id := c.shortID()
	fmt.Printf("%s/W/%s —— %s\n", id, StateString(state), h.String())
}
