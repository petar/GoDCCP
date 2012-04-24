// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)

// receiverLossTracker implements the algorithm that keeps track of loss events and constructs the
// loss intervals option at the receiver.
type receiverLossTracker struct {

	amb *dccp.Amb

	// pastHeaders keeps track of the last NDUPACK headers to overcome network re-ordering
	pastHeaders [NDUPACK]*dccp.FeedforwardHeader

	// evolveInterval keeps state of the currently evolving loss interval
	evolveInterval

	// lossHistory keeps a moving tail of the past few loss intervals
	lossHistory

	// lossRateCalculator calculates loss event rates
	lossRateCalculator
}

// Init initializes/resets the receiverLossTracker instance
func (t *receiverLossTracker) Init(amb *dccp.Amb) {
	t.amb = amb
	t.evolveInterval.Init(amb, func(lid *LossIntervalDetail) { t.lossHistory.Push(lid) })
	t.lossHistory.Init(NINTERVAL)
	t.lossRateCalculator.Init(NINTERVAL)
}

// pushPopHeader places the newly arrived header ff into pastHeaders and 
// returns potentially another header (if available) whose SeqNo is no later.
// Every header is returned exactly once.
func (t *receiverLossTracker) pushPopHeader(ff *dccp.FeedforwardHeader) *dccp.FeedforwardHeader {
	var popSeqNo int64 = dccp.SEQNOMAX + 1
	var pop int
	for i, ge := range t.pastHeaders {
		if ge == nil {
			t.pastHeaders[i] = ff
			return nil
		}
		// XXX: This must employ circular comparison
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
func (t *receiverLossTracker) skipLength(ackno int64) byte {
	var skip byte
	var dbgGSR int64 = 0
	for _, ge := range t.pastHeaders {
		if ge != nil {
			skip++
			dbgGSR = max64(dbgGSR, ge.SeqNo)
		}
	}
	if dbgGSR != ackno {
		panic("receiverLossTracker GSR != AckNo")
	}
	return byte(skip)
}

// receiver calls OnRead every time a new packet arrives
func (t *receiverLossTracker) OnRead(ff *dccp.FeedforwardHeader, rtt int64) error {
	ff = t.pushPopHeader(ff)
	if ff != nil {
		t.evolveInterval.OnRead(ff, rtt)
	}
	return nil
}

// listIntervals lists the finished loss intervals from most recent to least, including
// the current (unfinished) interval as long as it is sufficiently long
func (t *receiverLossTracker) listIntervals() []*LossIntervalDetail {
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
func (t *receiverLossTracker) LossIntervalsOption(ackno int64) *LossIntervalsOption {
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

func (t *receiverLossTracker) LossDigestOption() *LossDigestOption {
	panic("new loss count calculation not implemented")
	return &LossDigestOption{
		// RateInv is the inverse of the loss event rate, rounded UP, as calculated by the receiver.
		// A value of UnknownLossEventRateInv indicates that no loss events have been observed.
		RateInv: t.LossEventRateInv(),
		// NewLoss indicates how many new loss events are reported by the feedback packet carrying this option
		NewLossCount: 0, // TODO: This is not implemented
	}
}

// LossEventRateInv returns the inverse of the loss event rate, calculated using the recent
// history of loss intervals as well as the current (unfinished) interval, if sufficiently long.
// A return value of UknownLossEventRateInv indicates that no lost packets have been encountered yet.
func (t *receiverLossTracker) LossEventRateInv() uint32 {
	rateInv := t.lossRateCalculator.CalcLossEventRateInv(t.listIntervals())
	t.amb.E(
		dccp.EventMatch, 
		fmt.Sprintf("receiver est loss event rate inv %0.3f%%", 100 / float64(rateInv)),
		LossSample(LossReceiverEstimateSample, rateInv),
	)
	return rateInv
}

// lossHistory is a data structure that keeps track of a limited number of past loss
// intervals.
type lossHistory struct {

	// pastIntervals keeps the most recent NINTERVAL finalized loss intervals
	pastIntervals []*LossIntervalDetail

	// pushCount equals the total number of intervals pushed onto pastIntervals so far
	pushCount int64
}

const NINTERVAL = 8

// Init initializes or resets the data structure
func (h *lossHistory) Init(nInterval int) {
	h.pastIntervals = make([]*LossIntervalDetail, nInterval)
	h.pushCount = 0
}

// pushInterval saves li as the most recent finalized loss interval
func (h *lossHistory) Push(lid *LossIntervalDetail) {
	h.pastIntervals[int(h.pushCount%int64(len(h.pastIntervals)))] = lid
	h.pushCount++
}

// Len returns the number of intervals in the history
func (h *lossHistory) Len() int {
	return int(min64(h.pushCount, int64(len(h.pastIntervals))))
}

// Get returns the i-th element in the history. The 0-th element is the most recent.
func (h *lossHistory) Get(i int) *LossIntervalDetail {
	l := int64(len(h.pastIntervals))
	return h.pastIntervals[int((h.pushCount-1-int64(i))%l)]
}
