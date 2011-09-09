// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	//"fmt"
	//"rand"
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

type ffRateMaker struct {
	gap int64  // time between adjacent packets
	ppr int64  // packets per rtt
	t   int64  // time counter
	s   int64  // sequence number counter
}

func (rm *ffRateMaker) Init(gap int64, ppr int64) {
	rm.gap = gap
	rm.ppr = ppr
	rm.t = 1e11
	rm.s = 20000
}

func (rm *ffRateMaker) RTT() int64 { return rm.gap * rm.ppr }

func (rm *ffRateMaker) PPR() int64 { return rm.ppr }

func (rm *ffRateMaker) Next() *dccp.FeedforwardHeader {
	ff := &dccp.FeedforwardHeader{
		Type:    dccp.Data,
		X:       true,
		SeqNo:   rm.s,
		CCVal:   0,
		Options: nil,
		Time:    rm.t,
	}
	rm.s++
	rm.t += rm.gap
	return ff
}

type lossHistory struct {
	h []*LossInterval
}

func (h *lossHistory) Add(i *LossInterval) {
	h.h = append(h.h, i)
}

func (h *lossHistory) Check(tail []*LossInterval) bool {
	if len(tail) > len(h.h) {
		return false
	}
	for i, g := range tail {
		j := len(h.h)-1-i
		if g.LossLength != h.h[j].LossLength ||
			g.LosslessLength != h.h[j].LosslessLength ||
			g.DataLength != h.h[j].DataLength {
			return false
		}
	}
	return true
}

func TestLossEvents(t *testing.T) {
	var le lossReceiver
	le.Init()

	var rm ffRateMaker
	rm.Init(1e10, 10)
	
	var h lossHistory
	le.OnRead(rm.Next(), rm.RTT())
	for q := 0; q < 10; q++ {
		for i := 0; i < NDUPACK; i++ {
			le.OnRead(rm.Next(), rm.RTT())
		}

		rm.Next()

		for i := 0; i < 3*int(rm.PPR()); i++ {
			le.OnRead(rm.Next(), rm.RTT())
		}

		?

		un := le.currentInterval()
		if un == nil {
			t.Errorf("expecting non-nil current interval")
		}

		expLossLen := uint32(1)
		expLosslessLen := uint32(3*int(rm.PPR())-NDUPACK)
		expDataLen := expLossLen + expLosslessLen
		if un.LossLength != expLossLen || un.LosslessLength != expLosslessLen || un.DataLength != expDataLen {
			t.Errorf("expect %d,%d,%d; got %d,%d,%d", 
				 expLossLen, expLosslessLen, expDataLen,
				 un.LossLength, un.LosslessLength, un.DataLength)
		}
		// fmt.Printf("%d —— %d == %d\n", un.LossLength, un.LosslessLength, un.DataLength)
	}
}
