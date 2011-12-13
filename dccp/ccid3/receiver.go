// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)

func newReceiver(run *dccp.Runtime, logger *dccp.Logger) *receiver {
	return &receiver{ run: run, logger: logger }
}

// —————
// receiver is a CCID3 congestion control receiver
type receiver struct {
	run    *dccp.Runtime
	logger *dccp.Logger
	dccp.Mutex
	rttReceiver
	receiveRate
	lossReceiver

	open bool // Whether the CC is active

	lastWrite            int64  // The timestamp of the last call to OnWrite
	lastAck              int64  // The timestamp of the last call to OnWrite with an Ack packet type
	dataSinceAck         bool   // True if data packets have been received since the last Ack
	lastLossEventRateInv uint32 // The inverse loss event rate sent in the last Ack packet

	// The following fields are used to compute ElapsedTime options
	gsr          int64 // Greatest sequence number of packet received via OnRead
	gsrTimestamp int64 // Timestamp of packet with greatest sequence number received via OnRead

	// The greatest received value of the window counter since the last feedback message was sent
	lastCCVal byte
	// The window counter of the latest received packet. Used internally to update lastCCVal.
	latestCCVal byte
}

// GetID() returns the CCID of this congestion control algorithm
func (r *receiver) GetID() byte {
	return dccp.CCID3
}

// Open tells the Congestion Control that the connection has entered
// OPEN or PARTOPEN state and that the CC can now kick in.
func (r *receiver) Open() {
	r.Lock()
	defer r.Unlock()
	if r.open {
		panic("opening an open ccid3 receiver")
	}

	r.rttReceiver.Init(r.logger)
	r.receiveRate.Init()
	r.lossReceiver.Init(r.logger)
	r.open = true
	r.lastWrite = 0
	r.lastAck = 0
	r.dataSinceAck = false
	r.lastLossEventRateInv = UnknownLossEventRateInv

	r.gsr = 0
	r.gsrTimestamp = 0

	r.lastCCVal = 0
	r.latestCCVal = 0
}

func (r *receiver) makeElapsedTimeOption(ackNo int64, now int64) *dccp.ElapsedTimeOption {
	// The first Ack may be sent before receiver has had a chance to see a gsr, in which
	// case we return nil
	if r.gsr == 0 {
		return nil
	}
	if ackNo != r.gsr {
		panic("ccid3 receiver: GSR != AckNo")
	}
	elapsedNS := max64(0, now-r.gsrTimestamp)
	return &dccp.ElapsedTimeOption{dccp.TenUSFromNS(elapsedNS)}
}

// Conn calls OnWrite before a packet is sent to give CongestionControl
// an opportunity to add CCVal and options to an outgoing packet
func (r *receiver) OnWrite(ph *dccp.PreHeader) (options []*dccp.Option) {
	r.Lock()
	defer r.Unlock()
	rtt := r.rttReceiver.RTT(ph.Time)

	r.lastWrite = ph.Time
	if !r.open {
		return nil
	}

	switch ph.Type {
	case dccp.Ack, dccp.DataAck:
		// Record last Ack write separately from last writes (in general)
		r.lastAck = ph.Time
		r.dataSinceAck = false
		r.lastLossEventRateInv = r.lossReceiver.LossEventRateInv()
		r.lastCCVal = r.latestCCVal

		// Prepare feedback options, if we've seen packets before
		if r.gsr > 0 {
			opts := make([]*dccp.Option, 3)
			opts[0] = encodeOption(r.makeElapsedTimeOption(ph.AckNo, ph.Time))
			if opts[0] == nil {
				r.logger.E("r", "Warn", "ElapsedTime option encoding == nil", ph)
			}
			opts[1] = encodeOption(r.receiveRate.Flush(rtt, ph.Time))
			if opts[1] == nil {
				r.logger.E("r", "Warn", "ReceiveRate option encoding == nil", ph)
			}
			opts[2] = encodeOption(r.lossReceiver.LossIntervalsOption(ph.AckNo))
			if opts[2] == nil {
				r.logger.E("r", "Warn", "LossIntervals option encoding == nil", ph)
			}
			r.logger.E("r", "Info", fmt.Sprintf("Placed %d receiver opts", len(opts)), ph)
			return opts
		}
		r.logger.E("r", "Info", "OnWrite, not seen packs before", ph)
		return nil

	case dccp.Data /*, dccp.DataAck */:
		return nil
	default:
		return nil
	}
	panic("unreach")
}

// Conn calls OnRead after a packet has been accepted and validated
// If OnRead returns ErrDrop, the packet will be dropped and no further processing
// will occur. If the CC is not active, OnRead MUST return nil.
func (r *receiver) OnRead(ff *dccp.FeedforwardHeader) error {
	r.Lock()
	defer r.Unlock()
	if !r.open {
		return nil
	}

	// XXX: Must use circular arithmrtic here
	if ff.SeqNo > r.gsr {
		r.gsr = ff.SeqNo
		r.gsrTimestamp = ff.Time
	}

	if ff.Type == dccp.Data || ff.Type == dccp.DataAck {
		r.dataSinceAck = true
		r.latestCCVal = ff.CCVal
	}

	// Update RTT estimate
	r.rttReceiver.OnRead(ff.CCVal, ff.Time)
	rrtt, est := r.rttReceiver.RTT(ff.Time)
	r.logger.E("r", "rrtt", fmt.Sprintf("rRTT=%s est=%v", dccp.Nstoa(rrtt), est), ff, 
		dccp.LogArgs{"rtt": rrtt, "est": est})

	// Update receive rate
	r.receiveRate.OnRead(ff)

	// Update loss rate
	r.lossReceiver.OnRead(ff, r.rttReceiver.RTT(ff.Time))

	// Determine if feedback should be sent:

	// (Feedback-Condition-II) If the current calculated loss event rate is greater than its
	// previous value
	if r.lossReceiver.LossEventRateInv() < r.lastLossEventRateInv {
		return dccp.CongestionAck
	}

	// (Feedback-Condition-III) If receive window counter increases by 4 or more on a data
	// packet, since last time feedback was sent
	if ff.Type == dccp.Data || ff.Type == dccp.DataAck {
		if !lessWindowCounterMod(ff.CCVal, (r.lastCCVal+4)%WindowCounterMod) {
			return dccp.CongestionAck
		}
	}

	return nil
}

// OnIdle behaves identically to the same method of the HC-Sender CCID
func (r *receiver) OnIdle(now int64) error {
	r.Lock()
	defer r.Unlock()
	if !r.open {
		return nil
	}

	// Determine if an Ack packet should be sent:

	// (Feedback-Condition-I) If one (estimated) round-trip time time has expired since last Ack
	// AND data packets have been received in the meantime
	if r.dataSinceAck && now-r.lastWrite > r.rttReceiver.RTT(now) {
		return dccp.CongestionAck
	}

	return nil
}

// Close terminates the half-connection congestion control when it is not needed any longer
func (r *receiver) Close() {
	r.Lock()
	defer r.Unlock()
	r.open = false
}
