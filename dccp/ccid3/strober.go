// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// strober is an object that produces regular strobe intervals at a specified rate
type strober struct {
	dccp.Mutex
	interval int64
	last     int64
}

// newStrober creates a new strober initialized at 1 packet per second
func newStrober() *strober {
	return &strober{
		interval: 1e9,
	}
}

// SetWait sets the strobing rate by setting the time interval between two strobes in nanoseconds
func (s *strober) SetInterval(interval int64) {
	s.Lock()
	defer s.Unlock()
	s.interval = interval
}

// SetRate sets the strobing rate in strobes per 10 seconds
// Rates below 1 strobe per 10 sec are not allowed
func (s *strober) SetRate(per10sec int64) {
	s.Lock()
	defer s.Unlock()
	s.interval = 10e9 / per10sec
	if s.interval == 0 {
		panic("zero strobe rate")
	}
}

// Strobe ensures that the frequency with which (multiple calls) to Strobe return does not exceed
// the allowed rate.  Strobe MUST not be called concurrently. For efficiency, it does not use a lock
// to prevent concurrent invocation.
func (s *strober) Strobe() {
	s.Lock()
	now := time.Nanoseconds()
	delta = s.interval - (now - s.last)
	s.Unlock()
	if delta > 0 {
		<-time.NewTimer(delta).C
	}
	s.Lock()
	s.last = time.Nanoseconds()
	s.Unlock()
}
