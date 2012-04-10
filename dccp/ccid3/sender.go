// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)

func newSender(run *dccp.Runtime, amb *dccp.Amb) *sender {
	return &sender{ run: run, amb: amb.Refine("sender") }
}

// sender implements a CCID3 congestion control sender.
// It conforms to dccp.SenderCongestionControl.
type sender struct {
	run *dccp.Runtime
	amb *dccp.Amb
	senderStrober
	dccp.Mutex // Locks all fields below
	senderRoundtripEstimator
	senderRoundtripReporter
	senderWindowCounter
	senderNoFeedbackTimer
	senderSegmentSize
	senderLossTracker
	senderRateCalculator
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
	rtt, _ := s.senderRoundtripEstimator.RTT()
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
	s.senderWindowCounter.Init()
	s.senderRoundtripEstimator.Init(s.amb)
	rtt, _ := s.senderRoundtripEstimator.RTT()
	s.senderRoundtripReporter.Init()
	s.senderNoFeedbackTimer.Init()
	s.senderSegmentSize.Init()
	s.senderSegmentSize.SetMPS(FixedSegmentSize)
	s.senderLossTracker.Init(s.amb)
	s.senderRateCalculator.Init(s.amb, FixedSegmentSize, rtt)
	s.senderStrober.Init(s.run, s.amb, s.senderRateCalculator.X(), FixedSegmentSize)
	s.open = true
}

// Conn calls OnWrite before a packet is sent to give CongestionControl
// an opportunity to add CCVal and options to an outgoing packet
// If the CC is not active, OnWrite should return 0, nil.
func (s *sender) OnWrite(ph *dccp.PreHeader) (ccval int8, options []*dccp.Option) {
	s.Lock()
	defer s.Unlock()

	if !s.open {
		return 0, nil
	}

	s.senderNoFeedbackTimer.OnWrite(ph)

	s.senderRoundtripEstimator.OnWrite(ph.SeqNo, ph.Time)
	rtt, _ := s.senderRoundtripEstimator.RTT()

	ccval = s.senderWindowCounter.OnWrite(rtt, ph.SeqNo, ph.Time)
	s.amb.E(dccp.EventInfo, fmt.Sprintf("CCVAL=%d", ccval))

	reportOpt := s.senderRoundtripReporter.OnWrite(rtt, ph.Time)
	if reportOpt != nil {
		options = []*dccp.Option{ reportOpt }
	}

	return ccval, options
}

// Conn calls OnRead after a packet has been accepted and validated
// If OnRead returns ErrDrop, the packet will be dropped and no further processing
// will occur. If OnRead returns ResetError, the connection will be reset.
// If the CC is not active, OnRead MUST return nil.
func (s *sender) OnRead(fb *dccp.FeedbackHeader) error {
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
	s.senderRoundtripEstimator.OnRead(fb)
	rtt, rttEstimated := s.senderRoundtripEstimator.RTT()

	// Update the nofeedback timeout interval and reset the timer
	s.senderNoFeedbackTimer.OnRead(rtt, rttEstimated, fb)

	// Window counter update
	s.senderWindowCounter.OnRead(fb.AckNo)

	// Update loss estimates
	lossFeedback, err := s.senderLossTracker.OnRead(fb)
	if err != nil {
		return nil
	}

	// Update allowed sending rate
	xrecv, err := readReceiveRate(fb)
	if err != nil {
		s.amb.E(dccp.EventWarn, "Feedback packet with corrupt receive rate option", fb)
		return nil
	}
	xf := &XFeedback{
		Now:          fb.Time,
		SS:           FixedSegmentSize,
		XRecv:        xrecv,
		RTT:          rtt,
		LossFeedback: lossFeedback,
	}
	x := s.senderRateCalculator.OnRead(xf)
	// Flag "FixRate", if present, enforces a fixed send rate in strobes per 64 seconds
	flagFixRate, flagFixRatePresent := s.amb.Flags().GetUint32("FixRate")
	if flagFixRatePresent {
		s.senderStrober.SetRate(flagFixRate, FixedSegmentSize)
	} else {
		s.senderStrober.SetRate(x, FixedSegmentSize)
	}

	return nil
}

func readReceiveRate(fb *dccp.FeedbackHeader) (xrecv uint32, err error) {
	if fb.Type != dccp.Ack && fb.Type != dccp.DataAck {
		return 0, ErrNoAck
	}
	var receiverRateCalculator *ReceiveRateOption
	for _, opt := range fb.Options {
		if receiverRateCalculator = DecodeReceiveRateOption(opt); receiverRateCalculator != nil {
			break
		}
	}
	if receiverRateCalculator == nil {
		return 0, ErrMissingOption
	}
	return receiverRateCalculator.Rate, nil
}

// Strobe blocks until a new packet can be sent without violating the congestion control
// rate limit. If the CC is not active, Strobe MUST return immediately.
func (s *sender) Strobe() {
	s.Lock()
	open := s.open
	s.Unlock()

	if !open {
		s.amb.E(dccp.EventInfo, "Strobe immediate")
		return
	}

	s.senderStrober.Strobe()
}

// OnIdle is called periodically. If the CC is not active, OnIdle MUST to return nil.
func (s *sender) OnIdle(now int64) error {
	s.Lock()
	defer s.Unlock()

	if !s.open {
		return nil
	}

	if s.senderNoFeedbackTimer.IsExpired(now) {
		idleSince, nofeedbackSet := s.senderNoFeedbackTimer.GetIdleSinceAndReset()
		_, hasRTT := s.senderRoundtripEstimator.RTT()

		x := s.senderRateCalculator.OnNoFeedback(now, hasRTT, idleSince, nofeedbackSet)
		// Flag "FixRate" described above
		flagFixRate, flagFixRatePresent := s.amb.Flags().GetUint32("FixRate")
		if flagFixRatePresent {
			s.senderStrober.SetRate(flagFixRate, FixedSegmentSize)
		} else {
			s.senderStrober.SetRate(x, FixedSegmentSize)
		}

		s.senderNoFeedbackTimer.Reset(now)
	}

	return nil
}

// SetHeartbeat advices the CCID of the desired frequency of heartbeat packets.  A heartbeat
// interval value of zero indicates that no heartbeat is needed.
func (s *sender) SetHeartbeat(interval int64) {
	panic("un")
}

// Close terminates the half-connection congestion control when it is not needed any longer
func (s *sender) Close() {
	s.Lock()
	defer s.Unlock()
	s.open = false
}
