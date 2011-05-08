// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
	"time"
)

// backOff{}
type backOff struct {
	s int64		// Next sleep duration
	t int64		// Total sleep so far
	m int64		// Maximum sleep
}

// newBackoff() creates a new back-off timer whose first wait 
// period is d1 nanoseconds. Consecutive periods back off exponentially
// up to a maximum of dmax nanoseconds in total (over all periods).
func newBackOff(t0, tmax int64) *backOff {
	return &backoff{ t0, 0, tmax }
}

// Sleep() blocks for the duration of the next sleep interval in the back-off 
// sequence and return nil. If the maximum total sleep time has been reached,
// Sleep() returns os.EOF without sleeping.
func (b *backOff) Sleep() os.Error {
	if b.t >= b.m {
		return os.EOF
	}
	time.Sleep(b.s)
	b.t += b.s
	b.s = (4*b.s) / 3
	return nil
}
