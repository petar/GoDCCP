// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// receiveRate keeps track of the data receive rate at the CCID3 receiver,
// and produces Receive Rate options for outgoing feedback packets.
// It's function is specified in RFC 4342, Section 8.3.
//
// XXX: Section 8.1, on the other hand, seems to suggest an alternative
// mechanism for computing receive rate, based on the window counter values
// in CCVal.
type receiveRate struct {
	data0, data1 int
	time0, time1 int64
}

func (r *receiveRate) Init() {
	now := time.Nanoseconds()
	r.time0, r.time1 = now, now
	r.data0, r.data1 = 0, 0
}

// OnData is called to let the receiveRate know that data has been received.
// The ccval window counter value is not used in the current rate receiver algorithm
// explicitly. It is used implicitly in that the RTT estimate is based on these values.
func (r *receiveRate) OnRead(ff *dccp.FeedforwardHeader) {
	if ff.Type != dccp.Data && ff.Type != dccp.DataAck {
		return
	}
	r.data0 += ff.DataLen
	r.data1 += ff.DataLen
}

// Flush returns a Receive Rate option and indicates to receiveRate 
// that the next Ack-to-Ack window has begun
func (r *receiveRate) Flush(rtt int64) *ReceiveRateOption {
	now := time.Nanoseconds()
	if r.time0 > now || r.time1 > now {
		panic("receive rate time")
	}
	d0 := now - r.time0
	d1 := now - r.time1
	if d0 < d1 {
		panic("receive rate period")
	}
	if d1 < rtt {
		return &ReceiveRateOption{rate(r.data0, d0)}
	}
	rval := rate(r.data1, d1)
	r.data0, r.data1 = r.data1, 0
	r.time0, r.time1 = r.time1, now
	return &ReceiveRateOption{rval}
}

func rate(nbytes int, nsec int64) uint32 {
	sec := uint32(nsec / 1e9)
	if sec < 0 || nbytes < 0 {
		panic("receive rate, negative period")
	}
	if sec == 0 {
		return uint32(nbytes)
	}
	return uint32(nbytes) / sec
}
