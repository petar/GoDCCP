// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"time"
	"github.com/petar/GoDCCP/dccp"
)


// lossEvents is the algorithm that keeps track of loss events and constructs the
// loss intervals option upon request
type lossEvents struct {

	// pastHeaders keeps track of the last NDUPACK headers to overcome network re-ordering
	pastHeaders [NDUPACK]*headerEssence

	// pastIntervals keeps the most recent NINTERVAL finalized loss intervals
	pastIntervals [NINTERVAL]*LossInterval
	// nIntervals equals the total number of intervals pushed onto pastIntervals so far
	nIntervals    int64

	// lossyLen is the length of the lossy part of the current interval, so far.
	// A value of zero indicates that an interval has not begun yet.
	lossyLen   int
	// Sequence number of the first packet in the current loss interval
	startSeqNo int64
	// Timestamp of the first packet in the current loss interval
	startTime  int64
	// Round-trip time estimate at the beginning of the current loss interval
	startRTT   int64
}

type headerEssence struct {
	Type    byte
	X       bool
	SeqNo   int64
	CCVal   byte
	Options []*dccp.Options
}

const NINTERVAL = 8

// Init initializes/resets the lossEvents instance
func (t *lossEvents) Init() {
	?
}

// onOrderedRead is called after best effort has been made to fix packet 
// reordering. This function performs tha main loss intervals construction logic.
//
// NOTE: This implementation ignores loss events of non-Data or non-DataAck packets,
// however it includes them in the interval length reports. This significantly simplifies
// the data structures involved. It is thus imperative that non-data packets are sent
// much less often than data packets.
//
// XXX: It is possible to mend the sequence length overcounts by adding logic that
// subtracts the count of every successfully received non-data packet from the final
// sequence lengths.
func (t *lossEvents) onOrderedRead(he *headerEssence) {
	if he.Type != dccp.Data && he.Type != dccp.DataAck {
		return
	}
	?
}

// PushPopHeader places the newly arrived header he into pastHeaders and 
// returns potentially another header (if available) whose SeqNo is sooner.
// Every header is returned exactly once.
func (t *lossEvents) pushPopHeader(he *headerEssence) *headerEssence {
	var popSeqNo int64 = dccp.SEQNOMAX+1
	var pop int
	for i, ge := range t.pastHeaders {
		if ge == nil {
			t.pastHeaders[i] = he
			return nil
		}
		if ge.SeqNo < popSeqNo {
			pop = i
		}
	}
	r := t.pastHeaders[i]
	t.pastHeaders[i] = he
	return r
}

// receiver calls OnRead every time a new packet arrives
func (t *lossEvents) OnRead(htype byte, x bool, seqno int64, ccval byte, options []*dccp.Option) os.Error {
	he := t.pushPopHeader(&headerEssence{
		Type:    htype,
		X:       x,
		SeqNo:   seqno,
		CCVal:   ccval,
		Options: options,
	})
	if he != nil {
		t.onOrderedRead(he)
	}
}

// pushInterval saves li as the most recent finalized loss interval
func (t *lossEvents) pushInterval(li *LossInterval) {
	t.pastIntervals[int(t.nIntervals % len(t.pastIntervals))] = li
	t.nIntervals++
}

// listIntervals lists the available finalized loss from most recent to least
func (t *lossEvents) listIntervals() []*LossInterval {

	l := len(t.pastIntervals)
	k = int(min64(t.nIntervals, int64(l)))
	first := t.currentInterval()
	if first != nil {
		k++
	}

	// OPT: This slice allocation can be avoided by using a fixed instance
	r := make([]*LossInterval, k)

	var i int
	if first != nil {
		r[0] = first
		i++
	}
	p := int(t.nIntervals % int64(l)) + l
	for ; i < k; i++ {
		p--
		r[i] = t.pastIntervals[p % l]
	}
	return r
}

// currentInterval returns a loss interval for the current loss interval
// if it is considered long enough for inclusion. A nil is returned otherwise.
func (t *lossEvents) currentInterval() *LossInterval {
	?
}

func min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// Option returns the Loss Intervals option, representing the current state.
//
// NOTE: In a small deviation from the RFC, we don't send any loss intervals
// before the first loss event has occured. The sender is supposed to handle
// this adequately.
func (t *lossEvents) Option() *Option {
	return &LossIntervalsOption{
		SkipLength:    ?,
		LossIntervals: t.listIntervals(),
	}
}

