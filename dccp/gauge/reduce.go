// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package gauge

import (
	"sync"
	"github.com/petar/GoDCCP/dccp"
)

// LogReducer is a dccp.LogEmitter which processes the logs to a form
// that is convenient to illustrate via tools like D3 (Data-Driven Design).
type LogReducer struct {
	sync.Mutex
	checkIns []*dccp.LogRecord
	places   map[string]*Place
	trips    map[int64]*Trip
}

type Place struct {
	latest   int64
	CheckIns []*dccp.LogRecord
}

type Trip struct {
	Source string
	?
}

func NewLogReducer() *LogReducer {
	return &LogReducer{
		checkIns: make([]*dccp.LogRecord, 0, 16),
		places:   make(map[string]*Place),
		trips:    make(map[int64]*Trip),
	}
}

func (t *LogReducer) Emit(r *dccp.LogRecord) {
	t.Lock()
	defer t.Unlock()

	// Check-ins update
	t.checkIns = append(t.checkIns, r)

	// Places update
	p, ok := t.places[r.Module]
	if !ok {
		p = &Place{ 
			latest:   0,
			CheckIns: make([]*dccp.LogRecord, 0, 4) 
		}
		t.places[r.Module] = p
	}

	if r.Time <= p.latest {
		panic("backward time in reducer")
	}
	p.latest = r.Time

	if len(p.CheckIns) == 0 || p.CheckIns[len(p.CheckIns)-1].State != r.State {
		p.CheckIns = append(p.CheckIns, r)
	}

	// Trips update
	?
}

// CheckIns returns a list of all check-ins
func (t *LogReducer) CheckIns() []*dccp.LogRecord {
	t.Lock()
	defer func() { t.checkIns = nil }()  // So Emit does not try to update after this call accidentally
	defer t.Unlock()

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
