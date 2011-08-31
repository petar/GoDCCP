// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	//"os"
	"math"
	"github.com/petar/GoDCCP/dccp"
)

// rateCaclulator computers the allowed sending rate of the sender
type rateCalculator struct {
	x_bps int     // Allowed Sending Rate; bps (Bytes Per Second)
	x_sps int     // "       "       "   ; sps (Segments Per Second)

	ss    int     // Segment Size; b (Bytes)
	rtt   int64   // Round-Trip Time estimate, or zero if none available; ns (Nanoseconds)
	tld   int64   // Time Last Doubled (during slow start), or zero if unset; ns since UTC

	recvLimit uint32

	xRecvSet      // Data structure for x_recv_set (see RFC 5348)
}

const (
	X_MAX_INIT_WIN          = 4380  // Maximum size in bytes of initial window
	X_MAX_BACKOFF_INTERVAL  = 64e9  // Maximum backoff interval in ns (See RFC 5348, Section 4.3)
	X_TCP_MAX_PACKETS_ACKED = 1     // (Throughput Eq.) TCP max number of packets ack'd in single ack
)

// Init resets the rate calculator for new use
func (t *rateCalculator) Init() {
	X // ??
	t.x = 0
	t.ss = 0
	t.rtt = 0
	t.tld = 0
	t.xRecvSet.Init()
}

// SetSS sets the Segment Size (packet size)
func (t *rateCalculator) SetSS(ss int) { 
	t.ss = ss 
}

// Sender calls SetRTT every time a new RTT estimate is available. 
// SetRTT can result in a change of X (the Allowed Transmit Rate).
func (t *rateCalculator) SetRTT(rtt int64, now int64) {
	?? // this should be folded into OnRead
	if t.rtt <= 0 {
		r.tld = now
		t.x = t.initialRate()
	}
	t.rtt = rtt
}

func (t *rateCalculator) OnRead(now /* ns+UTC */ int64, x_recv /* bps */ uint32, rtt /* ns */ int64) {
	if /* the entire interval covered by the feedback packet was a data-limited interval */ {
		if /* the feedback packet reports a new loss event or an increase in the loss event rate p */ {
			t.xRecvSet.Halve()
			x_recv = (85 * x_recv) / 100  ???
			t.xRecvSet.Maximize(now, x_recv)
			t.recvLimit = t.xRecvSet.Max()
		} else {
			t.xRecvSet.Maximize(now, x_recv)
			t.recvLimit = 2 * t.xRecvSet.Max()
		}
	} else {
		t.xRecvSet.Update(now, x_recv, rtt)
		t.recvLimit = 2 * t.xRecvSet.Max()
	}
	var x_bps uint32
	if /* loss > 0 */ {
		x_eq_bps := equation ???
		x_bps = maxu32(
			minu32(x_eq_bps, t.recvLimit), 
			(1e9*t.ss)/X_MAX_BACKOFF_INTERVAL
		);
	} else if t_now - tld >= R {
		// Initial slow-start
		x_bps = max(min(2*X, recv_limit), initial_rate);
		tld = t_now;
	}
	// TODO: Place oscillation reduction code here (see RFC 5348, Section 4.3)
}

// ThruEq returns the allowed sending rate in bps according to the TCP throughput equation
func (t *rateCalculator) ThruEq(rtt int64, invLossRate uint32, rto int64) uint32 {
	??
}

// initRate returns the allowed initial sending rate in sps (segments per second) 
// after RTT and SS are known.
func (t *rateCalculator) initRate() uint32 {
	if t.ss <= 0 || t.rtt <= 0 {
		panic("unknown SS or RTT")
	}
	// window = bytes per round trip
	win := min(4*t.ss, max(2*t.ss, X_MAX_INIT_WIN))
	return uint32(max64((1e9*int64(win))/t.rtt, 1))
}

// —————
// xRecvSet maintains a set of recently received Receive Rates (via ReceiveRateOption)
type xRecvSet struct {
	set [X_RECV_SET_SIZE]xRecvEntry  // Set of recent rates
}

type xRecvEntry struct {
	Rate uint32  // Receive rate; bps (bytes per second)
	Time int64   // Timestamp when entry was received. Time=0 indicates nil entry.
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

// Max returns the highest rate in the set
func (t *xRecvSet) Max() uint32 {
	var r uint32 = -1
	for _, e := range t.set {
		if e.Time <= 0 {
			continue
		}
		if e.Rate > r {
			r = e.Rate
		}
	}
	if r < 0 {
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
func (t *xRecvSet) Maximize(now int64, x_recv uint32) {
	for i, e := range t.set {
		if e.Time > 0 {
			x_recv = max(x_recv, e.Rate)
		}
		t.set[i] = xRecvEntry{}
	}
	t.set[0].Time = now
	t.set[0].Rate = x_recv
}

// The procedure for updating X_recv_set keeps a set of X_recv values
// with timestamps from the two most recent round-trip times.
//
//   Update X_recv_set():
//     Add X_recv to X_recv_set;
//     Delete from X_recv_set values older than two round-trip times.
//
func (t *xRecvSet) Update(now int64, x_recv uint32, rtt int64) {
	// Remove entries older than two RTT
	for i, e := range t.set {
		if e.Time > 0 && now - e.Time > 2*rtt {
			t.set[i] = xRecvEntry{}
			i_free = i
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
	t.set[j].Rate = x_recv
	t.set[j].Time = now
}
