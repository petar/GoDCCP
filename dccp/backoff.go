// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// backOff{}
type backOff struct {
	Time
	sleep       int64 // Duration of next sleep interval
	lifetime    int64 // Total lifetime so far
	maxLifetime int64 // Maximum time the backoff mechanism stays alive
	backoffFreq int64 // Backoff period. The sleep duration backs off approximately every backoffFreq nanoseconds
	lastBackoff int64 // Last time the sleep interval was backed off, relative to the starting time
}

// newBackOff() creates a new back-off timer whose first wait period is firstSleep
// nanoseconds. Approximately every backoffFreq nanoseconds, the sleep timers backs off
// (increases by a factor of 4/3).  The lifetime of the backoff sleep intervals does not
// exceed maxLifetime.
func newBackOff(time Time, firstSleep, maxLifetime, backoffFreq int64) *backOff {
	return &backOff{
		Time:        time,
		sleep:       firstSleep,
		lifetime:    0,
		maxLifetime: maxLifetime,
		backoffFreq: backoffFreq,
		lastBackoff: 0,
	}
}

// Sleep() blocks for the duration of the next sleep interval in the back-off 
// sequence and return nil. If the maximum total sleep time has been reached,
// Sleep() returns os.EOF without sleeping.
func (b *backOff) Sleep() (os.Error, int64) {
	if b.lifetime >= b.maxLifetime {
		return os.EOF, 0
	}
	b.Time.Sleep(b.sleep)
	b.lifetime += b.sleep
	if b.lifetime - b.lastBackoff >= b.backoffFreq {
		b.sleep = (4 * b.sleep) / 3
		b.lastBackoff = b.lifetime
	}
	return nil, b.Time.Nanoseconds()
}
