// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)

// senderStrober is an object that produces regular strobe intervals at a specified rate.
// A senderStrober cannot be used before an initial call to SetInterval or SetRate.
type senderStrober struct {
	env *dccp.Env
	amb *dccp.Amb
	dccp.Mutex
	interval int64		// Maximum average time interval between packets, in nanoseconds
	last     int64
}

// BytesPerSecondToPacketsPer64Sec converts a rate in byter per second to
// packets of size ss per 64 seconds
func BytesPerSecondToPacketsPer64Sec(bps uint32, ss uint32) int64 {
	return (64 * int64(bps)) / int64(ss)
}

// Init resets the senderStrober instance for new use
func (s *senderStrober) Init(env *dccp.Env, amb *dccp.Amb, bps uint32, ss uint32) {
	s.env = env
	s.amb = amb.Refine("strober")
	s.SetRate(bps, ss)
}

// SetWait sets the strobing rate by setting the time interval between two strobes in nanoseconds
func (s *senderStrober) SetInterval(interval int64) {
	s.Lock()
	defer s.Unlock()
	s.interval = interval
	s.last = 0
}

// SetRate sets the strobing rate. The argument bps is the desired
// maximum bytes per second, while ss equals the maximum packet size.
// Internally, SetRate converts the two arguments into a maximum
// number of packets per 64 seconds, assuming all packets are of size ss.
// Rates below 1 strobe per 64 sec are not allowed by RFC 4342
func (s *senderStrober) SetRate(bps uint32, ss uint32) {
	s.Lock()
	defer s.Unlock()
	s.interval = 64e9 / BytesPerSecondToPacketsPer64Sec(bps, ss)
	if s.interval == 0 {
		panic("strobe rate infinity")
	}
	// This is high frequency. Consider calling it only when rate changes.
	// s.amb.E(dccp.EventInfo, fmt.Sprintf("Set strobe rate %d pps", 1e9 / s.interval))
}

func (s *senderStrober) SetRatePPS(pps uint32) {
	s.Lock()
	defer s.Unlock()
	if pps == 0 {
		panic("strobe rate zero pps")
	}
	s.interval = 1e9 / int64(pps)
	// This is high frequency. Consider calling it only when rate changes.
	// s.amb.E(dccp.EventInfo, fmt.Sprintf("Set strobe rate %d pps", 1e9 / s.interval))
}

// Strobe ensures that the frequency with which (multiple calls) to Strobe return does not
// exceed the allowed rate.  In particular, note that senderStrober makes sure that after data
// limited periods, when the application is not calling it for a while, there is no burst of
// high frequency returns.  Strobe MUST not be called concurrently. For efficiency, it does
// not use a lock to prevent concurrent invocation. DCCP currently calls Strobe in a loop,
// so concurrent invocations are not a concern.
//
// XXX: This routine should be optimized
func (s *senderStrober) Strobe() {
	s.Lock()
	now := s.env.Now()
	delta := s.interval - (now - s.last)
	_interval := s.interval
	s.Unlock()
	defer s.amb.E(dccp.EventInfo, fmt.Sprintf("Strobe at %d pps", 1e9 / _interval), nil)
	if delta > 0 {
		s.env.Sleep(delta)
	}
	s.Lock()
	s.last = s.env.Now()
	s.Unlock()
}
