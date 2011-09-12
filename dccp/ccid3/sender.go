// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"github.com/petar/GoDCCP/dccp"
)

// —————
// sender is a CCID3 congestion control sender
type sender struct {
	dccp.Mutex
	rttSender
	windowCounter
	nofeedbackTimer
	segmentSize
	lossTracker
	rateCalculator
	strober

	open bool // Whether the CC is active
}

// GetID() returns the CCID of this congestion control algorithm
func (s *sender) GetID() byte { return dccp.CCID3 }

// GetCCMPS returns the Congestion Control Maximum Packet Size, CCMPS. Generally, PMTU <= CCMPS
// TODO: For the time being we use a fixed CCMPS
func (s *sender) GetCCMPS() int32 { return FixedSegmentSize }

// GetRTT returns the Round-Trip Time as measured by this CCID
func (s *sender) GetRTT() int64 { return s.rttSender.RTT() }

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
	s.nofeedbackTimer.Init()
	s.segmentSize.Init()
	s.segmentSize.SetMPS(FixedSegmentSize)
	s.lossTracker.Init()
	s.rateCalculator.Init(FixedSegmentSize)
	s.strober.Init()
	s.open = true
}

// Conn calls OnWrite before a packet is sent to give CongestionControl
// an opportunity to add CCVal and options to an outgoing packet
// If the CC is not active, OnWrite should return 0, nil.
func (s *sender) OnWrite(htype byte, x bool, seqno, ackno int64, now int64) (ccval byte, options []*dccp.Option) {
	s.Lock()
	defer s.Unlock()

	if !s.open {
		return 0, nil
	}

	return s.windowCounter.OnWrite(s.rttSender.RTT(), ???, now), nil
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
	?? // This check should probably be module specific. Are all modules bellow just for Ack packets?
	if fb.Type != dccp.Ack && fb.Type != dccp.DataAck {
		return nil
	}

	// Update the round-trip estimate
	rttChanged := s.rttSender.OnRead(fb)
	rtt := s.rttSender.RTT()

	// Window counter update
	s.windowCounter.OnRead(fb.AckNo)
	
	// Update the nofeedback timeout interval
	// t.nofeedbackTimer. ??

	// Update loss estimates
	// t.lossTracker.??

	// Update allowed sending rate
	// t.rateCalculator.??

	// Reset the nofeedback timer
	panic("?")
}

// Strobe blocks until a new packet can be sent without violating the congestion control
// rate limit. If the CC is not active, Strobe MUST return immediately.
func (s *sender) Strobe() {
	s.Lock()
	defer s.Unlock()

	if !s.open {
		return
	}

	s.strober.Strobe()
}

// OnIdle is called periodically, giving the CC a chance to:
// (a) Request a connection reset by returning a CongestionReset, or
// (b) Request the injection of an Ack packet by returning a CongestionAck
// NOTE: If the CC is not active, OnIdle MUST to return nil.
func (s *sender) OnIdle(now int64) os.Error {
	s.Lock()
	defer s.Unlock()
	
	panic("?")
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
