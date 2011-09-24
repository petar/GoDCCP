// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"time"
	"github.com/petar/GoGauge/dyna"
)

// TODO:
//   * fuse conn name and conn literal

// CLog is a logging facility
type CLog string

func (t *CLog) Init(name string) {
	*t = CLog(name)
}

func (t CLog) GetName() string {
	return string(t)
}

func (t CLog) GetState() string {
	// The first literal holds the dynamic name for the connection
	return dyna.GetAttr([]string{t.GetName()}, "state").(string)
}

func (t CLog) SetState(s int) {
	dyna.SetAttr([]string{t.GetName()}, "state", StateString(s))
}

func (t CLog) Logf(modifier string, typ string, format string, v ...interface{}) {
	if !dyna.Selected(t.GetName(), modifier) {
		return
	}
	fmt.Printf("%d  @%-8s  %6s:%-7s  %-5s  ——  %s\n", 
		time.Nanoseconds(), t.GetState(), t.GetName(), modifier,
		typ, fmt.Sprintf(format, v...),
	)
}

// Logging utility functions

func (c *Conn) logState() {
	c.AssertLocked()
	c.CLog.SetState(c.socket.GetState())
}

func (c *Conn) logReadHeader(h *Header) {
	c.CLog.Logf("conn", "Read", h.String())
}

func (c *Conn) logWriteHeader(h *Header) {
	c.CLog.Logf("conn", "Write", h.String())
}

func (c *Conn) logEvent(s string) {
	c.CLog.Logf("conn", "Event", s)
}

func (c *Conn) logWarn(s string) {
	c.CLog.Logf("conn", "Warn", s)
}
