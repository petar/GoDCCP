// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// receiveRate keeps track of the data receive rate at the CCID3 receiver,
// and produces Receive Rate options for outgoing feedback packets.
// It's function is specified in RFC 4342, Section 8.3.
type receiveRate struct {
	data     int
	lastTime int64
}

// OnData is called to let the receiveRate know that data has been received
func (r *receiveRate) OnData(data int) {
	?
}

// Flush returns a Receive Rate option and indicates to receiveRate 
// that the next Ack-to-Ack window has begun
func (r *receiveRate) Flush(rtt int64) *Option {
	t := max64(now - r.lastTime, rtt)
	?
}
