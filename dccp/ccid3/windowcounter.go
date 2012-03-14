// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

// —————
// windowCounter maintains the window counter (WC) logic of the sender.
// It's logic is described in RFC 4342, Section 8.1.
type windowCounter struct {
	lastAckNoPresent bool   // Whether there have been any acks
	lastAckNo        int64  // The sequence number of the last acknowledged packet
	lastSeqNoPresent bool   // True if at least one packet has been sent
	lastSeqNo        int64  // Sequence number of last outgoing packet
	windowHistory
}

const (
	// Maximum value of window counter, RFC 4342 Section 10.2 and RFC 3448
	WindowCounterMod  = 16

	// The maximum increase in window counter from one window to the next
	WindowCounterMaxInc = 5

	// Counter increase since last ack'd window
	WindowCounterAckInc = 4
)

// diffWindowCounter returns the smallest non-negative integer than needs to be added to y
// to result in x, in the integers modulo WindowCounterMod.
func diffWindowCounter(x, y int8) int8 {
	x, y = x % WindowCounterMod, y % WindowCounterMod
	return (2*WindowCounterMod + x - y) % WindowCounterMod
}

// Init resets the windowCounter instance for new use
func (wc *windowCounter) Init() {
	wc.lastAckNoPresent = false
	wc.lastAckNo = 0
	wc.lastSeqNoPresent = false
	wc.lastSeqNo = 0
	wc.windowHistory.Init()
}

// The sender calls OnWrite in order to obtain the WC value to be included in the next
// outgoing packet
// TODO: Use RTT estimates from the sender's better estimator?
func (wc *windowCounter) OnWrite(rtt int64, seqNo int64, now int64) byte {
	// Update sequence number fields
	if wc.lastSeqNoPresent {
		if seqNo <= wc.lastSeqNo {
			panic("non-increasing seq no")
		}
	}
	wc.lastSeqNo = seqNo
	wc.lastSeqNoPresent = true

	// Check for first-packet condition
	latest := wc.windowHistory.Latest()
	var ccval int8
	if latest == nil {
		ccval = 0  // Initial ccval
		wc.windowHistory.Add(seqNo, now, ccval)
		return byte(ccval)
	}
	ccval = latest.CCVal

	// First compute the increase in ccval, based solely on how much time has passed since the
	// previous ccval number was issued and the estimated RTT. This number is never more than
	// WindowCounterMaxInc
	ccvalTimeInc := wc.getTimeBound(rtt, now)

	// After receiving an acknowledgement for a packet sent with window counter ccvalAck, the
	// sender SHOULD increase its window counter, if necessary, so that subsequent packets have
	// window counter value at least (ccvalAck + WindowCounterAckInc) mod WindowCounterMod.  
	ccvalAckInc := wc.getAckBound(now)

	ccvalInc := max8(ccvalTimeInc, ccvalAckInc)
	if ccvalInc > 0 {
		ccval = (ccval + ccvalInc) % WindowCounterMod
		wc.windowHistory.Add(seqNo, now, ccval)
	}
	return byte(ccval)
}

func (wc *windowCounter) getAckBound(now int64) (ccvalInc int8) {
	if !wc.lastAckNoPresent {
		return 0
	}
	latestToAckDiff, ok := wc.windowHistory.Lookup(wc.lastAckNo, WindowCounterAckInc)
	if !ok {
		// Latest ack is too far in the history
		return 0
	}
	return max8(0, (WindowCounterAckInc - latestToAckDiff))
}

// getTimeBound returns the least increase in ccval that the next packet must have,
// considering how much time has passed since the last window started.
// The returned value is never bigger than WindowCounterMaxInc.
func (wc *windowCounter) getTimeBound(rtt int64, now int64) (ccvalInc int8) {
	latest := wc.windowHistory.Latest()
	if latest == nil {
		panic("no window history")
	}
	quarterRTTs := (now - latest.StartTime) / (rtt / 4)
	if quarterRTTs < 0 {
		panic("time reversal")
	}
	if quarterRTTs > 0 {
		// The counter progresses up by the number of multiples of RTT/4, however
		// the progress never exceeds WindowCounterMaxInc counts.
		return int8(min64(quarterRTTs, WindowCounterMaxInc))
	}
	return 0
}

// Sender calls OnRead every time it receives an Ack or DataAck packet.
// OnRead simply keeps track of the highest acknowledged sequence number.
func (wc *windowCounter) OnRead(ackNo int64) {
	// Discard acknowledgements of unsent packets
	if !wc.lastSeqNoPresent || ackNo > wc.lastSeqNo {
		return
	}
	if wc.lastAckNoPresent {
		wc.lastAckNo = max64(wc.lastAckNo, ackNo)
	} else {
		wc.lastAckNoPresent = true
		wc.lastAckNo = ackNo
	}
}

// —————
// windowHistory remembers the CCVal window counter values of packets sent in the recent
// past, so that it can answer queries that map the sequence number of a past outgoing
// packet to its window counter value.
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
	if lastRec.StartTime != 0 {
		// ccvals cannot decrease
		if startSeqNo <= lastRec.StartSeqNo {
			panic("non-increasing sequence number")
		}
		// Time of outgoing packets should increase
		if startTime <= lastRec.StartTime {
			panic("non-increasing time")
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
// then to arrive at the latest window (in the history) ccval and ok is set to true. 
// Otherwise, ok is false.
func (t *windowHistory) Lookup(seqNo int64, ccvalDepth int) (ccvalDiff int8, ok bool) {
	prev := t.fetch(0)
	for i := 0; i < WindowHistoryLen && i < ccvalDepth; i++ {
		w := t.fetch(i)
		if w.StartTime == 0 {
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
	if w.StartTime == 0 {
		return nil
	}
	return w
}

// Latest returns a pointer to the most recent window
func (t *windowHistory) Latest() *windowStart {
	return t.fetch(0)
}
