// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)

// RoundtripSample converts a roundtrip time in nanosecond to a floating-point time in milliseconds
func RoundtripSample(rtt int64) dccp.Sample {
	return dccp.NewSample(float64(rtt) / 1e6)
}

// senderRoundtripReporter ensures that the sender's RTT estimate is regularly sent to the receiver
type senderRoundtripReporter struct {
	lastReportTime int64
}

func (t *senderRoundtripReporter) Init() {
	t.lastReportTime = 0
}

func (t *senderRoundtripReporter) OnWrite(rtt int64, now int64) *dccp.Option {
	if rtt <= 0 || now - t.lastReportTime < rtt {
		return nil
	}
	t.lastReportTime = now
	return encodeOption(&RoundtripReportOption{ Roundtrip: dccp.TenMicroFromNano(rtt) })
}

// senderRoundtripEstimator is a data structure that estimates the RTT at the sender end.
type senderRoundtripEstimator struct {
	logger   *dccp.Logger
	estimate int64
	k        int					// The index of the next history cell to write in
	history  [SenderRoundtripHistoryLen]sendTime	// Circular array, recording departure times of last few packets
}

type sendTime struct {
	SeqNo int64
	Time  int64	// Time=0 indicates that the struct is nil
}

const (
	SenderRoundtripHistoryLen = 20 // How many timestamps of sent packets to remember
	SenderRoundtripWeightNew = 1
	SenderRoundtripWeightOld = 9
)

// Init resets the senderRoundtripEstimator object for new use
func (t *senderRoundtripEstimator) Init(logger *dccp.Logger) {
	t.logger = logger.Refine("senderRoundtripEstimator")
	t.estimate = 0
	t.k = 0
	for i, _ := range t.history {
		t.history[i] = sendTime{} // Zero Time indicates no data
	}
}

// Sender calls OnWrite for every packet sent.
func (t *senderRoundtripEstimator) OnWrite(seqNo int64, now int64) {
	t.history[t.k % SenderRoundtripHistoryLen] = sendTime{seqNo, now}
	t.k++
	// Keep k small
	if t.k > 100 * SenderRoundtripHistoryLen {
		t.k %= SenderRoundtripHistoryLen
	}
}

// find returns the sendTime of the packet with the given seqNo, if still in the history slice
func (t *senderRoundtripEstimator) find(seqNo int64) *sendTime {
	for i := 0; i < len(t.history); i++ {
		r := &t.history[i]
		if r.SeqNo == seqNo {
			return r
		}
	}
	return nil
}

// Sender calls OnRead for every arriving Ack packet. 
// OnRead returns true if the RTT estimate has changed.
func (t *senderRoundtripEstimator) OnRead(fb *dccp.FeedbackHeader) bool {

	// Read ElapsedTimeOption
	if fb.Type != dccp.Ack && fb.Type != dccp.DataAck {
		return false
	}
	var elapsed *dccp.ElapsedTimeOption
	for _, opt := range fb.Options {
		if elapsed = dccp.DecodeElapsedTimeOption(opt); elapsed != nil {
			break
		}
	}
	if elapsed == nil {
		t.logger.E(dccp.EventWarn, "Missing elapsed opt", fb)
		return false
	}

	// Update RTT estimate
	s := t.find(fb.AckNo)
	if s == nil {
		return false
	}
	elapsedNS := dccp.NanoFromTenMicro(elapsed.Elapsed) // Elapsed time at receiver in nanoseconds
	if elapsedNS < 0 {
		t.logger.E(dccp.EventWarn, "Invalid elapsed opt", fb)
		return false
	}
	est := (fb.Time - s.Time - elapsedNS) / 2
	if est <= 0 {
		t.logger.E(dccp.EventWarn, "Invalid elapsed opt", fb)
		return false
	}
	est_old := t.estimate
	if est_old == 0 {
		t.estimate = est
	} else {
		t.estimate = (est * SenderRoundtripWeightNew + est_old * SenderRoundtripWeightOld) / 
			(SenderRoundtripWeightNew + SenderRoundtripWeightOld)
	}
	t.logger.E(dccp.EventMatch, fmt.Sprintf("Elapsed —> RTT=%s", dccp.Nstoa(t.estimate)), fb, RoundtripSample(t.estimate))

	return true
}

// RTT returns the current round-trip time estimate in ns, or the default if no estimate is
// available due to shortage of samples. estimated is set if the RTT is estimated based on sample
// data (as opposed to being equal to a default value).
func (t *senderRoundtripEstimator) RTT() (rtt int64, estimated bool) {
	if t.estimate <= 0 {
		return dccp.RoundtripDefault, false
	}
	return t.estimate, true
}

// HasRTT returns true if senderRoundtripEstimator has enough sample data for an estimate
func (t *senderRoundtripEstimator) HasRTT() bool {
	return t.estimate > 0
}

// receiverRoundtripEstimator is a data structure that estimates the RTT at the receiver end.
// Instead of using the less precise algorithm described in RFC 4342, towards the end of Section
// 8.1, we simply record the RTT estimate calculated at the sender and communicated via an option.
type receiverRoundtripEstimator struct {
	logger *dccp.Logger

	// rtt equals the latest RTT estimate, or 0 otherwise
	rtt int64

	// rttTime is the time when RTT estimate was received
	rttTime int64
}

// Init initializes the RTT estimator
func (t *receiverRoundtripEstimator) Init(logger *dccp.Logger) {
	t.logger = logger.Refine("receiverRoundtripEstimator")
	t.rtt = 0
	t.rttTime = 0
}

// receiver calls OnRead every time a packet is received
// OnRead returns true, if the roundtrip estimate has changed
func (t *receiverRoundtripEstimator) OnRead(ff *dccp.FeedforwardHeader) bool {

	// Read RoundtripReportOption
	// Currently RoundtripReportOption is allowed on any packet type
	var report *RoundtripReportOption
	for _, opt := range ff.Options {
		if report = DecodeRoundtripReportOption(opt); report != nil {
			break
		}
	}
	if report == nil {
		t.logger.E(dccp.EventWarn, "Missing roundtrip report opt", ff)
		return false
	}

	// Sanity checks
	rtt := dccp.NanoFromTenMicro(report.Roundtrip)
	if rtt <= 0 || rtt > 30e9 {
		t.logger.E(dccp.EventWarn, "Invalid roundtrip report opt", ff)
		return false
	}

	// Update RTT estimate
	t.rtt, t.rttTime = rtt, ff.Time
	t.logger.E(dccp.EventMatch, fmt.Sprintf("Report —> RTT=%s", dccp.Nstoa(t.rtt)), ff, RoundtripSample(t.rtt))

	return true
}

// RTT returns the best available estimate of the round-trip time
func (t *receiverRoundtripEstimator) RTT(now int64) (rtt int64, estimated bool) {
	if t.rtt != 0 &&  now - t.rttTime < 1e9 {
		return t.rtt, true
	}
	return dccp.RoundtripDefault, false
}
