// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"math"
	"github.com/petar/GoDCCP/dccp"
)

// CCID3-specific options
const (
	OptionLossEventRate   = 192
	OptionLossIntervals   = 193
	OptionReceiveRate     = 194
	// OptionLossDigest and OptionRoundtripReport ARE NOT part of CCID3. They are our own extension.
	OptionLossDigest      = 210
	OptionRoundtripReport = 150
)

// Unencoded option is a type that knows how to encode itself into a dccp.Option
type UnencodedOption interface {
	Encode() (*dccp.Option, error)
}

func encodeOption(u UnencodedOption) *dccp.Option {
	if u == nil {
		return nil
	}
	opt, err := u.Encode()
	if err != nil {
		panic("problem encoding unencoded option")
	}
	return opt
}

// RFC 4342, Section 8.5
type LossEventRateOption struct {
	// RateInv is the inverse of the loss event rate, rounded UP, as calculated by the receiver.
	// It is actually calculated as data packets per loss interval.
	RateInv uint32
}

const (
	// UnknownLossEventRateInv is the maximum representable loss event rate inverse, 
	// and therefore corresponds to the minimum representable loss event rate.
	// In the CCID protocol, this constant is interpreted as 'no loss events detected'.
	// Its numerical value allows it be used safely in comparison operations.
	UnknownLossEventRateInv = math.MaxUint32
)

func DecodeLossEventRateOption(opt *dccp.Option) *LossEventRateOption {
	if opt.Type != OptionLossEventRate || len(opt.Data) != 4 {
		return nil
	}
	return &LossEventRateOption{RateInv: dccp.DecodeUint32(opt.Data[0:4])}
}

func (opt *LossEventRateOption) Encode() (*dccp.Option, error) {
	d := make([]byte, 4)
	dccp.EncodeUint32(opt.RateInv, d)
	return &dccp.Option{
		Type:      OptionLossEventRate,
		Data:      d,
		Mandatory: false,
	}, nil
}

// LossIntervalsOption is described in RFC 4342, Section 8.6.
// Intervals are listed in reverse chronological order.
// Loss interval sequence numbers are delta encoded starting from the Acknowledgement
// Number.  Therefore, Loss Intervals options MUST NOT be sent on packets without an
// Acknowledgement Number, and any Loss Intervals options received on such packets MUST be
// ignored.
type LossIntervalsOption struct {
	// SkipLength indicates the number of packets up to and including the Acknowledgement Number
	// that are not part of any Loss Interval. It must be less than or equal to NDUPACK = 3
	SkipLength    byte
	LossIntervals []*LossInterval
}

const (
	MaxLossIntervals      = 28
	NDUPACK               = 3
	lossIntervalFootprint = 9
)

func DecodeLossIntervalsOption(opt *dccp.Option) *LossIntervalsOption {
	if opt.Type != OptionLossIntervals || len(opt.Data) < 1 {
		return nil
	}
	k, r := (len(opt.Data)-1)/lossIntervalFootprint, (len(opt.Data)-1)%lossIntervalFootprint
	if k > MaxLossIntervals || r != 0 {
		return nil
	}
	skip := dccp.DecodeUint8(opt.Data[0:1])
	intervals := make([]*LossInterval, k)
	for i := 0; i < k; i++ {
		start := 1 + lossIntervalFootprint*i
		intervals[i] = decodeLossInterval(opt.Data[start : start+lossIntervalFootprint])
		if intervals[i] == nil {
			return nil
		}
	}
	return &LossIntervalsOption{
		SkipLength:    skip,
		LossIntervals: intervals,
	}
}

func (opt *LossIntervalsOption) Encode() (*dccp.Option, error) {
	if opt.SkipLength > NDUPACK {
		return nil, dccp.ErrOverflow
	}
	if len(opt.LossIntervals) > MaxLossIntervals {
		return nil, dccp.ErrOverflow
	}
	d := make([]byte, 1+lossIntervalFootprint*len(opt.LossIntervals))
	dccp.EncodeUint8(opt.SkipLength, d[0:1])
	for i, lossInterval := range opt.LossIntervals {
		j := 1 + i*lossIntervalFootprint
		if lossInterval.encode(d[j:j+lossIntervalFootprint]) != nil {
			return nil, nil
		}
	}
	return &dccp.Option{
		Type:      OptionLossIntervals,
		Data:      d,
		Mandatory: false,
	}, nil
}

