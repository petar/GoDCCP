// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	//"os"
	"github.com/petar/GoDCCP/dccp"
)

// nofeedbackTimer keeps track of the CCID3 nofeedback timeout at the
// sender. The timeout may change in response to various events.
type nofeedbackTimer struct {
	lastFeedback int64 // Last time we got feedback. Timer is ticking since then
	lastDataSent int64 // Time last data packet was sent, or zero otherwise
	dataInvFreq  int64 // Interval between data packets, or zero if unknown
	rtt          int64 // Current known round-trip time estimate, or zero if none
}

// Init resets the nofeedback timer for new use
func (t *nofeedbackTimer) Init() {
	t.lastFeedback = 0
	t.lastDataSent = 0
	t.dataInvFreq = 0
	t.rtt = 0
}

// Sender calls OnRTT to announce a new round-trip time estimate
func (t *nofeedbackTimer) OnRTT(rtt int64) { t.rtt = rtt }

const (
	NOFEEDBACK_WEIGHT_NEW = 1
	NOFEEDBACK_WEIGHT_OLD = 2
)

// Sender calls OnRead each time a feedback packet is received.
// OnRead restarts the nofeedback timer each time a feedback packet is received.
func (t *nofeedbackTimer) OnRead(fb *dccp.FeedbackHeader) { t.lastFeedback = fb.Time }

// Sender calls OnWrite each time a packet is sent out to the receiver.
// OnWrite is used to calculate timing between data packet sends.
func (t *nofeedbackTimer) OnWrite(ff *dccp.FeedforwardHeader) {
	if ff.Type != dccp.Data && ff.Type != dccp.DataAck {
		return
	}
	if t.lastDataSent == 0 {
		t.lastDataSent = ff.Time
		return
	}
	d := ff.Time - t.lastDataSent
	if d <= 0 {
		return
	}
	if t.dataInvFreq == 0 {
		t.dataInvFreq = d
		return
	}
	t.dataInvFreq = (d*NOFEEDBACK_WEIGHT_NEW + t.dataInvFreq*NOFEEDBACK_WEIGHT_OLD) / 
		(NOFEEDBACK_WEIGHT_NEW + NOFEEDBACK_WEIGHT_OLD)
}

// Expired returns true if the nofeedback timer has expired
func (t *nofeedbackTimer) Expired (now int64) bool {
	if t.rtt <= 0 {
		panic("no rtt estimate in nofeedback timer")
	}
	if t.lastFeedback <= 0 {
		return false
	}
	var exp int64
	if t.dataInvFreq <= 0 {
		exp = 4*t.rtt
	} else {
		exp = max64(4*t.rtt, 2*t.dataInvFreq)
	}
	return now-t.lastFeedback >= exp
}
