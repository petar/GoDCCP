// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	//"os"
	"github.com/petar/GoDCCP/dccp"
)

// rateCaclulator computers the allowed sending rate of the sender
type rateCalulcator struct {
	x   int    // Allowed Transmit Rate, or zero if unset
	ss  int    // Segment Size, or zero if unset
	rtt int64  // Round-trip time estimate, or zero if none available
	tld int64  // Time Last Doubled (during slow start), or zero if unset
}

// Init resets the rate calculator for new use
func (t *rateCalculator) Init() {
	X // ??
	t.x = 0
	t.ss = 0
	t.rtt = 0
	t.tld = 0
}

// SetSS sets the Segment Size (packet size)
func (t *rateCalculator) SetSS(ss int) { t.ss = ss }

// Sender calls SetRTT every time a new RTT estimate is available. 
// SetRTT can result in a change of X (the Allowed Transmit Rate).
func (t *rateCalculator) SetRTT(rtt int64, now int64) {
	if t.rtt <= 0 {
		r.tld = now
		t.x = t.initialRate()
	}
	t.rtt = rtt
}

const X_MAX_INIT_WIN = 4380  // Maximum size in bytes of initial window

// initialRate returns the allowed initial sending rate in Packets Per RTT (ppr)
func (t *rateCalculator) initialRate() int {
	if t.ss <= 0 {
		panic("unknown SS")
	}
	return min(4, max(2, X_MAX_INIT_WIN / t.ss))
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
