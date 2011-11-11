// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"encoding/json"
	"os"
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
	Module    string  // Module where event occurred, e.g. "server", "client", "line"
	Submodule string  // Submodule where event occurred, e.g. "s-strober"
	Event     string  // Type of event
	State     string  // State of module
	Comment   string  // Textual comment describing the event

	Type     string
	SeqNo    int64
	AckNo    int64
}

func (t Logger) Logf(submodule string, event string, h interface{}, comment string, v ...interface{}) {
	if t == "" {
		return
	}
	if !gauge.Selected(t.GetName(), submodule) {
		return
	}
	sinceZero, sinceLast := SnapLog()
	var hType string
	var hSeqNo, hAckNo int64
	switch t := h.(type) {
	case *Header:
		hSeqNo, hAckNo = t.SeqNo, t.AckNo
		hType = typeString(t.Type)
	case *PreHeader:
		hSeqNo, hAckNo = t.SeqNo, t.AckNo
		hType = typeString(t.Type)
	case *FeedbackHeader:
		hSeqNo, hAckNo = t.SeqNo, t.AckNo
		hType = typeString(t.Type)
	case *FeedforwardHeader:
		hSeqNo = t.SeqNo
		hType = typeString(t.Type)
	}
	if logWriter != nil {
		r := &LogRecord{
			Time:      sinceZero,
			Module:    t.GetName(),
			Submodule: submodule,
			Event:     event,
			State:     t.GetState(),
			Comment:   fmt.Sprintf(comment, v...),
			Type:      hType,
			SeqNo:     hSeqNo,
			AckNo:     hAckNo,
		}
		logWriter.Write(r)
	}
	fmt.Printf("%15s %15s  %-8s  %6s:%-11s  %-7s  %8s  %8d·%-8d  ——  %s\n", 
		nstoa(sinceZero), nstoa(sinceLast), t.GetState(), t.GetName(), 
		submodule, event, 
		hType, hSeqNo, hAckNo,
		fmt.Sprintf(comment, v...),
	)
}

func indentEvent(event string) string {
	switch event {
	case "Write", "Read", "Strobe":
		return event
	default:
		return "  " + event
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

// LogWriter is a type that consumes log entries.
type LogWriter interface {
	Write(*LogRecord)
}

// FileLogWriter saves all log entries to a file in JSON format
type FileLogWriter struct {
	f   *os.File
	enc *json.Encoder
}

func NewFileLogWriter(name string) *FileLogWriter {
	f, err := os.Create(name)
	if err != nil {
		panic("cannot create log file")
	}
	return &FileLogWriter{f, json.NewEncoder(f)}
}

func (t *FileLogWriter) Write(r *LogRecord) {
	err := t.enc.Encode(r)
	if err != nil {
		panic("error encoding log entry")
	}
}

var logWriter LogWriter = NewFileLogWriter(os.Getenv("DCCPLOG"))

// SetLogWriter sets the DCCP-wide LogWriter facility
func SetLogWriter(e LogWriter) { logWriter = e }