// LossInterval describes an individual loss interval, RFC 4342, Section 8.6.2
type LossInterval struct {
	LosslessLength uint32 // Lossless Length, a 24-bit number, RFC 4342, Section 8.6.1
	LossLength     uint32 // Loss Length, a 23-bit number, RFC 4342, Section 8.6.1
	DataLength     uint32 // Data Length, a 24-bit number, RFC 4342, Section 8.6.1.
	// Specifies loss interval's data length, as defined in Section 6.1.1.
	ECNNonceEcho bool // ECN Nonce Echo, RFC 4342, Section 8.6.1
}

const _24thBit = 1 << 23

// SeqLen returns the sequence length of the loss interval
func (li *LossInterval) SeqLen() uint32 {
	return li.LosslessLength + li.LossLength
}

func (li *LossInterval) encode(p []byte) error {
	if len(p) != 9 {
		return dccp.ErrSize
	}
	if !dccp.FitsIn24Bits(uint64(li.LosslessLength)) ||
		!dccp.FitsIn23Bits(uint64(li.LossLength)) ||
		!dccp.FitsIn24Bits(uint64(li.DataLength)) {
		return dccp.ErrOverflow
	}

	dccp.EncodeUint24(li.LosslessLength, p[0:3])
	l := li.LossLength
	if li.ECNNonceEcho {
		l |= _24thBit
	}
	dccp.EncodeUint24(l, p[3:6])
	dccp.EncodeUint24(li.DataLength, p[6:9])

	return nil
}

func decodeLossInterval(p []byte) *LossInterval {
	li := &LossInterval{}
	li.LosslessLength = dccp.DecodeUint24(p[0:3])
	li.LossLength = dccp.DecodeUint24(p[3:6])
	li.ECNNonceEcho = (li.LossLength&_24thBit != 0)
	li.LossLength &= ^uint32(_24thBit)
	li.DataLength = dccp.DecodeUint24(p[6:9])
	return li
}

// ReceiveRateOption is described in RFC 4342, Section 8.3.
type ReceiveRateOption struct {
	// The rate at which receiver has received data since it last send an Ack; in bytes per second
	Rate uint32
}

func DecodeReceiveRateOption(opt *dccp.Option) *ReceiveRateOption {
	if opt.Type != OptionReceiveRate || len(opt.Data) != 4 {
		return nil
	}
	return &ReceiveRateOption{Rate: dccp.DecodeUint32(opt.Data[0:4])}
}

func (opt *ReceiveRateOption) Encode() (*dccp.Option, error) {
	d := make([]byte, 4)
	dccp.EncodeUint32(opt.Rate, d)
	return &dccp.Option{
		Type:      OptionReceiveRate,
		Data:      d,
		Mandatory: false,
	}, nil
}

// The LossDigest option directly carries, from the receiver to the sender, the types of loss
// information that a CCID3 sender would have to reconstruct from the LossIntervals option.  This is
// an extension to the RFC specification.
type LossDigestOption struct {
	// RateInv is the inverse of the loss event rate, rounded UP, as calculated by the receiver.
	// A value of UnknownLossEventRateInv indicates that no loss events have been observed.
	RateInv uint32

	// NewLoss indicates how many new loss events are reported by the feedback packet carrying this option
	NewLossCount byte
}

func DecodeLossDigestOption(opt *dccp.Option) *LossDigestOption {
	if opt.Type != OptionLossDigest || len(opt.Data) != 5 {
		return nil
	}
	return &LossDigestOption{
		RateInv:      dccp.DecodeUint32(opt.Data[0:4]),
		NewLossCount: dccp.DecodeUint8(opt.Data[4:5]),
	}
}

func (opt *LossDigestOption) Encode() (*dccp.Option, error) {
	d := make([]byte, 5)
	dccp.EncodeUint32(opt.RateInv, d)
	dccp.EncodeUint8(opt.NewLossCount, d[4:])
	return &dccp.Option{
		Type:      OptionLossDigest,
		Data:      d,
		Mandatory: false,
	}, nil
}

// RoundtripReportOption is used by the sender to communicate its RTT estimate to the receiver.
type RoundtripReportOption struct {
	// The Roundtrip estimate is given in ten microsecond units, similarly to the
	// ElapsedTimeOption
	Roundtrip uint32
}

func DecodeRoundtripReportOption(opt *dccp.Option) *RoundtripReportOption {
	if opt.Type != OptionRoundtripReport || len(opt.Data) != 4 {
		return nil
	}
	return &RoundtripReportOption{Roundtrip: dccp.DecodeUint32(opt.Data[0:4])}
}

func (opt *RoundtripReportOption) Encode() (*dccp.Option, error) {
	d := make([]byte, 4)
	dccp.EncodeUint32(opt.Roundtrip, d)
	return &dccp.Option{
		Type:      OptionRoundtripReport,
		Data:      d,
		Mandatory: false,
	}, nil
}
