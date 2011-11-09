// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package gauge

import (
	"json"
	"os"
	"unsafe"
	//"text/template"
	"github.com/petar/GoDCCP/dccp"
)

// D3 is a dccp.LogEmitter which processes the log stream into a JSON structure convenient
// for input into a D3 visualization.
type D3 struct {
	reducer LogReducer
}

func NewD3() *D3 {
	t := &D3{}
	t.Init()
	return t
}

func (t *D3) Init() {
	t.reducer.Init()
}

func (t *D3) Emit(r *dccp.LogRecord) {
	t.reducer.Emit(r)
}

// D3Data is a JSON structure passed to the D3-based visualizer
type D3Data struct {
	CheckIns  []*D3CheckIn  `json:"check_ins"`
	Places    []*D3Place    `json:"places"`
	Trips     []*D3Trip     `json:"trips"`
}

type D3CheckIn struct {
	Time      int64   `json:"time"`
	SeqNo     int64   `json:"seqno"`
	AckNo     int64   `json:"ackno"`
	Place     string  `json:"place"`
	Submodule string  `json:"sub"`
	Type      string  `json:"type"`
	State     string  `json:"state"`
	Comment   string  `json:"comment"`
}

type D3Place struct {
	Name      string        `json:"name"`
	Intervals []D3Interval  `json:"intervals"`
}

type D3Interval struct {
	State string  `json:"state"`
	Start int64   `json:"start"`
	End   int64   `json:"end"`
}

type D3Trip struct {
	SeqNo int64    `json:"seqno"`
	Path  []D3Stop `json:"path"`
}

type D3Stop struct {
	Place string  `json:"place"`
	Time  int64   `json:"time"`
}

func (t *D3) Close() *D3Data {
	d := &D3Data{}

	// Check-ins
	checkins := t.reducer.CheckIns()
	d.CheckIns = make([]*D3CheckIn, len(checkins))
	for i, rec := range checkins {
		d.CheckIns[i] = (*D3CheckIn)(unsafe.Pointer(rec))
	}

	// Places
	places := t.reducer.Places()
	d.Places = make([]*D3Place, len(places))
	var i int
	for name, place := range places {
		d3p := &D3Place{}
		d3p.Name = name
		d.Places[i] = d3p
		d3p.Intervals = make([]D3Interval, len(place.CheckIns))
		var prevInterval *D3Interval
		for j, placeCheckIn := range place.CheckIns {
			d3p.Intervals[j].State = placeCheckIn.State
			d3p.Intervals[j].Start = placeCheckIn.Time
			if prevInterval != nil {
				prevInterval.End = placeCheckIn.Time
			}
			d3p.Intervals[j].End = placeCheckIn.Time
			prevInterval = &d3p.Intervals[j]
		}
		i++
	}

	// Trips
	trips := t.reducer.Trips()
	d.Trips = make([]*D3Trip, len(trips))
	i = 0
	for seqno, trip := range trips {
		d3t := &D3Trip{}
		d3t.SeqNo = seqno
		d.Trips[i] = d3t
		d3t.Path = make([]D3Stop, len(trip.Forward)/*+len(trip.Backward)*/)
		for j, chk := range trip.Forward {
			d3t.Path[j].Place = chk.Module
			d3t.Path[j].Time = chk.Time
		}
		/*
		for j, chk := range trip.Backward {
			d3t.Path[len(trip.Forward) + j].Place = chk.Module
			d3t.Path[len(trip.Forward) + j].Time = chk.Time
		}
		*/
		i++
	}

	return d
}

func (t *D3) GetHTML() string {
	panic("un")
}

func OutToFile(env string, doc interface{}) error {
	name := os.Getenv(env)
	if name == "" {
		return nil
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	buf, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	if _, err = f.WriteString("data = "); err != nil {
		panic("export")
	}
	_, err = f.Write(buf)
	if err != nil {
		return err
	}
	if _, err = f.WriteString(";"); err != nil {
		panic("export")
	}
	return nil
}
