// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
)

// Trace stores a log event. It can be used to marshal to JSON and pass to external
// visualisation tools.
type Trace struct {

	// Time is the DCCP runtime time when the log was emitted
	Time      int64   `json:"t"`

	// Labels is a list of runtime labels that identify some dynamic
	// instance of the DCCP stack.  For example, if two instances of Conn
	// are available at runtime (as in the case of tests in the sandbox),
	// one can be labeled "client" and the other "server". The labels slice
	// is kept as a field inside the Amb object, so that it can be
	// filled in automatically upon calls to the Amb's E method.
	Labels    []string `json:"l"`

	// Event is an identifier representing the type of event that this trace represents. It
	// can be something like "Warn", "Info", etc.
	Event     Event   `json:"e"`

	// If applicable, State is the DCCP state of the runtime instance (or system) that this log
	// record pertains to. This is typically used only if the system is a dccp.Conn.
	State     string   `json:"s"`

	// Comment is a free-form textual comment
	Comment   string   `json:"c"`

	// Args are any additional arguments in the form of string keys mapped to open-ended values.
	// See documentation of E method for details how it is typically used.
	Args      map[string]interface{}  `json:"a"`

	// If this trace pertains to a DCCP header, Type is the DCCP type of this header.
	Type      string   `json:"ht"`

	// If this trace pertains to a DCCP header, SeqNo is the DCCP sequence number of this header.
	SeqNo     int64    `json:"hs"`

	// If this trace pertains to a DCCP header, AckNo is the DCCP acknowledgement number of
	// this header.
	AckNo     int64    `json:"ha"`

	// SourceFile is the name of the source file where this trace was emitted.
	SourceFile string  `json:"sf"`

	// SourceLine is the line number in the source file where this trace was emitted.
	SourceLine int     `json:"sl"`

	// Trace is the stack trace at the log entry's creation point
	Trace      string  `json:"st"`

	// Highlight indicates whether this record is of particular interest. Used for visualization purposes.
	// Currently, the inspector draws time series only for highlighted records.
	Highlight  bool
}

// LabelString returns a textual representation of the label stack of this log
func (x *Trace) LabelString() string {
	return labelString(x.Labels)
}

func labelString(labels []string) string {
	var w bytes.Buffer
	for _, token := range labels {
		w.WriteString(token)
		w.WriteRune('·')
	}
	return string(w.Bytes())
}

// ArgOfType returns an argument of the same type as example, if one is present
// in the log, or nil otherwise
func (x *Trace) ArgOfType(example interface{}) interface{} {
	a, ok := x.Args[TypeOf(example)]
	if !ok {
		return nil
	}
	return a
}

// Sample returns the value of a sample embedded in this trace, if it exists
func (x *Trace) Sample() (sample *Sample, present bool) {
	s_, ok := x.Args[TypeOf(Sample{})]
	if !ok {
		return nil, false
	}
	s := s_.(Sample)
	return &s, true
}

// Highlight sets the highlight flag on this Trace. This is used in test-specific
// Guzzles to indicate to the inspector that this record is of particular interest for
// visualization purposes.
func (x *Trace) SetHighlight() {
	x.Highlight = true
}

func (x *Trace) IsHighlighted() bool {
	return x.Highlight
}

// One Sample argument can be attached to a log. The inspector interprets it as a data point
// in a time series where: 
//   (i)   The time series name is given by the label stack of the amb
//   (ii)  The X-value of the data point equals the time the log was emitted
//   (iii) The Y-value of the data point is stored inside the Sample object
type Sample struct {
	Series string
	Value  float64
	Unit   string
}
var SampleType = TypeOf(Sample{})

func NewSample(name string, value float64, unit string) Sample {
	return Sample{name, value, unit}
}

// Event is the type of logging event.
// Events are wrapped in a special type to make sure that modifications/additions
// to the set of events impose respective modifications in the reducer and the inspector.
type Event int

const (
	EventTurn = Event(iota) // Turn events mark new information that is admissible (within expectations)
	EventMatch              // Match events mark an admissible outcome of a complex event/computation sequence
	EventCatch              // Catch events are breakpoints on conditions
	EventInfo
	EventWarn
	EventError
	EventIdle               // Idle events are emitted on the turn of the DCCP idle loop
	EventDrop               // Drop events are related to a packet drop
	EventRead               // Read events are related to a packet read
	EventWrite              // Write events are related to a packet write
)

// String returns a textual representation of the event
func (e Event) String() string {
	switch e {
	case EventTurn:
		return "Turn"
	case EventMatch:
		return "Match"
	case EventCatch:
		return "Catch"
	case EventInfo:
		return "Info"
	case EventWarn:
		return "Warn"
	case EventError:
		return "Error"
	case EventIdle:
		return "Idle"
	case EventDrop:
		return "Drop"
	case EventRead:
		return "Read"
	case EventWrite:
		return "Write"
	}
	panic("unknown event")
}

func indentEvent(event Event) string {
	s := event.String()
	switch s {
	case "Write", "Read", "Strobe":
		return s
	}
	return "  " + s
}
