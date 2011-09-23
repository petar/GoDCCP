// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"time"
	"github.com/petar/GoGauge/dyna"
)

// DLog is a logging facility
type DLog dyna.T

func (t *DLog) Init(parentDLog DLog, literals ...string) {
	*t = DLog(append([]string{dyna.T(parentDLog).Get(0)}, literals...))
}

func (t DLog) GetName() string {
	return dyna.T(t).Get(0)
}

func (t DLog) GetState() string {
	// The first literal holds the dynamic name for the connection
	return dyna.GetAttr(dyna.T{dyna.T(t).Get(0), "conn"}, "state").(string)
}

func (t DLog) SetState(s int) {
	dyna.SetAttr(dyna.T{dyna.T(t).Get(0), "conn"}, "state", StateString(s))
}

func (t DLog) GetFullName() string {
	ss := dyna.T(t).Strings()
	full := ss[0]+":"
	for i := 1; i < len(ss); i++ {
		full += ss[i]
		if i+1 < len(t) {
			full += "·"
		}
	}
	return full
}

func (t DLog) Emit(typ string, s string) {
	if !dyna.T(t).Selected() {
		return
	}
	fmt.Printf("%d @%-8s %s %s —— %s\n", time.Nanoseconds(), t.GetState(), typ, t.GetFullName(), s)
}

func (t DLog) Emitf(typ string, format string, v ...interface{}) {
	if !dyna.T(t).Selected() {
		return
	}
	fmt.Printf("%d @%-8s %s %s —— %s\n", 
		time.Nanoseconds(), t.GetState(), typ, t.GetFullName(), 
		fmt.Sprintf(format, v...),
	)
}

// Logging utility functions

func (c *Conn) logState() {
	c.AssertLocked()
	c.DLog.SetState(c.socket.GetState())
}

func (c *Conn) logReadHeader(h *Header) {
	c.DLog.Emit("R", h.String())
}

func (c *Conn) logWriteHeader(h *Header) {
	c.DLog.Emit("W", h.String())
}

func (c *Conn) logEvent(s string) {
	c.DLog.Emit("E", s)
}

func (c *Conn) logWarn(s string) {
	c.DLog.Emit("?", s)
}
