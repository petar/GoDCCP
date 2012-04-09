// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)

// —————
// evolveInterval manages the incremental construction of a loss interval
type evolveInterval struct {
	
	amb *dccp.Amb

	// push is called whenever an interval is finished
	push        pushIntervalFunc

	// --- Last received packet state

	// lastSeqNo is the sequence number of the last successfuly received packet
	lastSeqNo   int64

	// lastTime is the time when the last packet was received
	lastTime    int64

	// lastRTT is the round-trip time estimate when the last packet was received
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

// LossIntervalDetail is an internal structure that contains more detailed information about a loss
// interval than the LossInterval structure that is transported inside a DCCP option.
type LossIntervalDetail struct {
	LossInterval
	StartSeqNo int64 // The sequence number of the first packet in the loss interval
	StartTime  int64 // The reception time of the first packet in the loss interval
	StartRTT   int64 // The RTT estimate when the interval began
	Unfinished bool  // True if the loss interval is still evolving
}

type pushIntervalFunc func(*LossIntervalDetail)

func (t *evolveInterval) Init(amb *dccp.Amb, push pushIntervalFunc) {
	t.amb = amb.Refine("evolveInterval")
	t.push = push
	t.lastSeqNo = 0
	t.lastTime = 0
	t.lastRTT = 0
	t.clearInterval()
}

func (t *evolveInterval) clearInterval() {
	t.lossLen = 0
	t.startSeqNo = 0
	t.startTime = 0
	t.startRTT = 0
	t.losslessLen = 0
	t.nonDataLen = 0
}

// OnRead is called after best effort has been made to fix packet 
// reordering. This function performs tha main loss interval construction logic.
//
// TODO: Account for non-data packets
func (t *evolveInterval) OnRead(ff *dccp.FeedforwardHeader, rtt int64) {

	// If sequence number re-ordering present, packet is not considered here, because it was
	// already counted as a lost packet when t.lastSeqNo was considered
	if ff.SeqNo <= t.lastSeqNo {
		return
	}
	// Packet re-ordering may also occur if a packet is received with a timestamp smaller than
	// that of the previously received one
	// XXX: How can this condition occur?
	if ff.Time < t.lastTime {
		t.amb.E(dccp.EventTurn, 
			fmt.Sprintf("Time re-order; SeqNo %06x,%06x", t.lastSeqNo, ff.SeqNo),
			ff)
		return
	}

	// Keep a separate count of non-Data packets
	if ff.Type != dccp.Data && ff.Type != dccp.DataAck {
		t.nonDataLen++
	}

	// Number of lost packets between this and the last received packets
	nlost := int(ff.SeqNo - t.lastSeqNo) - 1
	lastTime := t.lastTime
	lastSeqNo := t.lastSeqNo

	// Update last received event
	t.lastSeqNo = ff.SeqNo
	t.lastTime = ff.Time
	t.lastRTT = rtt

	// Only perform updates after the second packet ever received
	if lastSeqNo > 0 {

		// Prepare tail between previous receive and this one
		t._tail.Init(lastTime, ff.Time, nlost, lastSeqNo)

		// Perform interval update
		t.eatTail(&t._tail)
	}
}

// eatTail updates the currently evolving loss interval by processing the newly received eventTail.
// eatTail does not interact with t.lastSeqNo, t.lastTime, or t.lastRTT.
// Upon return, eatTail always leaves a non-nil interval in progress.
func (t *evolveInterval) eatTail(tail *eventTail) {
	// If the tail has loss events or there was an interval in progress, eatTail must always
	// leave an interval in progress upon return.  In particular, the only time when t.lossLen == 0 
	// outside of eatTail, is when no packet drops have been witnessed yet.
	if t.lossLen > 0 || tail.LostCount() > 0 {
		defer func() {
			if t.lossLen <= 0 || t.losslessLen <= 0 {
				panic("eatTail leaves no interval in progress")
			}
		}()
	}
	for tail != nil {
		// If no interval in progress
		if t.lossLen == 0 {
			// If no losses, then we quit
			if tail.LostCount() == 0 {
				return
			}
			// Otherwise, we start a new interval whose loss section becomes the lossy
			// section of the tail and whose lossless section becomes the single
			// received packet at the end of the tail
			t.startTime, t.startSeqNo = tail.GetLossEvent(1)
			t.lossLen, t.startRTT = tail.LostCount(), t.lastRTT
			t.losslessLen = 1
			return
		} else {
			// If the tail has no loss events, increase the lossless interval and quit
			if tail.LostCount() == 0 {
				t.losslessLen++
				return
			}
			// Can we greedily increase the size of the loss section?
			if _, _, k := tail.LatestLoss(t.startTime + t.startRTT); k > 0 {
				t.lossLen += t.losslessLen + k
				t.losslessLen = 0
				tail.Chop(k)
				// If the new tail has no loss events, adjust the lossless section and return
				if tail.LostCount() == 0 {
					t.losslessLen = 1
					return
				}
				// Otherwise, finish the interval and consume the rest of the tail
				// in the next iteration
			}
			// If not, finish the current interval (and begin a new one in the next iteration)
			t.finishInterval()
			// When exiting the iteration here, the tail ALWAYS contains loss events
		}
	}
}

func (t *evolveInterval) finishInterval() {
	if t.lossLen <= 0 {
		panic("finishing an interval without loss")
	}
	t.push(&LossIntervalDetail{
		LossInterval: LossInterval{
			LosslessLength: uint32(t.losslessLen),
			LossLength:     uint32(t.lossLen),
			DataLength:     uint32(t.lossLen + t.losslessLen),  // TODO: We don't count non-data packets yet
			ECNNonceEcho:   false,
		},
		StartSeqNo: t.startSeqNo,
		StartTime:  t.startTime,
		StartRTT:   t.startRTT,
		Unfinished: false,
	})
	t.clearInterval()
}

// Unfinished returns a loss interval for the current unfinished loss interval.
// A nil is returned only if no packet drops have been witnessed yet, because in this
// case, and in this case only, there is no interval in progress.
func (t *evolveInterval) Unfinished() *LossIntervalDetail {
	if t.lossLen <= 0 {
		return nil
	}
	if t.losslessLen <= 0 {
		panic("interval in progress missing lossless section")
	}
	// TODO: An optimization might consider not creating a new LossIntervalDetail each time
	// Unfinished is called, since this may in fact happen multiple times per loss interval
	// lifetime.
	return &LossIntervalDetail{
		LossInterval: LossInterval{
			LosslessLength: uint32(t.losslessLen),
			LossLength:     uint32(t.lossLen),
			DataLength:     uint32(t.lossLen + t.losslessLen), // TODO: We don't account for non-data packets yet
			ECNNonceEcho:   false,
		},
		StartSeqNo: t.startSeqNo,
		StartTime:  t.startTime,
		StartRTT:   t.startRTT,
		Unfinished: true,
	}
}

// —————
// eventTail represents a yet unprocessed sequence of events that begins with a sequence of
// evenly spaced-in-time loss events and concludes with a successful receive event.
//
//    * <--gap--> X <--gap--> X <--gap--> X <-------> O
//  prev                                             recv
//
// We allow prevTime == recvTime for whenever two packets are perceived
// as being received at the same time.
type eventTail struct {
	prevSeqNo int64 // SeqNo of previous event
	prevTime  int64 // Time of previous event (success or loss)
	gap       int64 // Time between adjacent loss events
	recvTime  int64 // Time of receive event
	nlost     int   // Number of loss events ("X"s in picture above)
}

// Init initializes the event tail.
func (t *eventTail) Init(prevTime, recvTime int64, nlost int, prevSeqNo int64) {
	if recvTime < prevTime {
		panic(fmt.Sprintf("second receive happens before first: %d, %d", prevTime, recvTime))
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
func (t *eventTail) LostCount() int { return t.nlost }

// GetLossEvent returns the time and sequence number of the k-th loss event.
// Loss events are numbered 1,2, ... ,t.LostCount()
func (t *eventTail) GetLossEvent(k int64) (lossTime int64, lossSeqNo int64) {
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
	lossTime, lossSeqNo = t.GetLossEvent(k64)
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
