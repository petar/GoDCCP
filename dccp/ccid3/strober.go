// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// strober is an object that produces regular strobe intervals at a specified rate.
// A strober cannot be used before an initial call to SetInterval or SetRate.
type strober struct {
	dccp.CLog
	dccp.Mutex
	interval int64
	last     int64
}

// Per64FromBPS converts a rate in byter per second to
// packets of size ss per 64 seconds
func Per64FromBPS(bps uint32, ss uint32) int64 {
	return (64 * int64(bps)) / int64(ss)
}

// Init resets the strober instance for new use
func (s *strober) Init(clog dccp.CLog, bps uint32, ss uint32) {
	s.CLog = clog
	s.SetRate(bps, ss)
}

// SetWait sets the strobing rate by setting the time interval between two strobes in nanoseconds
func (s *strober) SetInterval(interval int64) {
	s.Lock()
	defer s.Unlock()
	s.interval = interval
	s.last = 0
}

// SetRate sets the strobing rate in strobes per 64 seconds
// Rates below 1 strobe per 64 sec are not allowed by RFC 4342
func (s *strober) SetRate(bps uint32, ss uint32) {
	s.Lock()
	defer s.Unlock()
	s.interval = 64e9 / Per64FromBPS(bps, ss)
	if s.interval == 0 {
		panic("zero strobe rate")
	}
}

// Strobe ensures that the frequency with which (multiple calls) to Strobe return does not
// exceed the allowed rate.  In particular, note that strober makes sure that after data
// limited periods, when the application is not calling it for a while, there is no burst of
// high frequency returns.  Strobe MUST not be called concurrently. For efficiency, it does
// not use a lock to prevent concurrent invocation. DCCP currently calls Strobe in a loop,
// so concurrent invocations are not a concern.
//
// TODO: This routine should be optimized
func (s *strober) Strobe() {
	s.Lock()
	now := time.Nanoseconds()
	delta := s.interval - (now - s.last)
	dbgInterval := s.interval // DBG
	s.Unlock()
	defer s.CLog.Logf("s-strober", "Event", "Strobe at %d pps", 1e9 / dbgInterval)
	if delta > 0 {
		<-time.NewTimer(delta).C
	}
	s.Lock()
	s.last = time.Nanoseconds()
	s.Unlock()
}
