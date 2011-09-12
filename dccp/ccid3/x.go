// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	//"os"
	"math"
	//"github.com/petar/GoDCCP/dccp"
)

// rateCaclulator computers the allowed sending rate of the sender
type rateCalculator struct {
	x           uint32 // Current allowed sending rate, in bytes per second
	tld         int64  // Time Last Doubled (during slow start) or zero if unset; in ns since UTC zero
	recvLimit   uint32 // Receive limit, in bytes per second
	recoverRate uint32 // (RFC 5348, Section 4.4)
	hasFeedback bool   // True if sender has received any feedback from the receiver
	xRecvSet           // Data structure for x_recv_set (see RFC 5348)
}

const (
	X_MAX_INIT_WIN          = 4380           // Maximum size of initial window in bytes
	X_MAX_BACKOFF_INTERVAL  = 64e9           // Maximum backoff interval in ns (See RFC 5348, Section 4.3)
	X_RECV_MAX              = math.MaxInt32  // Maximum receive rate, in bytes per second
	X_RECV_SET_SIZE         = 3              // Size of x_recv_set
)

// Init resets the rate calculator for new use and returns the initial 
// allowed sending rate (in bytes per second). The latter is the rate
// to be used before the first feedback packet is received and hence before
// an RTT estimate is available.
func (t *rateCalculator) Init(ss uint32) {
	// The allowed sending rate before the first feedback packet is received
	// is one packet per second.
	t.x = ss
	t.recoverRate = ss
	// tld = 0 indicates that the first feedback packet has yet not been received.
	t.tld = 0
	// Because X_recv_set is initialized with a single item, with value Infinity, recvLimit is
	// set to Infinity for the first two round-trip times of the connection.  As a result, the
	// sending rate is not limited by the receive rate during that period.  This avoids the
	// problem of the sending rate being limited by the value of X_recv from the first feedback
	// packet.
	t.recvLimit = X_RECV_MAX
	t.hasFeedback = false
	t.xRecvSet.Init()
}

// X returns the allowed sending rate in bytes per second
func (t *rateCalculator) X() uint32 { return t.x }

// onFirstRead is called internally to handle the very first feedback packet received.
func (t *rateCalculator) onFirstRead(now int64, ss uint32, rtt int64) uint32 {
	t.tld = now
	t.x = initRate(ss, rtt)
	return t.x
}

// initRate returns the allowed initial sending rate in bytes per second.
func initRate(ss uint32, rtt int64) uint32 {
	if ss <= 0 || rtt <= 0 {
		panic("unknown SS or RTT")
	}
	win := minu32(4*ss, maxu32(2*ss, X_MAX_INIT_WIN)) // window = bytes per round trip (bpr)
	return uint32(max64((1e9*int64(win)) / rtt, 1))
}

// XFeedback contains computed feedback variables that are used by the rate calculator to update the
// allowed sending rate
type XFeedback struct {
	Now   int64   // Time now
	SS    uint32  // Segment size
	XRecv uint32  // Receive rate
	RTT   int64   // Round-trip time
	LossFeedback  // Loss-related feedback
}

// Sender calls OnRead each time a new feedback packet (i.e. Ack or DataAck) arrives.
// OnRead returns the new allowed sending in bytes per second.
func (t *rateCalculator) OnRead(f XFeedback) uint32 {
	if f.LossFeedback.RateInv < 1 {
		panic("invalid loss rate inverse")
	}
	t.hasFeedback = true
	if t.tld <= 0 {
		return t.onFirstRead(f.Now, f.SS, f.RTT)
	}
	// TODO: We currently don't honor data-limited periods
	if false /* the entire interval covered by the feedback packet was a data-limited interval */ {
		if f.LossFeedback.RateInc || f.LossFeedback.NewLossCount > 0 {
			t.xRecvSet.Halve()
			f.XRecv = (85 * f.XRecv) / 100
			t.xRecvSet.Maximize(f.Now, f.XRecv)
			t.recvLimit = t.xRecvSet.Max()
		} else {
			t.xRecvSet.Maximize(f.Now, f.XRecv)
			t.recvLimit = 2 * t.xRecvSet.Max()
		}
	} else {
		t.xRecvSet.Update(f.Now, f.XRecv, f.RTT)
		t.recvLimit = 2 * t.xRecvSet.Max()
	}
	// Are we in the post-slow start phase
	if f.LossFeedback.RateInv < UnknownLossEventRateInv {
		xEq := t.thruEq(f.SS, f.RTT, f.LossFeedback.RateInv)
		t.x = maxu32(minu32(xEq, t.recvLimit), uint32((1e9 * int64(f.SS)) / X_MAX_BACKOFF_INTERVAL))
	} else if f.Now - t.tld >= f.RTT {
		// Initial slow-start
		t.x = maxu32(minu32(2*t.x, t.recvLimit), initRate(f.SS, f.RTT))
		t.tld = f.Now
	}
	// TODO: Place oscillation reduction code here (see RFC 5348, Section 4.3)
	return t.x
}

