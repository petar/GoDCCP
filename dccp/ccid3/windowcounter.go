// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

// windowCounter maintains the window counter (WC) logic of the sender.
// It's logic is described in RFC 4342, Section 8.1.
type windowCounter struct {
	lastCounter byte   // The last window counter value sent
	lastTime    int64  // The time at which the first packet with window counter value lastCounter was sent
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
