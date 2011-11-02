// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"github.com/petar/GoGauge/gauge"
)

type Logger string

var NoLogging Logger = ""

func (t Logger) GetName() string {
	return string(t)
}

func (t Logger) GetState() string {
	g := gauge.GetAttr([]string{t.GetName()}, "state")
	if g == nil {
		return ""
	}
	return g.(string)
}

func (t Logger) SetState(s int) {
	gauge.SetAttr([]string{t.GetName()}, "state", StateString(s))
}

// LogRecord stores a log event. It can be used to marshal to JSON and pass to external
// visualisation tools.
type LogRecord struct {
	Time      int64   // Time of event
	SeqNo     int64   // SeqNo, if applicable; zero otherwise
	AckNo     int64   // AckNo, if applicable; zero otherwise
	Module    string  // Module where event occurred, e.g. "server", "client", "line"
	Submodule string  // Submodule where event occurred, e.g. "s-strober"
	Type      string  // Type of event
	State     string  // State of module
	Comment   string  // Textual comment describing the event
}

func (t Logger) Logf(submodule string, typ string, seqno, ackno int64, comment string, v ...interface{}) {
	if t == "" {
		return
	}
	if !gauge.Selected(t.GetName(), submodule) {
		return
	}
	sinceZero, sinceLast := SnapLog()
	if emitter != nil {
		emitter.Emit(&LogRecord{
			Time:      sinceZero,
			SeqNo:     seqno,
			AckNo:     ackno,
			Module:    t.GetName(),
			Submodule: submodule,
			Type:      typ,
			State:     t.GetState(),
			Comment:   fmt.Sprintf(comment, v...),
		})
	} else {
		fmt.Printf("%15s %15s  %-8s  %6s:%-11s  %-7s  ——  %s\n", 
			nstoa(sinceZero), nstoa(sinceLast), t.GetState(), t.GetName(), 
			submodule, indentType(typ), fmt.Sprintf(comment, v...),
		)
	}
}

func indentType(typ string) string {
	switch typ {
	case "Write", "Read", "Strobe":
		return typ
	default:
		return "  " + typ
	}
	panic("")
}

const nsAlpha = "0123456789"

func nstoa(ns int64) string {
	if ns < 0 {
		panic("negative time")
	}
	if ns == 0 {
		return "0"
	}
	b := make([]byte, 32)
	z := len(b) - 1
	i := 0
	j := 0
	for ns != 0 {
		if j % 3 == 0 && j > 0 {
			b[z-i] = ','
			i++
		}
		b[z-i] = nsAlpha[ns % 10]
		j++
		i++
		ns /= 10
	}
	return string(b[z-i+1:])
}

// LogEmitter is a type that consumes log entries.
type LogEmitter interface {
	Emit(*LogRecord)
}

var emitter LogEmitter

// SetLogEmitter sets the DCCP-wide LogEmitter facility
func SetLogEmitter(e LogEmitter) { emitter = e }
