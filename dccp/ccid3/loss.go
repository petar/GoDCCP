// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	//"log"
	"os"
	"math"
	"github.com/petar/GoDCCP/dccp"
)

// —————
// lossReceiver is the algorithm that keeps track of loss events and constructs the
// loss intervals option at the receiver.
type lossReceiver struct {

	// pastHeaders keeps track of the last NDUPACK headers to overcome network re-ordering
	pastHeaders [NDUPACK]*dccp.FeedforwardHeader

	// evolveInterval keeps state of the currently evolving loss interval
	evolveInterval

	// lossHistory keeps a moving tail of the past few loss intervals
	lossHistory

	// lossRateCalculator calculates loss event rates
	lossRateCalculator
}

// Init initializes/resets the lossReceiver instance
func (t *lossReceiver) Init() {
	t.evolveInterval.Init(func(lid *LossIntervalDetail) { t.lossHistory.Push(lid) })
	t.lossHistory.Init(NINTERVAL)
	t.lossRateCalculator.Init(NINTERVAL)
}

// pushPopHeader places the newly arrived header ff into pastHeaders and 
// returns potentially another header (if available) whose SeqNo is no later.
// Every header is returned exactly once.
func (t *lossReceiver) pushPopHeader(ff *dccp.FeedforwardHeader) *dccp.FeedforwardHeader {
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

// skipLength returns the number of packets, before and including the one being
// acknowledged, that are in the re-ordering queue pastHeaders and have not yet been
// considered by the loss intervals logic.
func (t *lossReceiver) skipLength(ackno int64) byte {
	var skip byte
	var dbgGSR int64 = 0
	for _, ge := range t.pastHeaders {
		if ge != nil {
			skip++
			dbgGSR = max64(dbgGSR, ge.SeqNo)
		}
	}
	if dbgGSR != ackno {
		panic("lossReceiver GSR != AckNo")
	}
	return byte(skip)
}

// receiver calls OnRead every time a new packet arrives
func (t *lossReceiver) OnRead(ff *dccp.FeedforwardHeader, rtt int64) os.Error {
	ff = t.pushPopHeader(ff)
	if ff != nil {
		t.evolveInterval.OnRead(ff, rtt)
	}
	return nil
}

// listIntervals lists the finished loss intervals from most recent to least, including
// the current (unfinished) interval as long as it is sufficiently long
func (t *lossReceiver) listIntervals() []*LossIntervalDetail {
	current := t.evolveInterval.Unfinished()
	cInd := 0
	if current != nil {
		cInd = 1
	}
	k := t.lossHistory.Len() + cInd

	// TODO: This slice allocation can be avoided by making r into a field 
	r := make([]*LossIntervalDetail, k)

	if cInd == 1 {
		r[0] = current
	}
	for i := 0; i < k-cInd; i++ {
		r[i+cInd] = t.lossHistory.Get(i)
	}

	return r
}

// LossIntervalsOption returns the Loss Intervals option, representing the current state.
// ackno is the seq no that the Ack packet is acknowledging. It equals the AckNo field of
// that packet.
//
// NOTE: In a deviation from the RFC, we don't send any loss intervals
// before the first loss event has occured. The sender is supposed to handle
// this adequately.
func (t *lossReceiver) LossIntervalsOption(ackno int64) *LossIntervalsOption {
	return &LossIntervalsOption{
		SkipLength:    t.skipLength(ackno),
		LossIntervals: stripLossIntervalDetail(t.listIntervals()),
	}
}

func stripLossIntervalDetail(s []*LossIntervalDetail) []*LossInterval {
	r := make([]*LossInterval, len(s))
	for i, e := range s {
		r[i] = &e.LossInterval
	}
	return r
}

func (t *lossReceiver) LossDigestOption() *LossDigestOption {
	panic("new loss count calculation not implemented")
	return &LossDigestOption{
		// RateInv is the inverse of the loss event rate, rounded UP, as calculated by the receiver.
		// A value of UnknownLossEventRateInv indicates that no loss events have been observed.
		RateInv:      t.LossEventRateInv(),
		// NewLoss indicates how many new loss events are reported by the feedback packet carrying this option
		NewLossCount: 0, // TODO: This is not implemented
	}
}

// LossEventRateInv returns the inverse of the loss event rate, calculated using the recent
// history of loss intervals as well as the current (unfinished) interval, if sufficiently long.
// A return value of UknownLossEventRateInv indicates that no lost packets have been encountered yet.
func (t *lossReceiver) LossEventRateInv() uint32 {
	return t.lossRateCalculator.CalcLossEventRateInv(t.listIntervals())
}

// —————
// lossRateCalculator calculates the inverse of the loss event rate as
// specified in Section 5.4, RFC 5348. One instantiation can perform repeated
// calculations using a fixed nInterval parameter.
type lossRateCalculator struct {
	nInterval int
	w         []float64
	h         []float64
}

// Init resets the calculator for new use with the given nInterval parameter
func (t *lossRateCalculator) Init(nInterval int) {
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
// TODO: This calculation should be replaced with an entirely integral one.
// TODO: Remove the most recent unfinished interval from the calculation, if too small. Not crucial.
func (t *lossRateCalculator) CalcLossEventRateInv(history []*LossIntervalDetail) uint32 {

	// Prepare a slice with interval lengths
	k := max(len(history), t.nInterval)
	if k < 2 {
		// Too few loss events are reported as UnknownLossEventRateInv which signifies 'no loss'
		return UnknownLossEventRateInv
	}
	h := t.h[:k]
	for i := 0; i < k; i++ {
		h[i] = float64(history[i].LossInterval.SeqLen())
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

	if I_mean < 1.0 {
		panic("invalid inverse")
	}
	return uint32(I_mean)
}

// —————
// lossHistory is a data structure that keeps track of a limited number of past loss
// intervals.
type lossHistory struct {

	// pastIntervals keeps the most recent NINTERVAL finalized loss intervals
	pastIntervals []*LossIntervalDetail

	// pushCount equals the total number of intervals pushed onto pastIntervals so far
	pushCount     int64
}

const NINTERVAL = 8

// Init initializes or resets the data structure
func (h *lossHistory) Init(nInterval int) {
	h.pastIntervals = make([]*LossIntervalDetail, nInterval)
	h.pushCount = 0
}

// pushInterval saves li as the most recent finalized loss interval
func (h *lossHistory) Push(lid *LossIntervalDetail) {
	h.pastIntervals[int(h.pushCount % int64(len(h.pastIntervals)))] = lid
	h.pushCount++
}

// Len returns the number of intervals in the history
func (h *lossHistory) Len() int {
	return int(min64(h.pushCount, int64(len(h.pastIntervals))))
}

// Get returns the i-th element in the history. The 0-th element is the most recent.
func (h *lossHistory) Get(i int) *LossIntervalDetail {
	l := int64(len(h.pastIntervals))
	return h.pastIntervals[int((h.pushCount-1-int64(i)) % l)]
}

// —————
// lossSender process loss rate options received at the sender and maintains relevant loss history.
type lossSender struct {
}
