package main

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"text/template"
	"github.com/petar/GoDCCP/dccp"
)

// SeriesSweeper consumes log records in time order and outputs
// the embedded time series data in JavaScript array format that 
// can be fed into for dygraph.
type SeriesSweeper struct {
	series []string
	chrono list.List
}

type sample struct {
	Series string
	Time   float64
	Value  float64
}

// Init resets the SeriesSweeper for new user.
func (x *SeriesSweeper) Init() {
	x.series = make([]string, 0)
	x.chrono.Init()
}

// Add adds a new log record to the series. It assumes that records are added
// in increasing chronological order
func (x *SeriesSweeper) Add(r *dccp.Trace) {
	if !r.IsHighlighted() {
		return
	}
	// Check that the argument is a sample
	m_, ok := r.Args[dccp.SampleType]
	if !ok {
		return
	}
	// Read sample data
	m := m_.(map[string]interface{})
	value := m["Value"].(float64)
	series := r.LabelString() + m["Series"].(string)
	for _, u := range x.series {
		if u == series {
			goto __SeriesSaved
		}
	}
	x.series = append(x.series, series)
__SeriesSaved:
	// Remember the sample
	u := &sample{
		Series: series,
		Time:   float64(r.Time) / 1e6,	// Time, X-coordinate, always in milliseconds
		Value:  value,
	}
	x.chrono.PushBack(u)
}

// EncodeData encodes the entire data received so far into a JavaScript array format.
func (x *SeriesSweeper) EncodeData(w io.Writer) error {
	var bw *bufio.Writer = bufio.NewWriter(w)
	bw.WriteByte(byte('['))
	for e := x.chrono.Front(); e != nil; e = e.Next() {
		s := e.Value.(*sample)
		bw.WriteByte(byte('['))
		fmt.Fprintf(bw, "%0.3f,", s.Time)
		for i, series := range x.series {
			if series == s.Series {
				fmt.Fprintf(bw, "%0.3f", s.Value)
			} else {
				bw.WriteString("null")
			}
			if i < len(x.series)-1 {
				bw.WriteByte(byte(','))
			}
		}
		bw.WriteByte(']')
		if e.Next() != nil {
			bw.WriteByte(byte(','))
		}
		bw.WriteByte('\n')
	}
	bw.WriteByte(byte(']'))
	return bw.Flush()
}

// EncodeHeader encodes the names of the time series in a JavaScript array format.
func (x *SeriesSweeper) EncodeHeader(w io.Writer) error {
	var bw *bufio.Writer = bufio.NewWriter(w)
	bw.WriteString("[\"Time\",")
	for i, series := range x.series {
		bw.WriteByte(byte('"'))
		template.JSEscape(bw, []byte(series))
		bw.WriteByte(byte('"'))
		if i < len(x.series)-1 {
			bw.WriteByte(byte(','))
		}
	}
	bw.WriteByte(byte(']'))
	return bw.Flush()
}
