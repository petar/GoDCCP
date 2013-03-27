// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"sort"
	"github.com/petar/GoDCCP/dccp"
)

type PrintRecord struct {
	Log  *dccp.Trace
	Text string
}

// printRecord converts a log record into a PrintRecord
func printRecord(t *dccp.Trace) *PrintRecord {
	switch t.Event {
	case dccp.EventWrite:
		return printWrite(t)
	case dccp.EventRead:
		return printRead(t)
	case dccp.EventDrop:
		return printDrop(t)
	case dccp.EventIdle:
		return printIdle(t)
	}
	return printGeneric(t)
}

const (
	skip = "                                  "
	skipState = "         "
)

func printWrite(r *dccp.Trace) *PrintRecord {
	switch r.Labels[0] {
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

func printRead(r *dccp.Trace) *PrintRecord {
	switch r.Labels[0] {
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

func printDrop(r *dccp.Trace) *PrintRecord {
	var text string
	switch r.Labels[0] {
	// XXX: Seems there is a bug in the print out formats below (the server case format feels like it should be the line case)
	case "line":
		// XXX: this if makes no sense
		if r.Labels[1] == "server" {
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

func printIdle(r *dccp.Trace) *PrintRecord {
	var text string
	switch r.Labels[0] {
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

func printGeneric(r *dccp.Trace) *PrintRecord {
	var text string
	switch r.Labels[0] {
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

func sprintIdle(r *dccp.Trace) string {
	return "————————————————————————————————"
}

func sprintPacket(r *dccp.Trace) string {
	return sprintPacketWidth(r, 9)
}

func sprintPacketWide(r *dccp.Trace) string {
	if r.Type == "" {
		return ""
	}
	return fmt.Sprintf("Type=%s SeqNo=%06x AckNo=%06x", r.Type, r.SeqNo, r.AckNo)
}

func sprintPacketWidth(r *dccp.Trace, width int) string {
	var w bytes.Buffer
	w.WriteString(r.Type)
	for i := 0; i < width-len(r.Type); i++ {
		w.WriteRune('·')
	}
	return fmt.Sprintf(" %s%06x·%06x ", string(w.Bytes()), r.SeqNo, r.AckNo)
}

func sprintPacketEventComment(r *dccp.Trace) string {
	if r.SeqNo == 0 {
		return fmt.Sprintf("     %-22s     ", cut(r.Comment, 22))
	}
	return fmt.Sprintf("     %-14s %06x·     ", cut(r.Comment, 14), r.SeqNo)
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
