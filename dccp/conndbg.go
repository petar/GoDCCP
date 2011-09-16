// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
	"log"
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

func (c *Conn) stateString() string {
	c.Lock()
	defer c.Unlock()
	return c.stateStringLocked()
}

func (c *Conn) stateStringLocked() string {
	c.AssertLocked()
	var w bytes.Buffer
	fmt.Fprintf(&w, "%s @%-8s", c.name, StateString(c.socket.GetState()))
	return string(w.Bytes())
}

func (c *Conn) logState() {
	log.Printf(c.stateString())
}

func (c *Conn) logReadHeader(h *Header) {
	log.Printf("%s R —— %s\n", c.stateString(), h.String())
}

func (c *Conn) logWriteHeader(h *Header) {
	log.Printf("%s W —— %s\n", c.stateString(), h.String())
}

func (c *Conn) logWriteHeaderLocked(h *Header) {
	c.AssertLocked()
	log.Printf("%s W —— %s\n", c.stateStringLocked(), h.String())
}

func (c *Conn) logEvent(s string) {
	c.AssertLocked()
	log.Printf("%s * %s", c.stateStringLocked(), s)
}

func (c *Conn) logWarn(s string) {
	c.AssertLocked()
	log.Printf("%s · %s", c.stateStringLocked(), s)
}
