// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"flag"
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
		if logDec.Decode(rec) != nil {
			break
		}
		log = append(log, rec)
	}
	fmt.Printf("Read %d records\n", len(log))

	// Basic
	if *flagBasic {
		printBasic(log)
	}
}

func printBasic(log []*dccp.LogRecord) {
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
			prints = append(prints, p)
		}
	}
	Print(prints)
}

// MakeOneWayTripPrint prepares a print record for a one-way packet
// trip that reaches the destination. Otherwise, it returns nil.
func MakeOneWayTripPrint(t *dccp_gauge.Trip) *PrintRecord {
	var start *dccp.LogRecord
	for _, s := range t.Forward {
		if s.Event == "Write" && (s.Module == "client" || s.Module == "server") {
			start = s
			break
		}
	}
	if start == nil {
		fmt.Printf("Trip with no start\n")
		return nil
	}
	var end *dccp.LogRecord
	for i := 0; i < len(t.Forward); i++ {
		s := t.Forward[len(t.Forward)-1-i]
		if s.Event == "Read" && (s.Module == "client" || s.Module == "server") {
			end = s
			break
		}
	}
	if end == nil {
		return nil
	}
	if start.Type != end.Type {
		fmt.Printf("Start and end types differ\n")
		return nil
	}
	var text string
	switch {
	case start.Module == "client" && end.Module == "server":
		text = fmt.Sprintf("%15s:%-8s ---- %8s %8d (%8d) ---> %-8s:%15s",
			dccp.Nstoa(start.Time), start.State,
			start.Type, start.SeqNo, start.AckNo,
			end.State, dccp.Nstoa(end.Time),
		)
	case start.Module == "server" && end.Module == "client":
		text = fmt.Sprintf("%15s:%-8s <--- (%8d) %8d %8s ---- %-8s:%15s",
			dccp.Nstoa(end.Time), end.State,
			start.AckNo, start.SeqNo, start.Type,
			start.State, dccp.Nstoa(start.Time),
		)
	default:
		fmt.Printf("Start and end modules are the same\n")
		return nil
	}
	return &PrintRecord{
		Time: start.Time,
		Text: text,
	}
}

// Print orders a sequence of print recors by time and prints them to standard output
func Print(records []*PrintRecord) {
	sort.Sort(PrintTimeSort(records))
	for _, r := range records {
		fmt.Printf("%15s %s\n", dccp.Nstoa(r.Time), r.Text)
	}
}

type PrintRecord struct {
	Time int64
	Text string
}

// PrintTimeSort sorts print records by timestamp
type PrintTimeSort []*PrintRecord

func (t PrintTimeSort) Len() int {
	return len(t)
}

func (t PrintTimeSort) Less(i, j int) bool {
	return t[i].Time < t[j].Time
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
