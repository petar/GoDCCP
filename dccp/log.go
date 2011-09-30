// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"runtime/debug"
	"time"
	"github.com/petar/GoGauge/gauge"
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
	// The first literal holds the gaugemic name for the connection
	g := gauge.GetAttr([]string{t.GetName()}, "state")
	if g == nil {
		return ""
	}
	return g.(string)
}

func (t CLog) SetState(s int) {
	gauge.SetAttr([]string{t.GetName()}, "state", StateString(s))
}

func (t CLog) Logf(modifier string, typ string, format string, v ...interface{}) {
	if !gauge.Selected(t.GetName(), modifier) {
		return
	}
	fmt.Printf("%d  @%-8s  %6s:%-11s  %-5s  ——  %s\n", 
		time.Nanoseconds(), t.GetState(), t.GetName(), modifier,
		typ, fmt.Sprintf(format, v...),
	)
}

// Logging utility functions

func (c *Conn) logCatchSeqNo(h *Header, seqNos ...int64) {
	if h == nil { 
		return
	}
	for _, seqNo := range seqNos {
		if h.SeqNo == seqNo {
			c.CLog.Logf("conn", "Catch", "Caught SeqNo=%d: %s\n%s", 
				seqNo, h.String(), string(debug.Stack()))
			break
		}
	}
}

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