// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

// —————
// windowCounter maintains the window counter (WC) logic of the sender.
// It's logic is described in RFC 4342, Section 8.1.
type windowCounter struct {
	lastAckNo   int64  // The sequence number of the last acknowledged packet
	windowHistory
}

const (
	// Maximum value of window counter, RFC 4342 Section 10.2 and RFC 3448
	WindowCounterMod  = 16

	// For a number, x, modulo WindowCounterMod the range (x,x+WindowCounterHalf) 
	// is considered 'greater than' x; the range (x-WindowCounterHalf,x) is 
	// considered 'less than x'
	WindowCounterHalf = (WindowCounterMod / 2) + (WindowCounterMod & 0x1)
)

XXX // All ccval's should be int8?

// diffWindowCounter returns the smallest non-negative integer than needs to be added to y
// to result in x, in the integers modulo WindowCounterMod.
func diffWindowCounter(x, y int8) int8 {
	x, y = x % mod, y % mod
	return (mod + mod + x - y) % mod
}

// Init resets the windowCounter instance for new use
func (wc *windowCounter) Init() {
	wc.lastAckNo = 0
	wc.windowHistory.Init()
}

// The sender calls OnWrite in order to obtain the WC value to be included in the next
// outgoing packet
// XXX: Use RTT estimates from the sender's better estimator?
func (wc *windowCounter) OnWrite(rtt int64, seqNo int64, now int64) byte {
	// First compute the ccval, based solely on how much time has passed since the previous
	// ccval number was issued and the estimated RTT. This number is never more than 5 units
	// bigger than the last ccval issued.
	ccval, update := wc.issueUponTime(rtt, now)

	// After receiving an acknowledgement for a packet sent with window counter ccvalAck, the
	// sender SHOULD increase its window counter, if necessary, so that subsequent packets have
	// window counter value at least (ccvalAck + 4) mod WindowCounterMod.  
	??
	// XXX: What if local window counter has gone around the circle before the ack was received?
	// XXX: acknowledgements that are too far ahead should reset the wc.lastAckNo field; those
	// that are too far behind should be neglected.
	ccvalAck, ok := wc.windowHistory.Lookup(wc.lastAckNo)
	if ok {
		atLeast := (ccvalAck+4) % WindowCounterMod
		if lessWindowCounterMod(ccval, atLeast) {
			ccval = atLeast
			update = true
		}
	}

	if update {
		wc.windowHistory.Add(seqNo, now, ccval)
	}
	return ccval
}

// If a packet from one of the past 4 ccval units?? issued has been acknowledged, 
// getAckBound returns ...
func (wc *windowCounter) getAckBound(now int64) (addAtLeast int8, ok bool) {
	// If no ack is present
	if wc.lastAckNo == 0 {
		return 0, false
	}
	ccdiffAck, ok := wc.windowHistory.Lookup(wc.lastAckNo, 4)
	if !ok {
		return 0, false
	}
	return (ccvalAck+4) % WindowCounterMod, true
}

// getTimeBound returns the least ccval that the next packet must have,
// in consideration of the rule that
// on time difference to the previous one and the current round-trip estimate.
// update is set if the returned counter value represents a new window.
func (wc *windowCounter) getTimeBound(rtt int64, now int64) (ccval byte, update bool) {
	latest := wc.windowHistory.Latest()
	if latest == nil {
		??
	}
	quarterRTTs := (now - wc.lastTime) / (rtt / 4)
	if quarterRTTs > 0 {
		// The counter progresses up by the number of multiples of RTT/4, however
		// the progress never exceeds 5 counts.
		ccval = (wc.lastCCVal + byte(min64(quarterRTTs, 5))) % WindowCounterMod
		update = true
	} else {
		ccval = wc.lastCCVal
		update = false
	}
	return ccval, update
}

// Sender calls OnRead every time it receives an Ack or DataAck packet.
// OnRead simply keeps track of the highest acknowledged sequence number.
func (wc *windowCounter) OnRead(ackNo int64) {
	wc.lastAckNo = max64(wc.lastAckNo, ackNo)
}

// —————
// windowHistory remembers the CCVal window counter values of packets sent in the recent
// past, so that it can answer queries that map the sequence number of a past outgoing
// packet to its window counter value.
// XXX: Use circular arithmetic on sequence numbers
type windowHistory struct {
	j         int
	history   [WindowHistoryLen]windowStart
}

type windowStart struct {
	StartSeqNo int64
	StartTime  int64
	CCVal      int8
}

const WindowHistoryLen = 4*4*2

// Init resets the windowHistory instance for new use
func (t *windowHistory) Init() {
	t.j = 0
	for i, _ := range t.history {
		t.history[i] = windowStart{}
	}
}

// Add adds a new window to the history with a given starting sequence number and a counter value.
func (t *windowHistory) Add(startSeqNo int64, startTime int64, ccval int8) {
	lastRec := t.fetch(0)
	if lastRec.StartSeqNo != 0 {
		// ccvals cannot decrease
		if startSeqNo <= lastRec.StartSeqNo {
			panic("non-increasing sequence number")
		}
		// ccvals cannot increase faster than 5 units at a time
		if diffWindowCounter(ccval, lastRec.CCVal) > 5 {
			panic("ccvals increase too fast")
		}
	}
	t.history[t.j] = windowStart{startSeqNo, startTime, ccval}
	t.j = (t.j+1) % WindowHistoryLen
}

// Lookup locates the counter window containing sequence number seqNo.  If successful, it
// returns the number of units (not modulo WindowCounterMod) that were added to ccval since
// then to arrive at the latest window's ccval and ok is set to true. Otherwise, ok is false.
func (t *windowHistory) Lookup(seqNo int64, ccvalDepth int) (ccvalDiff int8, ok bool) {
	prev := t.fetch(0)
	for i := 0; i < WindowHistoryLen && i < ccvalDepth; i++ {
		w := t.fetch(i)
		if w.StartSeqNo == 0 {
			return 0, false
		}
		if prev != nil {
			ccvalDiff += diffWindowCounter(prev.CCVal, w.CCVal)
		}
		if w.StartSeqNo <= seqNo {
			return ccvalDiff, true
		}
	}
	return 0, false
}

// Fetch returns the i-th record, where the 0-th record is the last record
// added; the 1-st record is the one added before that, and so forth.
// Fetch returns nil, if the window history contains fewer than i+1 records.
func (t *windowHistory) fetch(i int) *windowStart {
	w := &t.history[(WindowHistoryLen + t.j - 1 - i) % WindowHistoryLen]
	if w.StartSeqNo == 0 {
		return nil
	}
	return w
}

// Latest returns a pointer to the most recent window
func (t *windowHistory) Latest() *windowStart {
	return t.fetch(0)
}
