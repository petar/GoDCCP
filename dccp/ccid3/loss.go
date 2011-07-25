// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"log"
	"os"
	"math"
	"github.com/petar/GoDCCP/dccp"
)

// —————
// lossEvents is the algorithm that keeps track of loss events and constructs the
// loss intervals option upon request
type lossEvents struct {

	// pastHeaders keeps track of the last NDUPACK headers to overcome network re-ordering
	pastHeaders [NDUPACK]*dccp.FeedforwardHeader

	// evolveInterval keeps state of the currently evolving loss interval
	evolveInterval

	// intervalHistory keeps a moving tail of the past few loss intervals
	intervalHistory

	// lossEventRateCalculator calculates loss event rates
	lossEventRateCalculator
}

// Init initializes/resets the lossEvents instance
func (t *lossEvents) Init() {
	t.evolveInterval.Init(func(li *LossInterval) { t.intervalHistory.Push(li) })
	t.intervalHistory.Init(NINTERVAL)
	t.lossEventRateCalculator.Init(NINTERVAL)
}

// pushPopHeader places the newly arrived header ff into pastHeaders and 
// returns potentially another header (if available) whose SeqNo is no later.
// Every header is returned exactly once.
func (t *lossEvents) pushPopHeader(ff *dccp.FeedforwardHeader) *dccp.FeedforwardHeader {
	var popSeqNo int64 = dccp.SEQNOMAX+1
	var pop int
	for i, ge := range t.pastHeaders {
		if ge == nil {
			t.pastHeaders[i] = ff
			return nil
		}
		// TODO: This must employ circular comparison
		if ge.SeqNo < popSeqNo {
			pop = i
			popSeqNo = ge.SeqNo
		}
	}
	r := t.pastHeaders[pop]
	t.pastHeaders[pop] = ff
	return r
}

func (t *lossEvents) skipLength(ackno int64) byte {
	var skip byte
	var dbgGSR int64 = 0
	for _, ge := range t.pastHeaders {
		if ge != nil {
			skip++
			dbgGSR = max64(dbgGSR, ge.SeqNo)
		}
	}
	if dbgGSR != ackno {
		log.Printf("lossEvents GSR != AckNo")
	}
	return byte(skip)
}

// receiver calls OnRead every time a new packet arrives
func (t *lossEvents) OnRead(ff *dccp.FeedforwardHeader, rtt int64) os.Error {
	ff = t.pushPopHeader(ff)
	if ff != nil {
		t.evolveInterval.OnRead(ff, rtt)
	}
	return nil
}

// listIntervals lists the finished loss intervals from most recent to least, including
// the current (unfinished) interval as long as it is sufficiently long
func (t *lossEvents) listIntervals() []*LossInterval {
	current := t.evolveInterval.Unfinished()
	cInd := 0
	if current != nil {
		cInd = 1
	}
	k := t.intervalHistory.Len() + cInd

	// OPT: This slice allocation can be avoided by using a field instance
	r := make([]*LossInterval, k)

	if cInd == 1 {
		r[0] = current
	}
	for i := 0; i < k-cInd; i++ {
		r[i+cInd] = t.intervalHistory.Get(i)
	}

	return r
}

// Option returns the Loss Intervals option, representing the current state.
//
// NOTE: In a small deviation from the RFC, we don't send any loss intervals
// before the first loss event has occured. The sender is supposed to handle
// this adequately.
func (t *lossEvents) Option(ackno int64) *LossIntervalsOption {
	return &LossIntervalsOption{
		SkipLength:    t.skipLength(ackno),
		LossIntervals: t.listIntervals(),
	}
}

// LossEventRateInv returns the inverse of the loss event rate, calculated using the recent
// history of loss intervals as well as the current (unfinished) interval, if sufficiently
// long.
func (t *lossEvents) LossEventRateInv() uint32 {
	return t.lossEventRateCalculator.CalcLossEventRateInv(t.listIntervals())
}

// —————
// lossEventRateCalculator calculates the inverse of the loss event rate as
// specified in Section 5.4, RFC 5348. One instantiation can perform repeated
// calculations using a fixed nInterval parameter.
type lossEventRateCalculator struct {
	nInterval int
	w         []float64
	h         []float64
}

func (t *lossEventRateCalculator) Init(nInterval int) {
	t.nInterval = nInterval
	t.w = make([]float64, nInterval)
	for i, _ := range t.w {
		t.w[i] = intervalWeight(i, nInterval)
	}
	t.h = make([]float64, nInterval)
}

func intervalWeight(i, nInterval int) float64 {
	if i < nInterval / 2 {
		return 1.0
	}
	return 2.0 * float64(nInterval-i) / float64(nInterval+2)
}

// CalcLossEventRateInv computes the inverse of the loss event rate, RFC 5348, Section 5.4.
// NOTE: We currently don't use the alternative algorithm, called History Discounting,
// discussed in RFC 5348, Section 5.5
func (t *lossEventRateCalculator) CalcLossEventRateInv(history []*LossInterval) uint32 {

	// Prepare a slice with interval lengths
	k := max(len(history), t.nInterval)
	if k < 2 {
		return UnknownLossEventRate
	}
	h := t.h[:k]
	for i := 0; i < k; i++ {
		h[i] = float64(history[i].SeqLen())
	}

	// Directly from the RFC
	var I_tot0 float64 = 0
	var I_tot1 float64 = 0
	var W_tot float64 = 0
	for i := 0; i < k-1; i++ {
		I_tot0 += h[i] * t.w[i]
		W_tot += t.w[i]
	}
	for i := 1; i < k; i++ {
		I_tot1 += h[i] * t.w[i-1]
	}
	I_tot := math.Fmax(I_tot0, I_tot1)
	I_mean := I_tot / W_tot

	return uint32(I_mean)
}

// —————
// intervalHistory is a data structure that keeps track of a limited number of past loss
// intervals.
type intervalHistory struct {

	// pastIntervals keeps the most recent NINTERVAL finalized loss intervals
	pastIntervals []*LossInterval

	// pushCount equals the total number of intervals pushed onto pastIntervals so far
	pushCount     int64
}

const NINTERVAL = 8

// Init initializes or resets the data structure
func (h *intervalHistory) Init(nInterval int) {
	h.pastIntervals = make([]*LossInterval, nInterval)
	h.pushCount = 0
}

// pushInterval saves li as the most recent finalized loss interval
func (h *intervalHistory) Push(li *LossInterval) {
	h.pastIntervals[int(h.pushCount % int64(len(h.pastIntervals)))] = li
	h.pushCount++
}

// Len returns the number of intervals in the history
func (h *intervalHistory) Len() int {
	return int(min64(h.pushCount, int64(len(h.pastIntervals))))
}

// Get returns the i-th element in the history. The 0-th element is the most recent.
func (h *intervalHistory) Get(i int) *LossInterval {
	l := int64(len(h.pastIntervals))
	return h.pastIntervals[int((h.pushCount-1-int64(i)) % l)]
}

func max64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}