// Sender calls OnFeedbackExpire when the no feedback timer expires.
// OnFeedbackExpire returns the new allowed sending rate.
// See RFC 5348, Section 4.4
func (t *rateCalculator) OnFeedbackTimer(hasRTT bool, ss uint32, lossRateInv uint32, idleSince int64, nofeedbackSet int64) uint32 {
	xRecv := t.xRecvSet.Max()
	if !hasRTT && !t.hasFeedback && idleSince > nofeedbackSet {
		// We do not have X_Bps or recover_rate yet.
		// Halve the allowed sending rate.
		t.x = maxu32(t.x/2, uint32((1e9 * int64(ss)) / X_MAX_BACKOFF_INTERVAL));
	} else if 
		((lossRateInv < UnknownLossEventRateInv && xRecv < t.recoverRate) ||
		(lossRateInv == UnknownLossEventRateInv && t.x < 2*t.recoverRate)) &&
		idleSince <= nofeedbackSet {
		// Don't halve the allowed sending rate. Do nothing.
	} else if lossRateInv == UnknownLossEventRateInv {
		// We do not have X_Bps yet.
		// Halve the allowed sending rate.
		t.x = maxu32(t.x/2, uint32((1e9 * int64(ss)) / X_MAX_BACKOFF_INTERVAL));
	} else if t.x > 2*xRecv {
		// 2*X_recv was already limiting the sending rate.
		// Halve the allowed sending rate.
		t.updateLimits(xRecv)
	} else {
		// The sending rate was limited by X_Bps, not by X_recv.
		// Halve the allowed sending rate.
		t.updateLimits(t.x/2)
	}
	return t.x
}

// See RFC 5348, Section 4.4
func (t *rateCalculator) updateLimits(x uint32) {
	panic("?")
}

// thruEq returns the allowed sending rate, in bytes per second, according to the TCP
// throughput equation, for the regime b=1 and t_RTO=4*RTT (See RFC 5348, Section 3.1).
func (t *rateCalculator) thruEq(ss uint32, rtt int64, lossRateInv uint32) uint32 {
	bps := (1e3*1e9*int64(ss)) / (rtt * thruEqQ(lossRateInv))
	return uint32(bps)
}

// thruEqDenom computes the quantity 1e3*(sqrt(2*p/3) + 12*sqrt(3*p/8)*p*(1+32*p^2)).
func thruEqQ(lossRateInv uint32) int64 {
	j := min(int(lossRateInv), len(qTable))
	return qTable[j-1].Q
}

// —————
// xRecvSet maintains a set of recently received Receive Rates (via ReceiveRateOption)
type xRecvSet struct {
	set [X_RECV_SET_SIZE]xRecvEntry  // Set of recent rates
}

type xRecvEntry struct {
	Rate uint32   // Receive rate; in bytes per second
	Time int64    // Entry timestamp or zero if unset; in ns since UTC zero
}

// Init resets the xRecvSet object for new use
func (t *xRecvSet) Init() {
	for i, _ := range t.set {
		t.set[i] = xRecvEntry{}
	}
}

// Halve halves all the rates in the set
func (t *xRecvSet) Halve() {
	for i, _ := range t.set {
		t.set[i].Rate /= 2
	}
}

// Max returns the highest rate in the set; in bytes per second
func (t *xRecvSet) Max() uint32 {
	var set bool
	var r uint32
	for _, e := range t.set {
		if e.Time <= 0 {
			continue
		}
		if e.Rate > r {
			r = e.Rate
			set = true
		}
	}
	if !set {
		return X_RECV_MAX
	}
	return r
}

// The procedure for maximizing X_recv_set keeps a single value, the
// largest value from X_recv_set and the new X_recv.
//
//   Maximize X_recv_set():
//     Add X_recv to X_recv_set;
//     Delete initial value Infinity from X_recv_set, if it is still a member.
//     Set the timestamp of the largest item to the current time;
//     Delete all other items.
//
func (t *xRecvSet) Maximize(now int64, xRecvBPS uint32) {
	for i, e := range t.set {
		if e.Time > 0 {
			xRecvBPS = maxu32(xRecvBPS, e.Rate)
		}
		t.set[i] = xRecvEntry{}
	}
	t.set[0].Time = now
	t.set[0].Rate = xRecvBPS
}

// The procedure for updating X_recv_set keeps a set of X_recv values
// with timestamps from the two most recent round-trip times.
//
//   Update X_recv_set():
//     Add X_recv to X_recv_set;
//     Delete from X_recv_set values older than two round-trip times.
//
func (t *xRecvSet) Update(now int64, xRecvBPS uint32, rtt int64) {
	// Remove entries older than two RTT
	for i, e := range t.set {
		if e.Time > 0 && now - e.Time > 2*rtt {
			t.set[i] = xRecvEntry{}
		}
	}
	// Find free cell or oldest entry
	var j int = -1
	var j_time int64 = now
	for i, e := range t.set {
		if e.Time <= 0 {
			j = i
			break
		} 
		if e.Time <= j_time {
			j = i
			j_time = e.Time
		}
	}
	t.set[j].Rate = xRecvBPS
	t.set[j].Time = now
}
