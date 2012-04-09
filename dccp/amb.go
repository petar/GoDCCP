// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
	"runtime"
	"reflect"
	"github.com/petar/GoGauge/filter"
)

// Logger is capable of emitting structured logs, which are consequently used for debuging
// and analysis purposes. It lives in the context of a shared time framework and a shared
// filter framework, which may filter some logs out
type Logger struct {
	run    *Runtime
	labels []string
}

// A zero-value Logger has the special-case behavior of ignoring all emits
var NoLogging *Logger = &Logger{}

// NewLogger creates a new Logger object with a single entry in the label stack
func NewLogger(label string, run *Runtime) *Logger {
	return &Logger{ run: run, labels: []string{label} }
}

// Refine clones this logger and stack the additional label l
func (t *Logger) Refine(l string) *Logger {
	return t.Copy().Push(l)
}

// Copy clones this logger into an identical new one
func (t *Logger) Copy() *Logger {
	var c Logger = *t
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
func stackTrace(labels []string, skip int, sfile string, sline int) string {
	var w bytes.Buffer
	var stk []uintptr = make([]uintptr, 32)	// DCCP logic stack should not be deeper than that
	n := runtime.Callers(skip+1, stk)
	stk = stk[:n]
	var utf2byte int
	for _, l := range labels {
		fmt.Fprintf(&w, "%s·", l)
		utf2byte++
	}
	for w.Len() < 40 + 4 + utf2byte {
		w.WriteRune(' ')
	}
	fmt.Fprintf(&w, " (%s:%d)\n", sfile, sline)
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
func (t *Logger) E(event Event, comment string, args ...interface{}) {
	t.EC(1, event, comment, args...)
}

func (t *Logger) EC(skip int, event Event, comment string, args ...interface{}) {
	if t.run == nil {
		return
	}
	sinceZero, _ := t.run.Snap()

	// Extract header information
	var hType string = ""
	var hSeqNo, hAckNo int64
	logargs := make(map[string]interface{})
	for _, a := range args {
		switch t := a.(type) {
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
		// By default, take the argument's type and use it as a key in the arguments structure
		default:
			if a != nil {
				logargs[TypeOf(a)] = a
			}
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
			Args:       logargs,
			Type:       hType,
			SeqNo:      hSeqNo,
			AckNo:      hAckNo,
			SourceFile: sfile,
			SourceLine: sline,
			Trace:      stackTrace(t.labels, skip+2, sfile, sline),
		}
		t.run.Writer().Write(r)
	}
}

func TypeOf(a interface{}) string {
	t := reflect.TypeOf(a)
	// Remove the '*' from pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.String()
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
