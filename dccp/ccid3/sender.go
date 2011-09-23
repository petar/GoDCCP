// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"github.com/petar/GoDCCP/dccp"
)

func newSender() *sender {
	return &sender{}
}

// —————
// sender is a CCID3 congestion control sender
type sender struct {
	dccp.DLog
	strober
	dccp.Mutex // Locks all fields below
	rttSender
	windowCounter
	nofeedbackTimer
	segmentSize
	lossTracker
	rateCalculator
	open bool // Whether the CC is active
}

// GetID() returns the CCID of this congestion control algorithm
func (s *sender) GetID() byte { return dccp.CCID3 }

// GetCCMPS returns the Congestion Control Maximum Packet Size, CCMPS. Generally, PMTU <= CCMPS
// TODO: For the time being we use a fixed CCMPS
func (s *sender) GetCCMPS() int32 { return FixedSegmentSize }

// GetRTT returns the Round-Trip Time as measured by this CCID
func (s *sender) GetRTT() int64 { 
	s.Lock()
	defer s.Unlock()
	rtt, _ := s.rttSender.RTT() 
	return rtt
}

// Open tells the Congestion Control that the connection has entered
// OPEN or PARTOPEN state and that the CC can now kick in. Before the
// call to Open and after the call to Close, the Strobe function is
// expected to return immediately.
func (s *sender) Open() {
	s.Lock()
	defer s.Unlock()
	if s.open {
		panic("opening an open ccid3 sender")
	}
	s.windowCounter.Init()
	s.rttSender.Init()
	rtt, _ := s.rttSender.RTT()
	s.nofeedbackTimer.Init()
	s.segmentSize.Init()
	s.segmentSize.SetMPS(FixedSegmentSize)
	s.lossTracker.Init()
	s.rateCalculator.Init(s.DLog, FixedSegmentSize, rtt)
	s.strober.Init((int64(s.rateCalculator.X())*64) / FixedSegmentSize)
	s.open = true
}

// Conn calls OnWrite before a packet is sent to give CongestionControl
// an opportunity to add CCVal and options to an outgoing packet
// If the CC is not active, OnWrite should return 0, nil.
func (s *sender) OnWrite(ph *dccp.PreHeader) (ccval byte, options []*dccp.Option) {
	s.Lock()
	defer s.Unlock()

	if !s.open {
		return 0, nil
	}

	s.nofeedbackTimer.OnWrite(ph)

	rtt, _ := s.rttSender.RTT()

	return s.windowCounter.OnWrite(rtt, ph.SeqNo, ph.Time), nil
}

// Conn calls OnRead after a packet has been accepted and validated
// If OnRead returns ErrDrop, the packet will be dropped and no further processing
// will occur. If OnRead returns ResetError, the connection will be reset.
// If the CC is not active, OnRead MUST return nil.
func (s *sender) OnRead(fb *dccp.FeedbackHeader) os.Error {
	s.Lock()
	defer s.Unlock()

	if !s.open {
		return nil
	}
	// Only feedback packets (Ack or DataAck) trigger updates in the congestion control
	if fb.Type != dccp.Ack && fb.Type != dccp.DataAck {
		return nil
	}

	// Update the round-trip estimate
	s.rttSender.OnRead(fb)
	rtt, rttEstimated := s.rttSender.RTT()

	// Update the nofeedback timeout interval and reset the timer
	s.nofeedbackTimer.OnRead(rtt, rttEstimated, fb)

	// Window counter update
	s.windowCounter.OnRead(fb.AckNo)
	
	// Update loss estimates
	lossFeedback, err := s.lossTracker.OnRead(fb)
	if err != nil {
		s.DLog.Emit("?", "feedback packet with corrupt loss option")
		return nil
	}

	// Update allowed sending rate
	xrecv, err := readReceiveRate(fb)
	if err != nil {
		s.DLog.Emit("?", "feedback packet with corrupt receive rate option")
		return nil
	}
	xf := &XFeedback{
		Now:          fb.Time,
		SS:           FixedSegmentSize,
		XRecv:        xrecv,
		RTT:          rtt,
		LossFeedback: lossFeedback,
	}
	x := s.rateCalculator.OnRead(xf)

	// Update rate at strober
	s.strober.SetRate((int64(x)*64) / FixedSegmentSize)

	return nil
}

func readReceiveRate(fb *dccp.FeedbackHeader) (xrecv uint32, err os.Error) {
	if fb.Type != dccp.Ack && fb.Type != dccp.DataAck {
		return 0, ErrNoAck
	}
	var receiveRate *ReceiveRateOption
	for _, opt := range fb.Options {
		if receiveRate = DecodeReceiveRateOption(opt); receiveRate != nil {
			break
		}
	}
	if receiveRate == nil {
		return 0, ErrMissingOption
	}
	return receiveRate.Rate, nil
}

// Strobe blocks until a new packet can be sent without violating the congestion control
// rate limit. If the CC is not active, Strobe MUST return immediately.
func (s *sender) Strobe() {
	s.Lock()
	open := s.open
	s.Unlock()

	if !open {
		return
	}

	s.strober.Strobe()
}

// OnIdle is called periodically. If the CC is not active, OnIdle MUST to return nil.
func (s *sender) OnIdle(now int64) os.Error {
	s.Lock()
	defer s.Unlock()

	if !s.open {
		return nil
	}
	
	if s.nofeedbackTimer.IsExpired(now) {
		idleSince, nofeedbackSet := s.nofeedbackTimer.GetIdleSinceAndReset()
		_, hasRTT := s.rttSender.RTT()
		s.rateCalculator.OnNoFeedback(now, hasRTT, idleSince, nofeedbackSet)
		s.nofeedbackTimer.Reset(now)
	}

	return nil
}

// Close terminates the half-connection congestion control when it is not needed any longer
func (s *sender) Close() {
	s.Lock()
	defer s.Unlock()
	if !s.open {
		panic("closing a non-open ccid3 sender")
	}
	s.open = false
}

func (s *sender) SetDLog(dlog dccp.DLog) {
	s.DLog.Init(dlog, "sender")
}
