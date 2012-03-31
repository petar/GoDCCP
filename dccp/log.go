// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
	"runtime"
	"encoding/json"
	"os"
	goruntime "runtime"
	"github.com/petar/GoGauge/filter"
)

// LogRecord stores a log event. It can be used to marshal to JSON and pass to external
// visualisation tools.
type LogRecord struct {
	Time      int64   `json:"t"`   // Time of event

	// System is a runtime identifier for a dynamic instance of some DCCP type.  For example, if
	// two instances of Conn are available at runtime (as in the case of tests in the sandbox),
	// one can be named "client" and the other "server". The System string is also kept as a
	// field inside the Logger object, so that it can be filled in automatically upon calls to
	// the Logger's E method.
	System    string  `json:"y"`

	// Module is a compile-time identifier of some logical subset of a DCCP system. For
	// instance, 's' can identify the sender congestion control logic, whereas 's-rtt' can
	// identify spcifically the roundtrip time estimation logic inside the sender congestion
	// control. Module is passed as a parameter to the Logger's E method, and is, ideally, a
	// string constant.
	Module    string  `json:"m"`

	// Event is an identifier representing the type of event that this log record represents. It
	// can be something like "Warn", "Info", etc.
	Event     string  `json:"e"`

	// If applicable, State is the DCCP state of the runtime instance (or system) that this log
	// record pertains to. This is typically used only if the system is a dccp.Conn.
	State     string  `json:"s"`

	// Comment is a free-form textual comment
	Comment   string  `json:"c"`

	// Args are any additional arguments in the form of string keys mapped to open-ended values
	Args      LogArgs `json:"a"`

	// If this log record pertains to a DCCP header, Type is the DCCP type of this header.
	Type      string  `json:"ht"`

	// If this log record pertains to a DCCP header, SeqNo is the DCCP sequence number of this header.
	SeqNo     int64   `json:"hs"`

	// If this log record pertains to a DCCP header, AckNo is the DCCP acknowledgement number of
	// this header.
	AckNo     int64   `json:"ha"`

	// SourceFile is the name of the source file where this log record was emitted.
	SourceFile string `json:"sf"`

	// SourceLine is the line number in the source file where this log record was emitted.
	SourceLine int    `json:"sl"`

	// Trace is the stack trace at the log entry's creation point
	Trace      string `json:"st"`
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
	system string
}

// A zero-value Logger ignores all emits
var NoLogging *Logger = &Logger{}

func NewLogger(system string, run *Runtime) *Logger {
	return &Logger{ run: run, system: system }
}

func (t *Logger) System() string {
	return t.system
}

func (t *Logger) Filter() *filter.Filter {
	return t.run.Filter()
}

func (t *Logger) GetState() string {
	if t.run == nil {
		return ""
	}
	g := t.run.Filter().GetAttr([]string{t.System()}, "state")
	if g == nil {
		return ""
	}
	return g.(string)
}

func (t *Logger) SetState(s int) {
	if t.run == nil {
		return
	}
	t.run.Filter().SetAttr([]string{t.System()}, "state", StateString(s))
}

// StackTrace formats the stack trace of the calling go routine, 
// excluding pointer information and including DCCP runtime-specific information, 
// in a manner convenient for debugging DCCP
func stackTrace(system, module string, skip int) string {
	var w bytes.Buffer
	var stk []uintptr = make([]uintptr, 32)	// DCCP logic stack should not be deeper than that
	n := runtime.Callers(skip+1, stk)
	stk = stk[:n]
	fmt.Fprintf(&w, "%s:%s\n", system, module)
	var nondccp bool
	for _, pc := range stk {
		f := runtime.FuncForPC(pc)
		if f == nil {
			break
		}
		file, line := f.FileLine(pc)
		fname, isdccp := TrimFuncName(f.Name())
		if !isdccp {
			nondccp = true
		} else {
			if nondccp {
				fmt.Fprintf(&w, "    ···· ···· ···· \n")
			}
			fmt.Fprintf(&w, "    %-40s (%s:%d)\n", fname, TrimSourceFile(file), line)
		}
	}
	return string(w.Bytes())
}

// E emits a new log record. The arguments args are scanned in turn. The first argument of
// type *Header, *PreHeader, *FeedbackHeader or *FeedforwardHeader is considered the DCCP
// header that this log pertains to. The first argument of type Args is saved in the log
// record.
func (t *Logger) E(module, event, comment string, args ...interface{}) {
	t.EC(1, module, event, comment, args...)
}

func (t *Logger) EC(skip int, module, event, comment string, args ...interface{}) {
	if t.run == nil {
		return
	}
	/*
	if !t.run.Filter().Selected(t.System(), module) {
		return
	}
	*/
	sinceZero, sinceLast := t.run.Snap()

	// Extract header information
	var hType string = ""
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

	sfile, sline := FetchCaller(1+skip)

	if t.run.Writer() != nil {
		r := &LogRecord{
			Time:       sinceZero,
			System:     t.System(),
			Module:     module,
			Event:      event,
			State:      t.GetState(),
			Comment:    comment,
			Args:       largs,
			Type:       hType,
			SeqNo:      hSeqNo,
			AckNo:      hAckNo,
			SourceFile: sfile,
			SourceLine: sline,
			Trace:      stackTrace(t.System(), module, skip+2),
		}
		t.run.Writer().Write(r)
	}
	// Print log messages to os.Stdout if environment variable DCCPRAW set
	if os.Getenv("DCCPRAW") != "" {
		fmt.Printf("%15s %15s %18s:%-3d %-8s %6s:%-9s %-9s %8s %6x|%-6x * %s\n", 
			Nstoa(sinceZero), Nstoa(sinceLast), 
			sfile, sline,
			t.GetState(), t.System(), 
			module, event, 
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

// NewFileLogWriterDup creates a LogWriter that saves logs in a file and also passes them to dup.
func NewFileLogWriterDup(filename string, dup LogWriter) *FileLogWriter {
	os.Remove(filename)
	f, err := os.Create(filename)
	if err != nil {
		panic(fmt.Sprintf("cannot create log file '%s'", filename))
	}
	w := &FileLogWriter{ f, json.NewEncoder(f), dup }
	goruntime.SetFinalizer(w, func(w *FileLogWriter) { 
		w.f.Close() 
	})
	return w
}

func NewFileLogWriter(filename string) *FileLogWriter {
	return NewFileLogWriterDup(filename, nil)
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
