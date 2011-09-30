// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"runtime/debug"
	"github.com/petar/GoGauge/gauge"
)

type Logger interface {
	GetName() string
	GetState() string
	SetState(s int)
	Logf(modifier string, typ string, format string, v ...interface{})
}

// NoLogging is a specialization of Logger that does not do anything
type NoLogging struct {}

func (NoLogging) GetName() string { 
	return "" 
}

func (NoLogging) GetState() string {
	return ""
}

func (NoLogging) SetState(s int) {}

func (NoLogging) Logf(modifier string, typ string, format string, v ...interface{}) {}

// detailLogger is a specialization of Logger that uses GoGauge to perform dynamic filtered logging
type detailLogger struct {
	Time
	name string
}

func NewLogger(time Time, name string) Logger {
	return &detailLogger{
		Time: time,
		name: name,
	}
}

func (t detailLogger) GetName() string {
	return t.name
}

func (t detailLogger) GetState() string {
	// The first literal holds the gaugemic name for the connection
	g := gauge.GetAttr([]string{t.GetName()}, "state")
	if g == nil {
		return ""
	}
	return g.(string)
}

func (t detailLogger) SetState(s int) {
	gauge.SetAttr([]string{t.GetName()}, "state", StateString(s))
}

func (t detailLogger) Logf(modifier string, typ string, format string, v ...interface{}) {
	if !gauge.Selected(t.GetName(), modifier) {
		return
	}
	fmt.Printf("%d  @%-8s  %6s:%-11s  %-5s  ——  %s\n", 
		t.Time.Nanoseconds(), t.GetState(), t.GetName(), modifier,
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
			c.Logger.Logf("conn", "Catch", "Caught SeqNo=%d: %s\n%s", 
				seqNo, h.String(), string(debug.Stack()))
			break
		}
	}
}

func (c *Conn) logState() {
	c.AssertLocked()
	c.Logger.SetState(c.socket.GetState())
}

func (c *Conn) logReadHeader(h *Header) {
	c.Logger.Logf("conn", "Read", h.String())
}

func (c *Conn) logWriteHeader(h *Header) {
	c.Logger.Logf("conn", "Write", h.String())
}

func (c *Conn) logEvent(s string) {
	c.Logger.Logf("conn", "Event", s)
}

func (c *Conn) logWarn(s string) {
	c.Logger.Logf("conn", "Warn", s)
}
