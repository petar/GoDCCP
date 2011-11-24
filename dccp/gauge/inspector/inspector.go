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
	flagFmt *string = flag.String("fmt", "basic", "Format: basic, trip")
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

	switch *flagFmt {
	case "basic":
		printBasic(log)
	case "trip":
		printTrip(log)
	}

	printStats(log)
}

var (
	printTripSep = &PrintRecord{
		Text: 
			"——————————————————————————————————————————————" +
			"——————————————————————————————————————————————" +
			"———————————————————————————————",
	}
	printHalveSep = &PrintRecord{
		Text: fmt.Sprintf("%s=%s=%s=%s=%s", skipState, skip, skip, skip, skipState),
	}
	printNop = &PrintRecord{}
)

func printTrip(log []*dccp.LogRecord) {
	reducer := dccp_gauge.NewLogReducer()
	for _, rec := range log {
		reducer.Write(rec)
	}
	trips := dccp_gauge.TripMapToSlice(reducer.Trips())
	sort.Sort(TripSeqNoSort(trips))
	
	prints := make([]*PrintRecord, 0)
	for _, t := range trips {
		prints = append(prints, printNop)
		for _, r := range t.Forward {
			p := printRecord(r)
			if p != nil {
				prints = append(prints, p)
			}
		}
		prints = append(prints, printHalveSep)
		for _, r := range t.Backward {
			p := printRecord(r)
			if p != nil {
				prints = append(prints, p)
			}
		}
		prints = append(prints, printNop)
		prints = append(prints, printTripSep)
	}
	Print(prints, false)
}

func printStats(log []*dccp.LogRecord) {
	reducer := dccp_gauge.NewLogReducer()
	for _, rec := range log {
		reducer.Write(rec)
	}
	trips := dccp_gauge.TripMapToSlice(reducer.Trips())
	sort.Sort(TripSeqNoSort(trips))
	sr, rr := dccp_gauge.CalcRates(trips)
	fmt.Printf("Send rate: %g pkt/sec, Receive rate: %g pkt/sec\n", sr, rr)
}

func printBasic(log []*dccp.LogRecord) {
	prints := make([]*PrintRecord, 0)
	for _, t := range log {
		var p *PrintRecord = printRecord(t)
		if p != nil {
			prints = append(prints, p)
		}
	}
	Print(prints, true)
}

func printRecord(t *dccp.LogRecord) *PrintRecord {
	switch t.Event {
	case "Write":
		return printWrite(t)
	case "Read":
		return printRead(t)
	case "Drop":
		return printDrop(t)
	case "Idle":
		return printIdle(t)
	default:
		return printGeneric(t)
	}
	panic("unreach")
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
		text = fmt.Sprintf("%8s | %s |%s|%s|%s",
			r.State, sprintPacketEventComment(r), skip, skip, skipState)
	case "server":
		text = fmt.Sprintf("%s|%s|%s| %s | %-8s",
			skipState, skip, skip, sprintPacketEventComment(r), r.State)
	case "line":
		text = fmt.Sprintf("%s|%s| %s |%s|%s",
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
	return fmt.Sprintf(" %9s%06x/%06x ", string(w.Bytes()), r.SeqNo, r.AckNo)
}

func sprintPacketEventComment(r *dccp.LogRecord) string {
	if r.SeqNo == 0 {
		return fmt.Sprintf("     %-22s     ", cut(r.Comment, 22))
	}
	return fmt.Sprintf("     %-14s %06x/     ", cut(r.Comment, 14), r.SeqNo)
}

func cut(s string, n int) string {
	if n >= len(s) {
		return s
	}
	return s[:n]
}

// Print orders a sequence of print recors by time and prints them to standard output
func Print(records []*PrintRecord, srt bool) {
	if srt {
		sort.Sort(PrintTimeSort(records))
	}
	var last int64
	var sec  int64
	var sflag rune = ' '
	for _, r := range records {
		if r.Log != nil {
			fmt.Printf("%15s %c  %s   %18s:%-3d\n", 
				dccp.Nstoa(r.Log.Time - last), sflag, r.Text, r.Log.SourceFile, r.Log.SourceLine)
			sflag = ' '
			last = r.Log.Time
			if last / 1e9 > sec {
				sflag = '*'
				sec = last / 1e9
			}
		} else {
			fmt.Printf("                   %s\n", r.Text)
			sflag = ' '
			last = 0
		}
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
