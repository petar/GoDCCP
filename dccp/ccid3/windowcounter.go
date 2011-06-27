// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"math"
	"os"
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// windowCounter maintains the window counter (WC) logic of the sender.
// It's logic is described in RFC 4342, Section 8.1.
type windowCounter struct {
	last     byte   // The last window counter value sent
	lastTime int64  // The time at which the first packet with window counter value "last" was sent
}

const (
	// Maximum value of window counter, RFC 4342 Section 10.2 and RFC 3448
	WCTRMAX = 16
)

func (wc *windowCounter) Init(firstSendTime int64) {
	wc.last = 0
	wc.lastTime = firstSendTime
}

// The sender calls Take in order to obtain the WC value to be included in the next
// outgoing packet
func (wc *windowCounter) Take(rtt int64) byte {
	now := time.Nanoseconds()
	quarterRTTs := (now - wc.lastTime) / (rtt / 4)
	if quarterRTTs > 0 {
		wc.last = (wc.last + byte(min64(quarter_RTTs, 5))) % WCTRMAX
		wc.lastTime = now
	}
	return wc.last
}

// After receiving an acknowledgement for a packet sent with window counter wcAckd, the sender
// SHOULD increase its window counter, if necessary, so that subsequent packets have window
// counter value at least (wcAckd + 4) mod WCTRMAX.
// XXX: What if local window counter has gone around the circle before the ack was received?
func (wc *windowCounter) Place(wcAckd byte) {
	atLeast := (wcAckd+4) % WCTRMAX
	if lessModWCTRMAX(wc.last, atLeast) {
		wc.last = atLeast
	}
}

const WCTRHALF = (WCTRMAX / 2) + (WCTRMAX & 0x1)

func lessModWCTRMAX(x, y byte) bool {
	return (y-x) % WCTRMAX < WCTRHALF
}

func min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}
