// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"time"
	"github.com/petar/GoGauge/context"
)

// logger is a logging facility
type logger struct {
	*context.Context
}

func (t *logger) Init(c *context.Context) {
	t.Context = c
}

func (t *logger) GetState() string {
	r := t.Context.GetRoot()
	return r.GetAttr("state").(string)
}

func (t *logger) SetState(s int) {
	t.Context.SetAttr("state", StateString(s))
}

func (t *logger) GetFullName() string {
	cached := t.Context.GetAttr("full")
	if cached != nil {
		return cached.(string)
	}
	p := t.Context.NamePath()
	full := ""
	for i := 0; i < len(p); i++ {
		full = full + p[len(p)-1-i]
		if i+1 < len(p) {
			full += "_"
		}
	}
	t.Context.SetAttr("full", full)
	return full
}

func (t *logger) Emit(typ string, s string) {
	fmt.Printf("%d @%-8s %s %s %s", time.Nanoseconds(), t.GetState(), typ, t.GetFullName(), s)
}

// Logging utility functions

func (c *Conn) logState() {
	c.AssertLocked()
	c.log.SetState(c.socket.GetState())
}

func (c *Conn) logReadHeader(h *Header) {
	c.log.Emit("R", h.String())
}

func (c *Conn) logWriteHeader(h *Header) {
	c.log.Emit("W", h.String())
}

func (c *Conn) logEvent(s string) {
	c.log.Emit("E", s)
}

func (c *Conn) logWarn(s string) {
	c.log.Emit("?", s)
}
