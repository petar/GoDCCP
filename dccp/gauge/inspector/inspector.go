// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"flag"
	"io"
	"os"
	"sort"
	//"github.com/petar/GoGauge/gauge"
	"github.com/petar/GoDCCP/dccp"
	dccp_gauge "github.com/petar/GoDCCP/dccp/gauge"
)

var (
	flagBasic *bool = flag.Bool("basic", true, "Basic format")
)

func main() {
	flag.Parse()

	// First non-flag argument is log file name
	nonflags := flag.Args()
	if len(nonflags) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	logFile, err := os.Open(nonflags[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log (%s)\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	logDec := json.NewDecoder(logFile)
	var log []*dccp.LogRecord = make([]*dccp.LogRecord, 0)
	for {
		rec := &dccp.LogRecord{}
		if err = logDec.Decode(rec); err != nil {
			break
		}
		log = append(log, rec)
	}
	fmt.Fprintf(os.Stderr, "Read %d records.\n", len(log))
	if err != io.EOF {
		fmt.Fprintf(os.Stderr, "Terminated unexpectedly (%s).\n", err)
	}

	// Basic
	if *flagBasic {
		printBasic(log)
	}
}

/*
func printTrips(log []*dccp.LogRecord) {
	reducer := dccp_gauge.NewLogReducer()
	for _, rec := range log {
		reducer.Write(rec)
	}
	trips := dccp_gauge.TripMapToSlice(reducer.Trips())
	sort.Sort(TripSeqNoSort(trips))
	
	prints := make([]*PrintRecord, 0)
	for _, t := range trips {
		p := MakeOneWayTripPrint(t)
		if p != nil {
			prints = append(prints, p...)
		}
	}
	Print(prints)
}

// MakeOneWayTripPrint prepares a print record for a one-way packet
// trip that reaches the destination. Otherwise, it returns nil.
func MakeOneWayTripPrint(t *dccp_gauge.Trip) []*PrintRecord {
	// Find write record
	var start *dccp.LogRecord
	for _, s := range t.Forward {
		if s.Event == "Write" && (s.Module == "client" || s.Module == "server") {
			start = s
			break
		}
	}
	if start == nil || start.Module == "line" {
		fmt.Printf("Trip with no start, seqno=%06x\n", t.SeqNo)
		return nil
	}
	// Find drop record
	var drop *dccp.LogRecord
	for _, s := range t.Forward {
		if s.Event == "Drop" {
			drop = s
			break
		}
	}
	// Find read record
	var end *dccp.LogRecord
	for i := 0; i < len(t.Forward); i++ {
		s := t.Forward[len(t.Forward)-1-i]
		if s.Event == "Read" && (s.Module == "client" || s.Module == "server") {
			end = s
			break
		}
	}
	...
}
*/

func printBasic(log []*dccp.LogRecord) {
	prints := make([]*PrintRecord, 0)
	for _, t := range log {
		var p *PrintRecord
		switch t.Event {
		case "Write":
			p = printWrite(t)
		case "Read":
			p = printRead(t)
		case "Drop":
			p = printDrop(t)
		case "Idle":
			p = printIdle(t)
		default:
			p = printGeneric(t)
		}
		if p != nil {
			prints = append(prints, p)
		}
	}
	Print(prints)
}

const (
	skip = "                                  "
	skipState = "         "
)

func printWrite(r *dccp.LogRecord) *PrintRecord {
	switch r.Module {
	case "server":
		return &PrintRecord{
			Log:  r,
			Text: fmt.Sprintf("%s|%s|%s|     %s<——W | %-8s", 
				skipState, skip, skip, sprintPacket(r), r.State),
		}
	case "client":
		return &PrintRecord{
			Log:  r,
			Text: fmt.Sprintf("%8s | W——>%s     |%s|%s|%s", 
				r.State, sprintPacket(r), skip, skip, skipState),
		}
	}
	return nil
}

func printRead(r *dccp.LogRecord) *PrintRecord {
	switch r.Module {
	case "client":
		return &PrintRecord{
			Log:  r,
			Text: fmt.Sprintf("%8s | R<——%s     |%s|%s|%s", 
				r.State, sprintPacket(r), skip, skip, skipState),
		}
	case "server":
		return &PrintRecord{
			Log:  r,
			Text: fmt.Sprintf("%s|%s|%s|     %s——>R | %-8s", 
				skipState, skip, skip, sprintPacket(r), r.State),
		}
	}
	return nil
}

func printDrop(r *dccp.LogRecord) *PrintRecord {
	var text string
	switch r.Module {
	case "line":
		if r.Submodule == "server" {
			text = fmt.Sprintf("%s|%s| D<——%s     |%s|%s",
				skipState, skip, sprintPacket(r), skip, skipState)
		} else {
			text = fmt.Sprintf("%s|%s|     %s——>D |%s|%s",
				skipState, skip, sprintPacket(r), skip, skipState)
		}
	case "client":
		switch r.Comment {
		case "Slow app":
			text = fmt.Sprintf("%8s |     %sD<—— |%s|%s|%s",
				r.State, sprintPacket(r), skip, skip, skipState)
		case "Slow strobe":
			text = fmt.Sprintf("%8s |     %s——>D |%s|%s|%s",
				r.State, sprintPacket(r), skip, skip, skipState)
		}
	case "server":
		switch r.Comment {
		case "Slow strobe":
			text = fmt.Sprintf("%s|%s|%s| D<——%s     | %-8s",
				skipState, skip, skip, sprintPacket(r), r.State)
		case "Slow app":
			text = fmt.Sprintf("%s|%s|%s|     %s——>D | %-8s",
				skipState, skip, skip, sprintPacket(r), r.State)
		}
	}
	if text == "" {
		return nil
	}
	return &PrintRecord{
		Log:  r,
		Text: text,
	}
}

func printIdle(r *dccp.LogRecord) *PrintRecord {
	var text string
	switch r.Module {
	case "client":
		text = fmt.Sprintf("%8s |—%s—|%s|%s|%s",
			r.State, sprintIdle(r), skip, skip, skipState)
	case "server":
		text = fmt.Sprintf("%s|%s|%s|—%s—| %-8s",
			skipState, skip, skip, sprintIdle(r), r.State)
	}
	if text == "" {
		return nil
	}
	return &PrintRecord{
		Log:  r,
		Text: text,
	}
}

func printGeneric(r *dccp.LogRecord) *PrintRecord {
	var text string
	switch r.Module {
	case "client":
		text = fmt.Sprintf("%8s |-%s-|%s|%s|%s",
			r.State, sprintPacketEventComment(r), skip, skip, skipState)
	case "server":
		text = fmt.Sprintf("%s|%s|%s|-%s-| %-8s",
			skipState, skip, skip, sprintPacketEventComment(r), r.State)
	case "line":
		text = fmt.Sprintf("%s|%s|-%s-|%s|%s",
			skipState, skip, sprintPacketEventComment(r), skip, skipState)
	}
	if text == "" {
		return nil
	}
	return &PrintRecord{
		Log:  r,
		Text: text,
	}
}

func sprintIdle(r *dccp.LogRecord) string {
	return "————————————————————————————————"
}

func sprintPacket(r *dccp.LogRecord) string {
	var w bytes.Buffer
	w.WriteString(r.Type)
	for i := 0; i < 9-len(r.Type); i++ {
		w.WriteRune('·')
	}
	return fmt.Sprintf(" %9s%06x·%06x ", string(w.Bytes()), r.SeqNo, r.AckNo)
}

func sprintPacketEventComment(r *dccp.LogRecord) string {
	var w bytes.Buffer
	fmt.Fprintf(&w, "-----%4s:%-17s-----", cut(r.Event, 4), cut(r.Comment, 17))
	//var p = 32 - w.Len()
	//for i := 0; i < p; i++ {
	//	w.WriteRune('·')
	//}
	return string(w.Bytes())
}

func cut(s string, n int) string {
	if n >= len(s) {
		return s
	}
	return s[:n]
}

// Print orders a sequence of print recors by time and prints them to standard output
func Print(records []*PrintRecord) {
	sort.Sort(PrintTimeSort(records))
	var last int64
	for _, r := range records {
		fmt.Printf("%15s   %s   %18s:%-3d\n", 
			dccp.Nstoa(r.Log.Time - last), r.Text, r.Log.SourceFile, r.Log.SourceLine)
		last = r.Log.Time
	}
}

type PrintRecord struct {
	Log  *dccp.LogRecord
	Text string
}

// PrintTimeSort sorts print records by timestamp
type PrintTimeSort []*PrintRecord

func (t PrintTimeSort) Len() int {
	return len(t)
}

func (t PrintTimeSort) Less(i, j int) bool {
	return t[i].Log.Time < t[j].Log.Time
}

func (t PrintTimeSort) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// TripSeqNoSort sorts an array of trips by sequence number
type TripSeqNoSort []*dccp_gauge.Trip

func (t TripSeqNoSort) Len() int {
	return len(t)
}

func (t TripSeqNoSort) Less(i, j int) bool {
	return t[i].SeqNo < t[j].SeqNo
}

func (t TripSeqNoSort) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
