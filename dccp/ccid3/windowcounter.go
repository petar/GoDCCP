// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

// —————
// windowCounter maintains the window counter (WC) logic of the sender.
// It's logic is described in RFC 4342, Section 8.1.
type windowCounter struct {
	lastCounter byte   // The last window counter value sent
	lastTime    int64  // The time at which the first packet with window counter value lastCounter was sent
	windowHistory
}

const (
	// Maximum value of window counter, RFC 4342 Section 10.2 and RFC 3448
	WindowCounterMod  = 16
	// Add doc here
	WindowCounterHalf = (WindowCounterMod / 2) + (WindowCounterMod & 0x1)
)

func lessWindowCounterMod(x, y byte) bool {
	return (y-x) % WindowCounterMod < WindowCounterHalf
}

// Init resets the windowCounter instance for new use
func (wc *windowCounter) Init() {
	wc.lastCounter = 0
	wc.lastTime = 0
}

// The sender calls OnWrite in order to obtain the WC value to be included in the next
// outgoing packet
func (wc *windowCounter) OnWrite(rtt int64, now int64) byte {
	quarterRTTs := (now - wc.lastTime) / (rtt / 4)
	if quarterRTTs > 0 {
		wc.lastCounter = (wc.lastCounter + byte(min64(quarterRTTs, 5))) % WindowCounterMod
		wc.lastTime = now
	}
	return wc.lastCounter
}

// After receiving an acknowledgement for a packet sent with window counter wcAckd, the sender
// SHOULD increase its window counter, if necessary, so that subsequent packets have window
// counter value at least (wcAckd + 4) mod WindowCounterMod.
// XXX: What if local window counter has gone around the circle before the ack was received?
func (wc *windowCounter) OnRead(rtt int64, wcAckd byte, now int64) {
	atLeast := (wcAckd+4) % WindowCounterMod
	wouldCounter := wc.OnWrite(rtt, now)
	if lessWindowCounterMod(wouldCounter, atLeast) {
		wc.lastCounter = atLeast
		wc.lastTime = now
	}
}

// —————
// windowHistory remembers the CCVal window counter values of packets sent in the recent
// past, so that it can answer queries that map the sequence number of a past outgoing
// packet to its window counter value.
// TODO: Use circular arithmetic on sequence numbers
type windowHistory struct {
	j         int
	history   [WindowHistoryLen]windowStart
	lastSeqNo int64
}

type windowStart struct {
	StartSeqNo int64
	CCVal      byte
}

const WindowHistoryLen = 4*4*2

// Init resets the windowHistory instance for new use
func (t *windowHistory) Init() {
	t.j = 0
	for i, _ := range t.history {
		t.history[i] = windowStart{}
	}
	t.lastSeqNo = 0
}

// Add records that the window counter has changed to ccval at a packet with sequence number
// startSeqNo
func (t *windowHistory) Add(startSeqNo int64, ccval byte) {
	if startSeqNo <= t.lastSeqNo {
		panic("non-increasing sequence number")
	}
	t.history[t.j] = windowStart{startSeqNo, ccval}
	t.lastSeqNo = startSeqNo
	t.j = (t.j+1) % WindowHistoryLen
}

// Lookup returns the window counter value of the requested sequence number if it is
// recoverable, otherwise ok equals false
func (t *windowHistory) Lookup(seqNo int64) (ccval byte, ok bool) {
	// The seqNo argument comes from an AckNo in a feedback packet. DCCP checks ensure that this
	// number is bigger than the initial sequence number. Therefore Lookup can only be called
	// after at least one call to Add. Consequently, the only case in which the history does not
	// have the ccval of the seqNo is if the history has moved past it. This corresponds to the
	// least recent entry in the history having a StartSeqNo greater than seqNo.
	l := len(t.history)
	if t.history[t.j % l].StartSeqNo > seqNo {
		return 0, false
	}
	for i := 0; i < l; i++ {
		w := t.history[(t.j+i) % l]
		if w.StartSeqNo > seqNo {
			break
		}
		ccval = w.CCVal
	}
	return ccval, true
}
