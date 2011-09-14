// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"github.com/petar/GoDCCP/dccp"
)

// nofeedbackTimer keeps track of the CCID3 nofeedback timeout at the
// sender. The timeout may change in response to various events.
type nofeedbackTimer struct {
	resetTime    int64 // Last time we got feedback; ns since UTC
	lastDataSent int64 // Time last data packet was sent, or zero otherwise; ns since UTC
	dataInvFreq  int64 // Interval between data packets, or zero if unknown; ns
	rtt          int64 // Current known round-trip time estimate, or zero if none; ns
}

const (
	NOFEEDBACK_WEIGHT_NEW      = 1
	NOFEEDBACK_WEIGHT_OLD      = 2
	NOFEEDBACK_TMO_WITHOUT_RTT = 2e9 // nofeedback timer expiration before RTT estimate, 2 sec
)

// Init resets the nofeedback timer for new use
func (t *nofeedbackTimer) Init() {
	t.resetTime = 0
	t.lastDataSent = 0
	t.dataInvFreq = 0
	t.rtt = 0
}

// Sender calls OnRead each time a feedback packet is received.
// OnRead restarts the nofeedback timer each time a feedback packet is received.
func (t *nofeedbackTimer) OnRead(rtt int64, rttEstimated bool, fb *dccp.FeedbackHeader) { 
	if fb.Type != dccp.Ack && fb.Type != dccp.DataAck {
		return
	}
	if rttEstimated {
		t.rtt = rtt
	} else {
		t.rtt = 0
	}
	t.Reset(fb.Time)
}

// Sender calls OnWrite each time a packet is sent out to the receiver.
// OnWrite is used to calculate timing between data packet sends.
func (t *nofeedbackTimer) OnWrite(ff *dccp.FeedforwardHeader) {
	// The very first time resetTime is set to equal the time when the first packet goes out,
	// since we are waiting for a feedback since that starting time. Afterwards, resetTime
	// can only assume times of incoming feedback packets.
	if t.resetTime <= 0 {
		t.resetTime = ff.Time
	}

	// Update inverse frequency of data packets estimate
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

// Sender calls OnIdle every time the idle clock ticks. OnIdle returns true if the
// nofeedback timer has expired.
func (t *nofeedbackTimer) IsExpired(now int64) bool {
	if t.resetTime <= 0 {
		return false
	}
	return now - t.resetTime >= t.timeout()
}

func (t *nofeedbackTimer) Reset(now int64) {
	t.resetTime = now
}

// timeout returns the current duration of the nofeedback timer in ns
func (t *nofeedbackTimer) timeout() int64 {
	if t.rtt <= 0 {
		return NOFEEDBACK_TMO_WITHOUT_RTT
	}
	if t.dataInvFreq <= 0 {
		return 4*t.rtt
	}
	return max64(4*t.rtt, 2*t.dataInvFreq)
}
