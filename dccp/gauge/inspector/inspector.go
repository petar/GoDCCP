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
		fmt.Printf("Trip with no start\n")
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
	// Make print records
	var rightleft bool
	if start.Module == "server" {
		rightleft = true
	}
	var ps []*PrintRecord = make([]*PrintRecord, 3)
	var pi int
	const flushRight = "                                                                                    "
	const flushMiddle = "                                       "
	// Print start record
	if rightleft {
		ps[pi] = &PrintRecord{}
		ps[pi].Time = start.Time
		pkt := fmt.Sprintf(" %8s %06x·%06x ", start.Type, start.SeqNo, start.AckNo)
		ps[pi].Text = fmt.Sprintf("%15s %s <———%s<——— %-8s", dccp.Nstoa(start.Time), flushRight, pkt, start.State)
		pi++
	} else {
		ps[pi] = &PrintRecord{}
		ps[pi].Time = start.Time
		pkt := fmt.Sprintf(" %8s %06x·%06x ", start.Type, start.SeqNo, start.AckNo)
		ps[pi].Text = fmt.Sprintf("%15s %-8s ———>%s———>", dccp.Nstoa(start.Time), start.State, pkt)
		pi++
	}
	// Print end record
	if end != nil {
		if rightleft {
			ps[pi] = &PrintRecord{}
			ps[pi].Time = end.Time
			pkt := fmt.Sprintf(" %8s %06x·%06x ", end.Type, end.SeqNo, end.AckNo)
			ps[pi].Text = fmt.Sprintf("%15s %-8s <———%s<———", dccp.Nstoa(end.Time), end.State, pkt)
			pi++
		} else {
			ps[pi] = &PrintRecord{}
			ps[pi].Time = end.Time
			pkt := fmt.Sprintf(" %8s %06x·%06x ", end.Type, end.SeqNo, end.AckNo)
			ps[pi].Text = fmt.Sprintf("%15s %s ———>%s———> %-8s", dccp.Nstoa(end.Time), flushRight, pkt, end.State)
			pi++
		}
	}

	drop, end = end, drop
	return ps[:pi]
}

// Print orders a sequence of print recors by time and prints them to standard output
func Print(records []*PrintRecord) {
	sort.Sort(PrintTimeSort(records))
	for _, r := range records {
		fmt.Printf("%s\n", r.Text)
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
