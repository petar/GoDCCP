// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
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
	t.evolveInterval.Init(func(li *LossInterval) { t.pushInterval(li) })
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

// receiver calls OnRead every time a new packet arrives
func (t *lossEvents) OnRead(ff *dccp.FeedforwardHeader, rtt int64) os.Error {
	ff = t.pushPopHeader(ff)
	if ff != nil {
		t.evolveInterval.OnRead(ff, rtt)
	}
	return nil
}

// pushInterval saves li as the most recent finalized loss interval
func (t *lossEvents) pushInterval(li *LossInterval) {
	t.pastIntervals[int(t.nIntervals % int64(len(t.pastIntervals)))] = li
	t.nIntervals++
}

// listIntervals lists the available finalized loss intervals from most recent to least
func (t *lossEvents) listIntervals() []*LossInterval {

	l := len(t.pastIntervals)
	k := int(min64(t.nIntervals, int64(l)))
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
func (t *lossEvents) currentInterval() *LossInterval { return t.evolveInterval.Unfinished() }

// Option returns the Loss Intervals option, representing the current state.
//
// NOTE: In a small deviation from the RFC, we don't send any loss intervals
// before the first loss event has occured. The sender is supposed to handle
// this adequately.
func (t *lossEvents) Option() *LossIntervalsOption {
	return &LossIntervalsOption{
		// NOTE: We don't support SkipLength currently
		SkipLength:    0,
		LossIntervals: t.listIntervals(),
	}
}

// evolveInterval manages the incremental construction of a loss interval
type evolveInterval struct {

	// push is called whenever an interval is finished
	push        pushIntervalFunc

	// --- Last received packet state

	// lastSeqNo is the sequence number of the last successfuly received packet
	lastSeqNo   int64

	// lastTime is the time when the last packet was received
	lastTime    int64

	// lastRTT is the round-trip time when the last packet was received
	lastRTT     int64

	// nonDataLen is the number of non-data packets since last received (data) packet
	nonDataLen  int

	// --- Current interval state below

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

	// _tail is used as a volatile variable in OnRead
	_tail       eventTail
}

type pushIntervalFunc func(*LossInterval)

func (t *evolveInterval) Init(push pushIntervalFunc) {
	t.push = push
	t.lastSeqNo = 0
	t.lastTime = 0
	t.lastRTT = 0
	t.reset()
}

func (t *evolveInterval) reset() {
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
// NOTE: This implementation ignores receive events of non-Data or non-DataAck packets,
// however it includes them in the interval length reports (as losses). This simplifies
// the data structures involved. It is thus imperative that non-data packets are sent
// much less often than data packets.
//
// TODO: It is possible to mend the sequence length overcounts by adding logic that
// subtracts the count of every successfully received non-data packet from the final
// sequence lengths.
func (t *evolveInterval) OnRead(ff *dccp.FeedforwardHeader, rtt int64) {

	// If re-ordering still present, packet must be discarded
	if ff.SeqNo <= t.lastSeqNo {
		return
	}

	// Discard non-data packets from loss interval construction,
	if ff.Type != dccp.Data && ff.Type != dccp.DataAck {
		// But count the number of non-data packets since last received (data) packet
		t.nonDataLen++
		return
	}

	// Number of lost packets between this and the last received packets
	nlost := int(ff.SeqNo - t.lastSeqNo) - 1
	lastTime := t.lastTime
	lastSeqNo := t.lastSeqNo

	// Update last received event
	t.lastSeqNo = ff.SeqNo
	t.lastTime = ff.Time
	t.lastRTT = rtt
	t.nonDataLen = 0

	// Only perform updates after the second received packet
	if lastSeqNo > 0 {

		// Prepare tail between previous receive and this one
		t._tail.Init(lastTime, ff.Time, nlost, lastSeqNo)

		// Perform interval update
		t.eatTail(&t._tail)
	}
}

// eatTail does not interact with t.lastSeqNo, t.lastTime, or t.lastRTT.
func (t *evolveInterval) eatTail(tail *eventTail) {
	for tail != nil {
		// If no interval has started yet
		if t.lossLen == 0 {
			// If no losses, then we quit
			if tail.Lost() == 0 {
				return
			}
			// Otherwise, lossy section becomes a loss interval and
			// received packet becomes a lossless interval
			t.startTime, t.startSeqNo = tail.LossInfo(1)
			t.lossLen, t.startRTT = tail.Lost(), t.lastRTT
			t.losslessLen = 1
			return
		} else {
			// If the tail has no loss events, increase the lossless interval and quit
			if tail.Lost() == 0 {
				t.losslessLen++
				return
			}
			// Can we greedily increase the size of the loss section?
			if _, _, k := tail.LatestLoss(t.startTime+t.startRTT); k > 0 {
				t.lossLen += t.losslessLen + k
				t.losslessLen = 0
				tail.Chop(k)
				// If the new tail has no loss events, adjust the lossless section and quit
				if tail.Lost() == 0 {
					t.losslessLen = 1
					return
				}
				// Otherwise, finish the interval and consume the rest of tail
			}
			// If not, finish the current interval
			t.finishInterval()
		}
	}
}

func (t *evolveInterval) finishInterval() {
	if t.lossLen <= 0 {
		panic("no interval")
	}
	t.push(&LossInterval{
	       LosslessLength: uint32(t.losslessLen),
	       LossLength:     uint32(t.lossLen),
	       DataLength:     uint32(max(1, t.lossLen+t.losslessLen)),  // TODO: We don't count non-data packets yet
	       ECNNonceEcho:   false,
	})
	// This indicates that no interval is in progress
	t.lossLen = 0
}

// Unfinished returns a loss interval option for the current unfinished loss interval
// if it is considered long enough for inclusion in feedback to sender. A nil is returned
// otherwise.
func (t *evolveInterval) Unfinished() *LossInterval {
	if t.lossLen == 0 || t.lastTime - t.startTime < 2*t.startRTT {
		return nil
	}
	return &LossInterval{
		LosslessLength: uint32(t.losslessLen),
		LossLength:     uint32(t.lossLen),
		DataLength:     uint32(max(1, t.lossLen + t.losslessLen)),
		ECNNonceEcho:   false,
	}
}

// eventTail represents a yet unprocessed sequence of events that begins with a sequence of
// evenly spaced in time loss events and concludes with a successful receive event.
//
//    * <--gap--> X <--gap--> X <--gap--> X <-------> O
//  prev                                             recv
//
// Crucially, we allow prevTime == recvTime for whenever two packets are perceived
// as being received at the same time.
type eventTail struct {
	prevSeqNo int64 // SeqNo of previous event
	prevTime  int64 // Time of previous event (success or loss)
	gap       int64 // Time between adjacent loss events
	recvTime  int64 // Time of receive event
	nlost     int   // Number of loss events (X's in picture above)
}

// Init initializes the event tail.
func (t *eventTail) Init(prevTime, recvTime int64, nlost int, prevSeqNo int64) {
	if recvTime < prevTime {
		panic("second receive happens before first")
	}
	t.prevSeqNo = prevSeqNo
	t.prevTime = prevTime
	t.recvTime = recvTime
	t.nlost = nlost
	if nlost == 0 {
		t.gap = 1
	} else {
		t.gap = (recvTime-prevTime) / int64(nlost)
	}
}

// Lost returns the number of lost packets in this tail
func (t *eventTail) Lost() int { return t.nlost }

// LossInfo returns the time and sequence number of the k-th loss event.
// Loss events are numbered 1,2, ... ,t.Lost()
func (t *eventTail) LossInfo(k int64) (lossTime int64, lossSeqNo int64) {
	if k <= 0 {
		return -1, -1
	}
	return t.prevTime + k*t.gap, t.prevSeqNo + k
}

// LatestLoss returns the identity of the latest loss that occurred BEFORE-or-ON
// the deadline, as well as the number of loss events that fall within the deadline.
// The deadline is given in absolute time in nanoseconds. If no eligible loss
// is available, k equals 0.
func (t *eventTail) LatestLoss(deadline int64) (lossTime int64, lossSeqNo int64, k int) {
	d := deadline - t.prevTime
	if d < 0 || t.nlost == 0 {
		return -1, -1, 0
	}
	var k64 int64
	if t.gap == 0 {
		k64 = int64(t.nlost)
	} else {
		k64 = min64(d/t.gap, int64(t.nlost))
	}
	lossTime, lossSeqNo = t.LossInfo(k64)
	// TODO: Handle case when k overflows the int type
	return lossTime, lossSeqNo, int(k64)
}

// Chop removes the first k loss events from t.
func (t *eventTail) Chop(k int) {
	if k > t.nlost {
		panic("chopping more than available")
	}
	t.prevSeqNo += int64(k)
	t.prevTime += int64(k)*t.gap
	t.nlost -= k
}

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
