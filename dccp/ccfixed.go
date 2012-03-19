// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

type CCFixed struct {

}

func (CCFixed) NewSender(run *Runtime, logger *Logger) SenderCongestionControl {
	return newFixedRateSenderControl(run, 1e9) // one packet per second. sendsPerSecond
}

func (CCFixed) NewReceiver(run *Runtime, logger *Logger) ReceiverCongestionControl {
	return newFixedRateReceiverControl(run)
}

// ---> Fixed-rate HC-Sender Congestion Control

type fixedRateSenderControl struct {
	run         *Runtime
	Mutex
	every       int64 // Strobe every every nanoseconds
	strobeRead  chan int
	strobeWrite chan int
}

func newFixedRateSenderControl(run *Runtime, every int64) *fixedRateSenderControl {
	strobe := make(chan int)
	return &fixedRateSenderControl{run: run, every: every, strobeRead: strobe, strobeWrite: strobe}
}

func (scc *fixedRateSenderControl) Open() {
	go func() {
		for {
			scc.Lock()
			if scc.strobeWrite == nil {
				scc.Unlock()
				break
			}
			scc.strobeWrite <- 1
			scc.Unlock()
			scc.run.Sleep(scc.every)
		}
	}()
}

const CCID_FIXED = 0xf

func (scc *fixedRateSenderControl) GetID() byte { return CCID_FIXED }

func (scc *fixedRateSenderControl) GetCCMPS() int32 { return 1e9 }

func (scc *fixedRateSenderControl) GetRTT() int64 { return RoundtripDefault }

func (scc *fixedRateSenderControl) OnWrite(ph *PreHeader) (ccval int8, options []*Option) {
	return 0, nil
}

func (scc *fixedRateSenderControl) OnRead(fb *FeedbackHeader) error { return nil }

func (scc *fixedRateSenderControl) OnIdle(now int64) error { return nil }

func (scc *fixedRateSenderControl) Strobe() {
	<-scc.strobeRead
}

func (scc *fixedRateSenderControl) SetHeartbeat(interval int64) {
}

func (scc *fixedRateSenderControl) Close() {
	scc.Lock()
	defer scc.Unlock()
	if scc.strobeWrite != nil {
		close(scc.strobeWrite)
		scc.strobeWrite = nil
	}
}

// ---> Fixed-rate HC-Receiver Congestion Control

type fixedRateReceiverControl struct{
	run *Runtime
}

func newFixedRateReceiverControl(run *Runtime) *fixedRateReceiverControl {
	return &fixedRateReceiverControl{ run: run }
}

func (rcc *fixedRateReceiverControl) Open() {}

func (rcc *fixedRateReceiverControl) GetID() byte { return CCID_FIXED }

func (rcc *fixedRateReceiverControl) OnWrite(ph *PreHeader) (options []*Option) { return nil }

func (rcc *fixedRateReceiverControl) OnRead(ff *FeedforwardHeader) error { return nil }

func (rcc *fixedRateReceiverControl) OnIdle(now int64) error { return nil }

func (rcc *fixedRateReceiverControl) Close() {}
