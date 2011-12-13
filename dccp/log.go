// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"encoding/json"
	"os"
	goruntime "runtime"
	"github.com/petar/GoGauge/filter"
)

// LogRecord stores a log event. It can be used to marshal to JSON and pass to external
// visualisation tools.
type LogRecord struct {
	Time      int64   //`json:"t"`   // Time of event
	Module    string  //`json:"m"`   // Module where event occurred, e.g. "server", "client", "line"
	Submodule string  //`json:"sm"`  // Submodule
	Event     string  //`json:"e"`   // Type of event
	State     string  //`json:"s"`   // State of module
	Comment   string  //`json:"c"`   // Textual comment describing the event
	Args      LogArgs                // Any additional arguments that need to be attached to the log

	Type      string  //`json:"t"`   // Type of header, if applicable
	SeqNo     int64   //`json:"sn"`  // Sequence number, if applicable
	AckNo     int64   //`json:"an"`  // Acknowledgement Number, if applicable

	SourceFile string //`json:"sf"`
	SourceLine int    //`json:"sl"`
}

type LogArgs map[string]interface{}

func (t LogArgs) Int64(k string) (value int64, success bool) {
	if t == nil {
		return 0, false
	}
	v, ok := t[k]
	if !ok {
		return 0, false
	}
	u, ok := v.(int64)
	if !ok {
		return 0, false
	}
	return u, true
}

func (t LogArgs) Bool(k string) (value bool, success bool) {
	if t == nil {
		return false, false
	}
	v, ok := t[k]
	if !ok {
		return false, false
	}
	u, ok := v.(bool)
	if !ok {
		return false, false
	}
	return u, true
}

// Logger is capable of emitting structured logs, which are consequently used for debuging
// and analysis purposes. It lives in the context of a shared time framework and a shared
// filter framework, which may filter some logs out
type Logger struct {
	run  *Runtime
	name string
}

// A zero-value Logger ignores all emits
var NoLogging *Logger = &Logger{}

func NewLogger(name string, run *Runtime) *Logger {
	return &Logger{ run: run, name: name }
}

func (t *Logger) Name() string {
	return t.name
}

func (t *Logger) Filter() *filter.Filter {
	return t.run.Filter()
}

func (t *Logger) GetState() string {
	if t.run == nil {
		return ""
	}
	g := t.run.Filter().GetAttr([]string{t.Name()}, "state")
	if g == nil {
		return ""
	}
	return g.(string)
}

func (t *Logger) SetState(s int) {
	if t.run == nil {
		return
	}
	t.run.Filter().SetAttr([]string{t.Name()}, "state", StateString(s))
}

// E emits a new log record.
// The arguments args are scanned in turn. The first argument of type
// *Header, *PreHeader, *FeedbackHeader or *FeedforwardHeader is treated
// as the DCCP header involved in this log. The first argument of type
// Args is saved in the log record.
func (t *Logger) E(submodule, event, comment string, args ...interface{}) {
	t.EC(1, submodule, event, comment, args...)
}

func (t *Logger) EC(level int, submodule, event, comment string, args ...interface{}) {
	if t.run == nil {
		return
	}
	/*
	if !t.run.Filter().Selected(t.Name(), submodule) {
		return
	}
	*/
	sinceZero, sinceLast := t.run.Snap()

	// Extract header information
	var hType string = "nil"
	var hSeqNo, hAckNo int64
__FindHeader:
	for _, a := range args {
		switch t := a.(type) {
		case *Header:
			if t != nil {
				hSeqNo, hAckNo = t.SeqNo, t.AckNo
				hType = typeString(t.Type)
			}
			break __FindHeader
		case *PreHeader:
			if t != nil {
				hSeqNo, hAckNo = t.SeqNo, t.AckNo
				hType = typeString(t.Type)
			}
			break __FindHeader
		case *FeedbackHeader:
			if t != nil {
				hSeqNo, hAckNo = t.SeqNo, t.AckNo
				hType = typeString(t.Type)
			}
			break __FindHeader
		case *FeedforwardHeader:
			if t != nil {
				hSeqNo = t.SeqNo
				hType = typeString(t.Type)
			}
			break __FindHeader
		}
	}
	var largs LogArgs
__FindArgs:
	for _, a := range args {
		m, ok := a.(LogArgs)
		if ok {
			largs = m
			break __FindArgs
		}
	}

	sfile, sline := FetchCaller(1+level)

	if t.run.Writer() != nil {
		r := &LogRecord{
			Time:       sinceZero,
			Module:     t.Name(),
			Submodule:  submodule,
			Event:      event,
			State:      t.GetState(),
			Comment:    comment,
			Args:       largs,
			Type:       hType,
			SeqNo:      hSeqNo,
			AckNo:      hAckNo,
			SourceFile: sfile,
			SourceLine: sline,
		}
		t.run.Writer().Write(r)
	}
	if os.Getenv("DCCPRAW") != "" {
		fmt.Printf("%15s %15s %18s:%-3d %-8s %6s:%-9s %-9s %8s %6x|%-6x * %s\n", 
			Nstoa(sinceZero), Nstoa(sinceLast), 
			sfile, sline,
			t.GetState(), t.Name(), 
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
	Sync() error
	Close() error
}

// FileLogWriter saves all log entries to a file in JSON format
type FileLogWriter struct {
	f   *os.File
	enc *json.Encoder
	dup LogWriter
}

// NewFileLogWriter creates a new log writer. It panics if the file cannot be created.
func NewFileLogWriterDup(name string, dup LogWriter) *FileLogWriter {
	os.Remove(name)
	f, err := os.Create(name)
	if err != nil {
		panic(fmt.Sprintf("cannot create log file '%s'", name))
	}
	w := &FileLogWriter{ f, json.NewEncoder(f), dup }
	goruntime.SetFinalizer(w, func(w *FileLogWriter) { 
		fmt.Printf("Flushing log\n")
		w.f.Close() 
	})
	return w
}

func NewFileLogWriter(name string) *FileLogWriter {
	return NewFileLogWriterDup(name, nil)
}

func (t *FileLogWriter) Write(r *LogRecord) {
	err := t.enc.Encode(r)
	if err != nil {
		panic(fmt.Sprintf("error encoding log entry (%s)", err))
	}
	if t.dup != nil {
		t.dup.Write(r)
	}
}

func (t *FileLogWriter) Sync() error {
	if t.dup != nil {
		t.dup.Sync()
	}
	return t.f.Sync()
}

func (t *FileLogWriter) Close() error {
	if t.dup != nil {
		t.dup.Close()
	}
	return t.f.Close()
}
