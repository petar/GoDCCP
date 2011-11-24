// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"encoding/json"
	"os"
	"path"
	goruntime "runtime"
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
	Time      int64   //`json:"t"`   // Time of event
	Module    string  //`json:"m"`   // Module where event occurred, e.g. "server", "client", "line"
	Submodule string  //`json:"sm"`  // Submodule where event occurred, e.g. "s-strober"
	Event     string  //`json:"e"`   // Type of event
	State     string  //`json:"s"`   // State of module
	Comment   string  //`json:"c"`   // Textual comment describing the event

	Type      string  //`json:"t"`
	SeqNo     int64   //`json:"sn"`
	AckNo     int64   //`json:"an"`

	SourceFile string //`json:"sf"`
	SourceLine int    //`json:"sl"`
}

func (t Logger) Emit(submodule string, event string, h interface{}, comment string, v ...interface{}) {
	t.EmitCaller(1, submodule, event, h, comment, v...)
}

func (t Logger) EmitCaller(level int, submodule string, event string, h interface{}, comment string, v ...interface{}) {
	if t == "" {
		return
	}
	if !gauge.Selected(t.GetName(), submodule) {
		return
	}
	sinceZero, sinceLast := SnapLog()

	// Extract header information
	var hType string = "NIL"
	var hSeqNo, hAckNo int64
	switch t := h.(type) {
	case *Header:
		if t != nil {
			hSeqNo, hAckNo = t.SeqNo, t.AckNo
			hType = typeString(t.Type)
		}
	case *PreHeader:
		if t != nil {
			hSeqNo, hAckNo = t.SeqNo, t.AckNo
			hType = typeString(t.Type)
		}
	case *FeedbackHeader:
		if t != nil {
			hSeqNo, hAckNo = t.SeqNo, t.AckNo
			hType = typeString(t.Type)
		}
	case *FeedforwardHeader:
		if t != nil {
			hSeqNo = t.SeqNo
			hType = typeString(t.Type)
		}
	}

	_, sfile, sline, _ := goruntime.Caller(1+level)
	sdir, sfile := path.Split(sfile)
	if len(sdir) > 0 {
		_, sdir = path.Split(sdir[:len(sdir)-1])
	}
	sfile = path.Join(sdir, sfile)

	if len(v) > 0 {
		comment = fmt.Sprintf(comment, v...)
	}

	if logWriter != nil {
		r := &LogRecord{
			Time:       sinceZero,
			Module:     t.GetName(),
			Submodule:  submodule,
			Event:      event,
			State:      t.GetState(),
			Comment:    comment,
			Type:       hType,
			SeqNo:      hSeqNo,
			AckNo:      hAckNo,
			SourceFile: sfile,
			SourceLine: sline,
		}
		logWriter.Write(r)
	}
	if os.Getenv("DCCPRAW") != "" {
		fmt.Printf("%15s %15s %18s:%-3d %-8s %6s:%-11s %-7s %8s %6x|%-6x * %s\n", 
			Nstoa(sinceZero), Nstoa(sinceLast), 
			sfile, sline,
			t.GetState(), t.GetName(), 
			submodule, event, 
			hType, hSeqNo, hAckNo,
			comment,
		)
	}
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

func Nstoa(ns int64) string {
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
	os.Remove(name)
	f, err := os.Create(name)
	if err != nil {
		panic("cannot create log file")
	}
	w := &FileLogWriter{f, json.NewEncoder(f)}
	goruntime.SetFinalizer(w, func(w *FileLogWriter) { 
		fmt.Printf("Flushing log\n")
		w.f.Close() 
	})
	return w
}

func (t *FileLogWriter) Write(r *LogRecord) {
	err := t.enc.Encode(r)
	if err != nil {
		panic("error encoding log entry")
	}
}

func (t *FileLogWriter) Sync() error {
	return t.f.Sync()
}

func (t *FileLogWriter) Close() error {
	return t.f.Close()
}

var logWriter LogWriter

// SetLogWriter sets the DCCP-wide LogWriter facility
func SetLogWriter(e LogWriter) { logWriter = e }
