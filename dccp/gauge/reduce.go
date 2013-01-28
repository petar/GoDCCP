// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package gauge

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"github.com/petar/GoDCCP/dccp"
)

// LogReducer is a dccp.TraceWriter which processes the logs to a form
// that is convenient as input to illustration tools
type LogReducer struct {
	sync.Mutex
	checkIns []*dccp.Trace
	places   map[string]*Place
	trips    map[int64]*Trip
}

type Place struct {
	latest   *dccp.Trace
	CheckIns []*dccp.Trace
}

// A Trip instance captures all packet check-ins whose SeqNo or AckNo are related
type Trip struct {
	SeqNo    int64
	Forward  []*dccp.Trace
	Backward []*dccp.Trace
	Source   string
	Sink     string
	Round    bool
}

// Make a new log reducer
func NewLogReducer() *LogReducer {
	t := &LogReducer{}
	t.Init()
	return t
}

// Init (re)initializes an existing log reducer structure
func (t *LogReducer) Init() {
	t.checkIns = make([]*dccp.Trace, 0, 16)
	t.places = make(map[string]*Place)
	t.trips = make(map[int64]*Trip)
}

func (t *LogReducer) Write(r *dccp.Trace) {
	t.Lock()
	defer t.Unlock()

	// Check-ins update
	if r.Labels[0] == "" {
		panic("empty root label string in log")
	}
	t.checkIns = append(t.checkIns, r)

	// Places update
	p, ok := t.places[r.Labels[0]]
	if !ok {
		p = &Place{ 
			latest:   nil,
			CheckIns: make([]*dccp.Trace, 0),
		}
		t.places[r.Labels[0]] = p
	}

	if p.latest != nil && r.Time <= p.latest.Time {
		fmt.Fprintf(os.Stderr, "lastTime=%d, thisTime=%d\n", p.latest.Time, r.Time)
		fmt.Fprintf(os.Stderr, "last=%v\nthis=%v\n", p.latest, r)
		panic("backward time in reducer")
	}
	p.latest = r
	p.CheckIns = append(p.CheckIns, r)

	// Trips update
	if r.SeqNo != 0 {
		t.tripForward(r)
	}
	if r.AckNo != 0 {
		t.tripBackward(r)
	}
}

func (t *LogReducer) tripForward(r *dccp.Trace) {
	x, ok := t.trips[r.SeqNo]
	if !ok {
		x = &Trip{
			SeqNo:    r.SeqNo,
			Forward:  make([]*dccp.Trace, 0),
			Backward: make([]*dccp.Trace, 0),
		}
		t.trips[r.SeqNo] = x
	}

	x.Forward = append(x.Forward, r)
	sort.Sort(TraceChrono(x.Forward))
	x.Source = x.Forward[0].Labels[0]
	
	updateTrip(x)
}

func (t *LogReducer) tripBackward(r *dccp.Trace) {
	y, ok := t.trips[r.AckNo]
	if !ok {
		y = &Trip{
			SeqNo:    r.AckNo,
			Forward:  make([]*dccp.Trace, 0),
			Backward: make([]*dccp.Trace, 0),
		}
		t.trips[r.AckNo] = y
	}

	y.Backward = append(y.Backward, r)
	sort.Sort(TraceChrono(y.Backward))
	y.Sink = y.Backward[len(y.Backward)-1].Labels[0]

	updateTrip(y)
}

func updateTrip(t *Trip) {
	if t.Source == "" {
		return
	}
	if t.Source == t.Sink {
		t.Round = true
	}
}

// CheckIns returns a list of all check-ins
func (t *LogReducer) CheckIns() []*dccp.Trace {
	t.Lock()
	defer func() { t.checkIns = nil }()  // So Write does not try to update after this call accidentally
	defer t.Unlock()

	sort.Sort(TraceChrono(t.checkIns))
	return t.checkIns
}

// Places returns places' histories, keyed by place name
func (t *LogReducer) Places() map[string]*Place {
	t.Lock()
	defer func() { t.places = nil }() 
	defer t.Unlock()

	return t.places
}

// Trips returns trip records, keyed by SeqNo
func (t *LogReducer) Trips() map[int64]*Trip {
	t.Lock()
	defer func() { t.trips = nil }() 
	defer t.Unlock()
	
	return t.trips
}

func TripMapToSlice(m map[int64]*Trip) []*Trip {
	s := make([]*Trip, len(m))
	var i int
	for _, t := range m {
		s[i] = t
		i++
	}
	return s
}

// TraceChrono is a chronological sort driver for []*dccp.Trace
type TraceChrono []*dccp.Trace

func (t TraceChrono) Len() int {
	return len(t)
}

func (t TraceChrono) Less(i, j int) bool {
	return t[i].Time < t[j].Time
}

func (t TraceChrono) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
