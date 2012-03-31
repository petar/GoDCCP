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

	// Time is the DCCP runtime time when the log was emitted
	Time      int64   `json:"t"`

	// Labels is a list of runtime labels that identify some dynamic
	// instance of the DCCP stack.  For example, if two instances of Conn
	// are available at runtime (as in the case of tests in the sandbox),
	// one can be labeled "client" and the other "server". The labels slice
	// is kept as a field inside the Logger object, so that it can be
	// filled in automatically upon calls to the Logger's E method.
	Labels    []string `json:"l"`

	// Event is an identifier representing the type of event that this log record represents. It
	// can be something like "Warn", "Info", etc.
	Event     string   `json:"e"`

	// If applicable, State is the DCCP state of the runtime instance (or system) that this log
	// record pertains to. This is typically used only if the system is a dccp.Conn.
	State     string   `json:"s"`

	// Comment is a free-form textual comment
	Comment   string   `json:"c"`

	// Args are any additional arguments in the form of string keys mapped to open-ended values
	Args      LogArgs  `json:"a"`

	// If this log record pertains to a DCCP header, Type is the DCCP type of this header.
	Type      string   `json:"ht"`

	// If this log record pertains to a DCCP header, SeqNo is the DCCP sequence number of this header.
	SeqNo     int64    `json:"hs"`

	// If this log record pertains to a DCCP header, AckNo is the DCCP acknowledgement number of
	// this header.
	AckNo     int64    `json:"ha"`

	// SourceFile is the name of the source file where this log record was emitted.
	SourceFile string  `json:"sf"`

	// SourceLine is the line number in the source file where this log record was emitted.
	SourceLine int     `json:"sl"`

	// Trace is the stack trace at the log entry's creation point
	Trace      string  `json:"st"`
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
	run    *Runtime
	labels []string
}

// A zero-value Logger has the special-case behavior of ignoring all emits
var NoLogging *Logger = &Logger{}

// NewLogger creates a new Logger object with an empty label stack
func NewLogger(system string, run *Runtime) *Logger {
	return &Logger{ run: run, labels: make([]string, 0) }
}

// Refine clones this logger and stack the additional label l
func (t *Logger) Refine(l string) *Logger {
	return t.Copy().Push(l)
}

// Copy clones this logger into an identical new one
func (t *Logger) Copy() *Logger {
	c = *t
	c.labels = make([]string, len(t.labels))
	copy(c.labels, t.labels)
	return &c
}

// Labels returns the label stack of this logger
func (t *Logger) Labels() []string {
	return t.labels
}

// Push adds the label l onto this logger's label stack
func (t *Logger) Push(l string) *Logger {
	t.labels = append(t.labels, l)
	return t
}

func (t *Logger) Filter() *filter.Filter {
	return t.run.Filter()
}

// GetState retrieves the state of the owning object, using the runtime value store
func (t *Logger) GetState() string {
	if t.run == nil || len(t.labels) == 0 {
		return ""
	}
	g := t.run.Filter().GetAttr([]string{t.labels[0]}, "state")
	if g == nil {
		return ""
	}
	return g.(string)
}

// SetState saves the state s into the runtime value store
func (t *Logger) SetState(s int) {
	if t.run == nil {
		return
	}
	t.run.Filter().SetAttr([]string{t.labels[0]}, "state", StateString(s))
}

// StackTrace formats the stack trace of the calling go routine, 
// excluding pointer information and including DCCP runtime-specific information, 
// in a manner convenient for debugging DCCP
func stackTrace(labels []string, skip int) string {
	var w bytes.Buffer
	var stk []uintptr = make([]uintptr, 32)	// DCCP logic stack should not be deeper than that
	n := runtime.Callers(skip+1, stk)
	stk = stk[:n]
	for _, l := range labels {
		fmt.Fprintf(&w, "%s·", l)
	}
	fmt.Fprintf(&w, "\n")
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
func (t *Logger) E(event, comment string, args ...interface{}) {
	t.EC(1, event, comment, args...)
}

func (t *Logger) EC(skip int, event, comment string, args ...interface{}) {
	if t.run == nil {
		return
	}
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
			Labels:     t.labels,
			Event:      event,
			State:      t.GetState(),
			Comment:    comment,
			Args:       largs,
			Type:       hType,
			SeqNo:      hSeqNo,
			AckNo:      hAckNo,
			SourceFile: sfile,
			SourceLine: sline,
			Trace:      stackTrace(t.labels, skip+2),
		}
		t.run.Writer().Write(r)
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
