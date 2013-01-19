// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"github.com/petar/GoDCCP/dccp"
)

// receiverRateCalculator keeps track of the data receive rate at the CCID3 receiver,
// and produces Receive Rate options for outgoing feedback packets.
// It's function is specified in RFC 4342, Section 8.3.
//
// XXX: Section 8.1, on the other hand, seems to suggest an alternative
// mechanism for computing receive rate, based on the window counter values
// in CCVal.
type receiverRateCalculator struct {
	data0, data1 int
	time0, time1 int64
}

func (r *receiverRateCalculator) Init() {
	r.time0, r.time1 = 0, 0
	r.data0, r.data1 = 0, 0
}

// OnRead is called to let the receiverRateCalculator know that data has been received.
// The ccval window counter value is not used in the current rate receiver algorithm
// explicitly. It is used implicitly in that the RTT estimate is based on these values.
func (r *receiverRateCalculator) OnRead(ff *dccp.FeedforwardHeader) {
	if ff.Type != dccp.Data && ff.Type != dccp.DataAck {
		return
	}
	if r.time0 == 0 || r.time1 == 0 {
		r.time0 = ff.Time
		r.time1 = ff.Time
	}
	r.data0 += ff.DataLen
	r.data1 += ff.DataLen
}

// Flush returns a Receive Rate option and indicates to receiverRateCalculator 
// that the next Ack-to-Ack window has begun
func (r *receiverRateCalculator) Flush(rtt int64, timeWrite int64) *ReceiveRateOption {
	if r.time0 > timeWrite || r.time1 > timeWrite {
		panic("receive rate time")
	}
	d0 := timeWrite - r.time0
	d1 := timeWrite - r.time1
	if d0 < d1 {
		panic("receive rate period")
	}
	if d1 < rtt {
		return &ReceiveRateOption{rate(r.data0, d0)}
	}
	rval := rate(r.data0, d0)
	r.data0, r.data1 = r.data1, 0
	r.time0, r.time1 = r.time1, timeWrite
	return &ReceiveRateOption{rval}
}

// Given data volume in bytes and time duration in nanoseconds, rate returns the
// corresponding rate in bytes per second
func rate(nbytes int, nsec int64) uint32 {
	if nbytes < 0 || nsec <= 0 {
		panic("receive rate, negative bytes or time")
	}
	return uint32((int64(nbytes)*1e9)/nsec)
}
