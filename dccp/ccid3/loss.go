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
	pastHeaders [NDUPACK]*dccp.FeedforwardHeader

	// pastIntervals keeps the most recent NINTERVAL finalized loss intervals
	pastIntervals [NINTERVAL]*LossInterval
	// nIntervals equals the total number of intervals pushed onto pastIntervals so far
	nIntervals    int64

	// evolveInterval keeps state of the currently evolving loss interval
	evolveInterval
}

const NINTERVAL = 8

// Init initializes/resets the lossEvents instance
func (t *lossEvents) Init() {
	?
}

// PushPopHeader places the newly arrived header he into pastHeaders and 
// returns potentially another header (if available) whose SeqNo is sooner.
// Every header is returned exactly once.
func (t *lossEvents) pushPopHeader(ff *dccp.FeedforwardHeader) *dccp.FeedforwardHeader {
	var popSeqNo int64 = dccp.SEQNOMAX+1
	var pop int
	for i, ge := range t.pastHeaders {
		if ge == nil {
			t.pastHeaders[i] = ff
			return nil
		}
		if ge.SeqNo < popSeqNo {
			pop = i
		}
	}
	r := t.pastHeaders[i]
	t.pastHeaders[i] = ff
	return r
}

// receiver calls OnRead every time a new packet arrives
func (t *lossEvents) OnRead(ff *dccp.FeedforwardHeader, rtt int64) os.Error {
	ff = t.pushPopHeader(ff)
	if ff != nil {
		t.evolveInterval.OnRead(ff, rtt)
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
func (t *lossEvents) currentInterval() *LossInterval { return t.evolveInterval.Option() }

func min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
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

// evolveInterval manages the incremental construction of a loss interval
type evolveInterval struct {

	// lastSeqNo is the sequence number of the last successfuly received packet
	lastSeqNo   int64

	// lossLen is the length of the lossy part of the current interval so far, counting all
	// packet types. A value of zero indicates that an interval has not begun yet.
	lossLen     int

	// Sequence number of the first packet in the current loss interval
	startSeqNo  int64

	// Timestamp of the first packet in the current loss interval
	startTime   int64

	// Round-trip time estimate at the beginning of the current loss interval
	startRTT    int64

	// losslessLen is the length of the lossless part of the current interval so far, counting
	// all packet types. A non-zero value indicates that the lossless part has begun.
	losslessLen int

	// nonDataLen is the number of non-Data or non-DataAck packets successfully received in
	// the interval starting from startSeqNo up to now
	nonDataLen  int
}

func (t *evolveInterval) Init() {
	t.lastSeqNo = 0 XXX // Should this be reset?
	t.lossLen = 0
	t.startSeqNo = 0
	t.startTime = 0
	t.startRTT = 0
	t.losslessLen = 0
	t.nonDataLen = 0
}

// OnRead is called after best effort has been made to fix packet 
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
func (t *evolveInterval) OnRead(ff *dccp.FeedforwardHeader, rtt int64) {

	// If re-ordering still present, packet must be discarded
	if ff.SeqNo <= t.lastSeqNo {
		return
	}
	// Number of lost packets between this and the last received packets
	nLost := int(ff.SeqNo - t.lastSeqNo) - 1
	t.lastSeqNo = ff.SeqNo

	// Discard non-data packets from loss interval construction,
	if ff.Type != dccp.Data && ff.Type != dccp.DataAck {
		// But count the number of non-data packets inside a loss interval
		if t.lossLen > 0 {
			t.nonDataLen++
		}
		return
	}

	// If no interval has started yet
	if t.lossLen == 0 {
		// Cannot start a loss interval on a succesfully received packet
		if nLost == 0 {
			return
		}
		?
	}

	// Otherwise, we are in the middle of an ongoing interval
	?
}

// Given a sequence of nLost lost packets, sandwiched by two received packets
// whose receive times are preFirst and postLast, estimateLostTimes returns the
// estimated timestamps of the first and last lost packets, assuming a constant
// time period between pairs of adjacent packets.
func estimateLostTimes(preFirst, postLast int64, nLost int) (first, last int64) {
	?
}

// evolveInterval returns a loss interval option for the current state loss if it is
// considered long enough for inclusion in feedback to sender. A nil is returned otherwise.
func (t *evolveInterval) Option() *LossInterval {
	? // Add condition for including the current loss interval
	return &LossInterval{
		LosslessLength: t.losslessLen,
		LossLength:     t.lossLen,
		DataLength:     max(1, t.lossLen + t.losslessLen - t.nonDataLen),
		ECNNonceEcho:   false,
	}
}
