// Copyright 2010 GoDCCP Authors. All rights reserved.
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
	x_bps uint32 // Allowed Sending Rate; in bytes per second
	tld   int64  // Time Last Doubled (during slow start) or zero if unset; in ns since UTC zero

	recvLimit uint32

	xRecvSet      // Data structure for x_recv_set (see RFC 5348)
}

const (
	X_MAX_INIT_WIN          = 4380  // Maximum size of initial window in bytes
	X_MAX_BACKOFF_INTERVAL  = 64e9  // Maximum backoff interval in ns (See RFC 5348, Section 4.3)
)

// Init resets the rate calculator for new use
func (t *rateCalculator) Init() {
	// X // ??
	t.x_bps = 0
	t.tld = 0
	t.xRecvSet.Init()
}

// Sender calls SetRTT every time a new RTT estimate is available. 
// SetRTT can result in a change of X (the Allowed Transmit Rate).
/*
func (t *rateCalculator) SetRTT(rtt int64, now int64) {
	?? // this should be folded into OnRead
	if t.rtt <= 0 {
		r.tld = now
		t.x = t.initialRate()
	}
	t.rtt = rtt
}
*/

//func (t *rateCalculator) OnRead(now int64, x_recv uint32, rtt int64) {
//	if /* the entire interval covered by the feedback packet was a data-limited interval */ {
//		if /* the feedback packet reports a new loss event or an increase in the loss event rate p */ {
//			t.xRecvSet.Halve()
//			x_recv = (85 * x_recv) / 100  ???
//			t.xRecvSet.Maximize(now, x_recv)
//			t.recvLimit = t.xRecvSet.Max()
//		} else {
//			t.xRecvSet.Maximize(now, x_recv)
//			t.recvLimit = 2 * t.xRecvSet.Max()
//		}
//	} else {
//		t.xRecvSet.Update(now, x_recv, rtt)
//		t.recvLimit = 2 * t.xRecvSet.Max()
//	}
//	var x_bps uint32
//	if /* loss > 0 */ {
//		x_eq_bps := equation ???
//		x_bps = maxu32(
//			minu32(x_eq_bps, t.recvLimit), 
//			(1e9*t.ss)/X_MAX_BACKOFF_INTERVAL
//		);
//	} else if t_now - tld >= R {
//		// Initial slow-start
//		x_bps = max(min(2*X, recv_limit), initial_rate);
//		tld = t_now;
//	}
//	// TODO: Place oscillation reduction code here (see RFC 5348, Section 4.3)
//}

// ThruEq returns the allowed sending rate, in bytes per second, according to the TCP
// throughput equation, for the regime b=1 and t_RTO=4*RTT (See RFC 5348, Section 3.1).
func (t *rateCalculator) ThruEq(ss_b uint32, rtt_ns int64, lossRateInv uint32) uint32 {
	// XX
	bps := (1e9*int64(ss_b)) / (rtt_ns * thruEqDenom(lossRateInv))
	return uint32(bps)
}

// thruEqDenom computes the quantity (sqrt(2*p/3) + 12*sqrt(3*p/8)*p*(1+32*p^2)).
// XXX
func thruEqDenom(lossRateInv uint32) int64 {
	return -1
}

// initRate returns the allowed initial sending rate in bytes per second
func (t *rateCalculator) initRate(ss_b uint32, rtt_ns int64) uint32 {
	if ss_b <= 0 || rtt_ns <= 0 {
		panic("unknown SS or RTT")
	}
	win_bpr := minu32(4*ss_b, maxu32(2*ss_b, X_MAX_INIT_WIN)) // window = bytes per round trip (bpr)
	return uint32(max64((1e9*int64(win_bpr))/rtt_ns, 1))
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

const (
	X_RECV_SET_SIZE = 3              // Size of x_recv_set
	X_RECV_MAX      = math.MaxInt32  // Maximum receive rate
)

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
func (t *xRecvSet) Maximize(now int64, x_recv_bps uint32) {
	for i, e := range t.set {
		if e.Time > 0 {
			x_recv_bps = maxu32(x_recv_bps, e.Rate)
		}
		t.set[i] = xRecvEntry{}
	}
	t.set[0].Time = now
	t.set[0].Rate = x_recv_bps
}

// The procedure for updating X_recv_set keeps a set of X_recv values
// with timestamps from the two most recent round-trip times.
//
//   Update X_recv_set():
//     Add X_recv to X_recv_set;
//     Delete from X_recv_set values older than two round-trip times.
//
func (t *xRecvSet) Update(now int64, x_recv_bps uint32, rtt int64) {
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
	t.set[j].Rate = x_recv_bps
	t.set[j].Time = now
}
