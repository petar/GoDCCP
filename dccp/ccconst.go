// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
	"time"
)

// NewConstRateControlFunc creates a function that makes new Congestion Control 
// which sends packets at a constant rate of sendsPerSecond packets per second
func NewConstRateControlFunc (sendsPerSecond int64) NewCongestionControlFunc {
	return func() CongestionControl {
		return newConstRateControl(1e9 / sendsPerSecond)
	}
}

type constRateControl struct {
	Mutex
	every  int64 // Strobe every every nanoseconds
	strobe chan int
}

// How to close the congestion control
func newConstRateControl(every int64) *constRateControl {
	return &constRateControl{ every: every, strobe: make(chan int) }
}

func (cc *constRateControl) Start() {
	go func() {
		for {
			cc.Lock()
			if cc.strobe == nil {
				cc.Unlock()
				break
			}
			cc.strobe <- 1
			cc.Unlock()
			time.Sleep(cc.every)
		}
	}()
}

const CCID_CONST = 7

func (cc *constRateControl) GetID() byte { return CCID_CONST }

func (cc *constRateControl) GetCCMPS() int32 { return 1e9 }

func (cc *constRateControl) GetRTT() int64 { return RTT_DEFAULT }

func (cc *constRateControl) GetSWABF() (swaf int64, swbf int64) { return 20, 20 }

func (cc *constRateControl) OnWrite(htype byte, x bool, seqno int64) (ccval byte, options []Option) { return 0, nil }

func (cc *constRateControl) OnRead(htype byte, x bool, seqno int64, ccval byte, options []Option) os.Error { return nil }

func (cc *constRateControl) Strobe() os.Error {
	cc.Lock()
	defer cc.Unlock()
	if cc.strobe == nil {
		return os.EBADF
	}
	_, ok := <-cc.strobe 
	if !ok {
		return os.EBADF
	}
	return nil
}

func (cc *constRateControl) Close() os.Error { 
	cc.Lock()
	defer cc.Unlock()
	if cc.strobe != nil {
		close(cc.strobe) 
		cc.strobe = nil
		return nil
	}
	return os.EBADF
}
