// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// rttReceiver is the data structure that estimates the RTT at the receiver end.
// It's function is described in RFC 4342, towards the end of Section 8.1.
//
// TOOD: Because of the necessary constraint that measurements only come from packet pairs
// whose CCVals differ by at most 4, this procedure does not work when the inter-packet
// sending times are significantly greater than the RTT, resulting in packet pairs whose
// CCVals differ by 5.  Explicit RTT measurement techniques, such as Timestamp and Timestamp
// Echo, should be used in that case.
// 
type rttReceiver struct {

	// rtt equals the latest RTT estimate, or 0 otherwise
	rtt int64

	// rttTime is the time when rtt was estimated
	rttTime int64

	// ccvalNow is what we believe is the value of the current window is.
	// A value of CCValNil means no value.
	ccvalNow byte

	// ccvalTime[i] is the time when the first packet with CCVal=i was received.
	// A value of 0 indicates that no packet with this CCVal has been received yet.
	ccvalTime [WCTRMAX]int64
}
const CCValNil = 0xff

// Init initializes the RTT estimator algorithm
func (t *rttReceiver) Init() {
	t.rtt = 0
	t.rttTime = 0
	t.ccvalNow = CCValNil
	for i, _ := range t.ccvalTime {
		t.ccvalTime[i] = 0
	}
}

// receiver calls OnRead every time a packet is received
func (t *rttReceiver) OnRead(ccval byte) {
	ccval = ccval % WCTRMAX // Safety

	now := time.Nanoseconds()
	if t.ccvalNow == CCValNil || lessModWCTRMAX(ccval, t.ccvalNow) {
		t.Init()
		t.ccvalNow = ccval
		t.ccvalTime[ccval] = now
	} else {
		t.ccvalNow = ccval
		for i := byte(0); lessModWCTRMAX(ccval, (ccval+i) % WCTRMAX); i++ {
			t.ccvalTime[(ccval+i) % WCTRMAX] = 0
		}
	}

	t.calcCCValRTT()
}

// calcCCValRTT calculates RTT based on CCVal timing.
// This approximation is nicer than direct measurement, since it essentially 
// tries to approximate the sender's opinion of the RTT.
func (t *rttReceiver) calcCCValRTT() {
	if t.ccvalNow == CCValNil || t.ccvalTime[t.ccvalNow] == 0 {
		return
	}
	t0 := t.ccvalTime[t.ccvalNow]
	var t1 int64
	var q byte
	k := (t.ccvalNow + (WCTRMAX-4)) % WCTRMAX  // Equals (ccvalNow-4) mod WCTRMAX
	switch {
	case t.ccvalTime[k] != 0:
		t1 = t.ccvalTime[k]
		q = 4
	case t.ccvalTime[(k+1) % WCTRMAX] != 0:
		t1 = t.ccvalTime[(k+1) % WCTRMAX]
		q = 3
	case t.ccvalTime[(k+2) % WCTRMAX] != 0:
		t1 = t.ccvalTime[(k+2) % WCTRMAX]
		q = 2
	}
	if t1 == 0 {
		return
	}

	t.rtt = (4 * (t0-t1)) / int64(q)
	t.rttTime = t0
}

// RTT returns the best available estimate of the round-trip time
func (t *rttReceiver) RTT() int64 {
	now := time.Nanoseconds()
	if t.rtt != 0 &&  now - t.rttTime < 1e9 {
		return t.rtt
	}
	return 1e9
}


// rttSender is the data structure that estimates the RTT at the sender end.
type rttSender struct {
	estimate int64
	nSent    int
	history  [SENDER_RTT_HISTORY]sendTime  // Circular array, recording departure times of last few packets
}

type sendTime struct {
	SeqNo int64
	Time  int64
}

const (
	SENDER_RTT_HISTORY = 20 // How many timestamps of sent packets to remember
	SENDER_RTT_WEIGHT_NEW = 1
	SENDER_RTT_WEIGHT_OLD = 9
)

// Init resets the rttSender object for new use
func (t *rttSender) Init() {
	t.estimate = 0
	t.nSent = 0
	for i, _ := range t.history {
		t.history[i] = sendTime{} // Zero SeqNo indicates no data
	}
}

// Sender calls OnWrite for every packet sent.
func (t *rttSender) OnWrite(seqNo int64, now int64) {
	t.history[t.nSent % SENDER_RTT_HISTORY] = sendTime{seqNo, now}
	t.nSent++
	if t.nSent > 3 * SENDER_RTT_HISTORY {
		// Keep nSent small, yet reflecting that we've had some history already
		t.nSent = SENDER_RTT_HISTORY + ((t.nSent+1) % SENDER_RTT_HISTORY)
	}
}

func (t *rttSender) find(seqNo int64) *sendTime {
	for i := 0; i < SENDER_RTT_HISTORY && i < t.nSent; i++ {
		r := &t.history[(t.nSent-i-1) % SENDER_RTT_HISTORY]
		if r.SeqNo == seqNo {
			return r
		}
	}
	return nil
}

// Sender calls OnRead for every arriving Ack packet. OnRead returns
// true if the RTT estimate has been updated.
func (t *rttSender) OnRead(ackNo int64, elapsed *dccp.ElapsedTimeOption, now int64) bool {
	if elapsed == nil {
		return false
	}
	s := t.find(ackNo)
	if s == nil {
		return false
	}
	elapsedNS := dccp.NSFromTenUS(elapsed.Elapsed) // Elapsed time at receiver in nanoseconds
	if elapsedNS < 0 {
		return false
	}
	est := (now - s.Time - elapsedNS) / 2
	if est <= 0 {
		return false
	}
	est_old := t.estimate
	if est_old == 0 {
		t.estimate = est
	} else {
		t.estimate = (est*SENDER_RTT_WEIGHT_NEW + est_old*SENDER_RTT_WEIGHT_OLD) / 
			(SENDER_RTT_WEIGHT_NEW + SENDER_RTT_WEIGHT_OLD)
	}
	return true
}

// RTT returns the current round-trip time estimate, or the default if no estimate is
// available yet
func (t *rttSender) RTT() int64 {
	if t.estimate <= 0 {
		return dccp.RTT_DEFAULT
	}
	return t.estimate
}
