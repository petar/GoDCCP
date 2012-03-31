// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)

// —————
// senderLossTracker processes loss intervals options received at the sender and maintains relevant loss
// statistics.
type senderLossTracker struct {
	logger *dccp.Logger
	lastAckNo   int64  // SeqNo of the last ack'd segment; equals the AckNo of the last feedback
	lastRateInv uint32 // Last known value of loss event rate inverse
	lossRateCalculator
}

// Init resets the senderLossTracker instance for new use
func (t *senderLossTracker) Init(logger *dccp.Logger) {
	t.logger = logger.Refine("senderLossTracker")
	t.lastAckNo = 0
	t.lastRateInv = UnknownLossEventRateInv
	t.lossRateCalculator.Init(NINTERVAL)
}

// calcRateInv computes the loss event rate inverse encoded in the loss intervals
func (t *senderLossTracker) calcRateInv(details []*LossIntervalDetail) uint32 {
	return t.lossRateCalculator.CalcLossEventRateInv(details)
}

// LossFeedback contains summary of loss information updates returned by OnRead
type LossFeedback struct {
	RateInv      uint32 // Loss event rate inverse
	NewLossCount byte   // Number of loss events reported in this feedback packet
	RateInc      bool   // Has the loss rate increased since the last feedback packet
}

// Sender calls OnRead whenever a new feedback packet arrives
func (t *senderLossTracker) OnRead(fb *dccp.FeedbackHeader) (LossFeedback, error) {

	// Read the loss options
	if fb.Type != dccp.Ack && fb.Type != dccp.DataAck {
		return LossFeedback{}, ErrNoAck
	}
	var lossIntervals *LossIntervalsOption
	t.logger.E("Event", fmt.Sprintf("Encoded option count = %d", len(fb.Options)), fb)
	for i, opt := range fb.Options {
		if lossIntervals = DecodeLossIntervalsOption(opt); lossIntervals != nil {
			break
		}
		t.logger.E("Event", fmt.Sprintf("Decodingd option %d", i), fb)
	}
	if lossIntervals == nil {
		t.logger.E("Event", "Missing lossIntervals", fb)
		return LossFeedback{}, ErrMissingOption
	}

	// Calcuate new loss count
	var r LossFeedback
	details := recoverIntervalDetails(fb.AckNo, lossIntervals.SkipLength, lossIntervals.LossIntervals)
	r.NewLossCount = calcNewLossCount(details, t.lastAckNo)

	// Calculate new rate inverse
	rateInv := t.calcRateInv(details)
	r.RateInv = rateInv
	if rateInv < t.lastRateInv {
		r.RateInc = true
	}
	t.lastRateInv = rateInv

	// XXX: Must use circular arithmetic here
	t.lastAckNo = max64(t.lastAckNo, fb.AckNo)

	return r, nil
}

// recoverIntervalDetails returns a slice containing the estimated details of the loss intervals
func recoverIntervalDetails(ackno int64, skip byte, lis []*LossInterval) []*LossIntervalDetail {
	r := make([]*LossIntervalDetail, len(lis))
	var head int64 = ackno + 1 - int64(skip)
	for i, li := range lis {
		r[i] = &LossIntervalDetail{}
		r[i].LossInterval = *li
		head -= int64(li.SeqLen())
		r[i].StartSeqNo = head
		// TODO: StartTime, StartRTT, Unfinished are not recovered (but also not used)
	}
	return r
}

// calcNewLossCount calculates the number of new loss intervals reported in this feedback packet,
// since the last packet (identified by lastAckNo)
// XXX: Must use circular arithmetic here
func calcNewLossCount(details []*LossIntervalDetail, lastAckNo int64) byte {
	// If lastAckNo is zero (no acks have been received), this function works correctly
	var r byte
	for _, d := range details {
		if d.StartSeqNo <= lastAckNo {
			break
		}
		r++
	}
	return r
}
