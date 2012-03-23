// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

func usage() {
	fmt.Printf("%s [optional_flags] log_file\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Parse()

	// First non-flag argument is log file name
	nonflags := flag.Args()
	if len(nonflags) == 0 {
		usage()
	}

	// Open and decode log file
	logFile, err := os.Open(nonflags[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log (%s)\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	logDec := json.NewDecoder(logFile)

	// Raw log entries will go into emits
	var emits []*dccp.LogRecord = make([]*dccp.LogRecord, 0)
	for {
		rec := &dccp.LogRecord{}
		if err = logDec.Decode(rec); err != nil {
			break
		}
		emits = append(emits, rec)
	}
	fmt.Fprintf(os.Stderr, "Read %d records.\n", len(emits))
	if err != io.EOF {
		fmt.Fprintf(os.Stderr, "Terminated unexpectedly (%s).\n", err)
	}

	// Fork to desired reducer
	switch *flagFmt {
	case "basic":
		printBasic(emits)
	case "trip":
		printTrip(emits)
	}

	printStats(emits)
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

func printTrip(emits []*dccp.LogRecord) {
	reducer := dccp_gauge.NewLogReducer()
	for _, rec := range emits {
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

func printStats(emits []*dccp.LogRecord) {
	reducer := dccp_gauge.NewLogReducer()
	for _, rec := range emits {
		reducer.Write(rec)
	}
	trips := dccp_gauge.TripMapToSlice(reducer.Trips())
	sort.Sort(TripSeqNoSort(trips))
	sr, rr := dccp_gauge.CalcRates(trips)
	fmt.Printf("Send rate: %g pkt/sec, Receive rate: %g pkt/sec\n", sr, rr)
}

func printBasic(emits []*dccp.LogRecord) {
	prints := make([]*PrintRecord, 0)
	for _, t := range emits {
		var p *PrintRecord = printRecord(t)
		if p != nil {
			prints = append(prints, p)
		}
	}
	Print(prints, true)
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
