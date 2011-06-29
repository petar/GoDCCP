// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"time"
	"github.com/petar/GoDCCP/dccp"
)


//             0         10        20        30        40  44
//             |         |         |         |         |   |
//             ----------*--------***-*--------*----------*-
//             \________/\_______/\___________/\_________/
//                 L0       L1         L2           L3


type lossEvents struct {

	// pastHeaders keeps track of the last NDUPACK headers to overcome network re-ordering
	pastHeaders [NDUPACK]*headerEssence

	// pastIntervals keeps the most recent NINTERVAL+1 finalized loss intervals
	pastIntervals [NINTERVAL+1]*LossInterval
	// nIntervals equals the total number of intervals pushed onto pastIntervals so far
	nIntervals    int64
}

type headerEssence struct {
	Type    byte
	X       bool
	SeqNo   int64
	CCVal   byte
	Options []*dccp.Options
}

const NINTERVAL = 8

// Init initializes/resets the lossEvents instance
func (t *lossEvents) Init() {}

// PushPopHeader places the newly arrived header he into pastHeaders and 
// returns potentially another header (if available) whose SeqNo is sooner.
// Every header is returned exactly once.
func (t *lossEvents) pushPopHeader(he *headerEssence) *headerEssence {
	var popSeqNo int64 = dccp.SEQNOMAX+1
	var pop int
	for i, ge := range t.pastHeaders {
		if ge == nil {
			t.pastHeaders[i] = he
			return nil
		}
		if ge.SeqNo < popSeqNo {
			pop = i
		}
	}
	r := t.pastHeaders[i]
	t.pastHeaders[i] = he
	return r
}

// receiver calls OnRead every time a new packet arrives
func (t *lossEvents) OnRead(htype byte, x bool, seqno int64, ccval byte, options []*dccp.Option) os.Error {
	he := t.pushPopHeader(&headerEssence{
		Type:    htype,
		X:       x,
		SeqNo:   seqno,
		CCVal:   ccval,
		Options: options,
	})
	?
}

// pushInterval saves li as the most recent finalized loss interval
func (t *lossEvents) pushInterval(li *LossInterval) {
	t.pastIntervals[int(t.nIntervals % len(t.pastIntervals))] = li
	t.nIntervals++
}

// listIntervals lists the available finalized loss from most recent to least
func (t *lossEvents) listIntervals() []*LossInterval {
	l := len(t.pastIntervals)
	k = int(min64(t.nIntervals, int64(l)))
	// OPT: This slice allocation can be avoided by using a fixed instance
	r := make([]*LossInterval, k)
	p := int(t.nIntervals % int64(l)) + l
	for i := 0; i < k; i++ {
		p--
		r[i] = t.pastIntervals[p % l]
	}
	return r
}

func min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// Option returns the Loss Intervals option, representing the current state
func (t *lossEvents) Option() *Option {
	return &LossIntervalsOption{
		SkipLength:    ?,
		LossIntervals: t.listIntervals(),
	}
}

