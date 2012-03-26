// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"sort"
	//"github.com/petar/GoGauge/gauge"
	"github.com/petar/GoDCCP/dccp"
	dccp_gauge "github.com/petar/GoDCCP/dccp/gauge"
)

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
