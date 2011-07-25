// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"log"
	"os"
	"github.com/petar/GoDCCP/dccp"
)

// —————
// receiver is a CCID3 congestion control receiver
type receiver struct {
	dccp.Mutex
	rttReceiver
	receiveRate
	lossEvents

	open                 bool     // Whether the CC is active

	lastWrite            int64    // The timestamp of the last call to OnWrite
	lastAck              int64    // The timestamp of the last call to OnWrite with an Ack packet type
	dataSinceAck         bool     // True if data packets have been received since the last Ack
	lastLossEventRateInv uint32   // The inverse loss event rate sent in the last Ack packet

	// The following fields are used to compute ElapsedTime options
	gsr                  int64    // Greatest sequence number of packet received via OnRead
	gsrTimestamp         int64    // Timestamp of packet with greatest sequence number received via OnRead

	// The greatest received value of the window counter since the last feedback message was sent
	lastCCVal            byte
	// The window counter of the latest received packet. Used internally to update lastCCVal.
	latestCCVal          byte
}

// GetID() returns the CCID of this congestion control algorithm
func (r *receiver) GetID() byte { return dccp.CCID3 }

// Open tells the Congestion Control that the connection has entered
// OPEN or PARTOPEN state and that the CC can now kick in.
func (r *receiver) Open() {
	r.Lock()
	defer r.Unlock()
	if r.open {
		panic("opening an open ccid3 receiver")
	}
	r.rttReceiver.Init()
	r.receiveRate.Init()
	r.lossEvents.Init()
	r.open = true
	r.lastWrite = 0
	r.lastAck = 0
	r.dataSinceAck = false
	r.lastLossEventRateInv = UnknownLossEventRate

	r.gsr = 0
	r.gsrTimestamp = 0

	r.lastCCVal = 0
	r.latestCCVal = 0
}

func (r *receiver) makeElapsedTimeOption(ackNo int64, now int64) *dccp.ElapsedTimeOption {
	if ackNo != r.gsr {
		log.Printf("CCID3 GSR != packet AckNo")
		return nil
	}
	elapsedNS := max64(0, now - r.gsrTimestamp)
	return &dccp.ElapsedTimeOption{ dccp.Nano2TenMicroTimeLen(elapsedNS) }
}

// Conn calls OnWrite before a packet is sent to give CongestionControl
// an opportunity to add CCVal and options to an outgoing packet
func (r *receiver) OnWrite(htype byte, x bool, seqno, ackno int64, now int64) (options []*dccp.Option) {
	r.Lock()
	defer r.Unlock()
	rtt := r.rttReceiver.RTT()

	r.lastWrite = now
	if !r.open {
		return nil
	}
	
	switch htype {
	case dccp.Ack:
		// Record last Ack write separately from last writes (in general)
		r.lastAck = now
		r.dataSinceAck = false
		r.lastLossEventRateInv = r.lossEvents.LossEventRateInv()
		r.lastCCVal = r.latestCCVal

		// Prepare feedback options
		opts := make([]*dccp.Option, 3)
		opts[0] = encodeOption(r.makeElapsedTimeOption(ackno, now))
		opts[1] = encodeOption(r.receiveRate.Flush(rtt))
		opts[2] = encodeOption(r.lossEvents.Option(ackno))
		if opts[0] == nil {
			opts = opts[1:3]
		}
		return opts

	case dccp.Data, dccp.DataAck:
		return nil
	default:
		return nil
	}
	panic("unreach")
}

// Conn calls OnRead after a packet has been accepted and validated
// If OnRead returns ErrDrop, the packet will be dropped and no further processing
// will occur. 
// NOTE: If the CC is not active, OnRead MUST return nil.
func (r *receiver) OnRead(ff *dccp.FeedforwardHeader) os.Error {
	r.Lock()
	defer r.Unlock()
	if !r.open {
		return nil
	}

	if ff.Type == dccp.Data || ff.Type == dccp.DataAck {
		r.dataSinceAck = true
		r.latestCCVal = ff.CCVal
	}
	r.rttReceiver.OnRead(ff.CCVal)
	r.receiveRate.OnRead(ff)
	r.lossEvents.OnRead(ff, r.rttReceiver.RTT())

	// Determine if feedback should be sent:

	// (Feedback-Condition-II) If the current calculated loss event rate is greater than its
	// previous value
	if r.lossEvents.LossEventRateInv() < r.lastLossEventRateInv {
		return dccp.CongestionAck
	}

	// (Feedback-Condition-III) If receive window counter increases by 4 or more on a data
	// packet, since last time feedback was sent
	if ff.Type == dccp.Data || ff.Type == dccp.DataAck {
		if !lessModWCTRMAX(ff.CCVal, r.lastCCVal+4) {
			return dccp.CongestionAck
		}
	}

	return nil
}

// OnIdle behaves identically to the same method of the HC-Sender CCID
func (r *receiver) OnIdle(now int64) os.Error {
	r.Lock()
	defer r.Unlock()
	if !r.open {
		return nil
	}

	// Determine if an Ack packet should be sent:

	// (Feedback-Condition-I) If one (estimated) round-trip time time has expired since last Ack
	// AND data packets have been received in the meantime
	if r.dataSinceAck && now - r.lastWrite > r.rttReceiver.RTT() {
		return dccp.CongestionAck
	}

	return nil
}

// Close terminates the half-connection congestion control when it is not needed any longer
func (r *receiver) Close() {
	r.Lock()
	defer r.Unlock()
	if !r.open {
		panic("closing a closed ccid3 receiver")
	}
	r.open = false
}
