// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"math"
	"os"
	"github.com/petar/GoDCCP/dccp"
)

// CCID3-specific options
const (
	OptionLossEventRate = 192
	OptionLossIntervals = 193
	OptionReceiveRate   = 194
	// OptionLossDigest IS NOT a part of CCID3. It is an extension.
	OptionLossDigest    = 210
)

// —————
// Unencoded option is a type that knows how to encode itself into a dccp.Option
type UnencodedOption interface {
	Encode() (*dccp.Option, os.Error)
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

// —————
// RFC 4342, Section 8.5
type LossEventRateOption struct {
	// RateInv is the inverse of the loss event rate, rounded UP, as calculated by the receiver.
	// It is actually calculated as data packets per loss interval.
	RateInv uint32
}

const UnknownLossEventRate = math.MaxUint32

func DecodeLossEventRateOption(opt *dccp.Option) *LossEventRateOption {
	if opt.Type != OptionLossEventRate || len(opt.Data) != 4 {
		return nil
	}
	return &LossEventRateOption{RateInv: dccp.Decode4ByteUint(opt.Data[0:4])}
}

func (opt *LossEventRateOption) Encode() (*dccp.Option, os.Error) {
	d := make([]byte, 4)
	dccp.Encode4ByteUint(opt.RateInv, d)
	return &dccp.Option{
		Type:      OptionLossEventRate,
		Data:      d,
		Mandatory: false,
	}, nil
}


// —————
// RFC 4342, Section 8.6
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
	skip := dccp.Decode1ByteUint(opt.Data[0:1])
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

func (opt *LossIntervalsOption) Encode() (*dccp.Option, os.Error) {
	if opt.SkipLength > NDUPACK {
		return nil, dccp.ErrOverflow
	}
	if len(opt.LossIntervals) > MaxLossIntervals {
		return nil, dccp.ErrOverflow
	}
	d := make([]byte, 1+lossIntervalFootprint*len(opt.LossIntervals))
	dccp.Encode1ByteUint(opt.SkipLength, d[0:1])
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
	ECNNonceEcho   bool   // ECN Nonce Echo, RFC 4342, Section 8.6.1
}

const _24thBit = 1 << 23

// SeqLen returns the sequence length of the loss interval
func (li *LossInterval) SeqLen() uint32 {
	return li.LosslessLength + li.LossLength
}

func (li *LossInterval) encode(p []byte) os.Error {
	if len(p) != 9 {
		return dccp.ErrSize
	}
	if !dccp.FitsIn3Bytes(uint64(li.LosslessLength)) ||
		!dccp.FitsIn23Bits(uint64(li.LossLength)) ||
		!dccp.FitsIn3Bytes(uint64(li.DataLength)) {
		return dccp.ErrOverflow
	}

	dccp.Encode3ByteUint(li.LosslessLength, p[0:3])
	l := li.LossLength
	if li.ECNNonceEcho {
		l |= _24thBit
	}
	dccp.Encode3ByteUint(l, p[3:6])
	dccp.Encode3ByteUint(li.DataLength, p[6:9])

	return nil
}

func decodeLossInterval(p []byte) *LossInterval {
	li := &LossInterval{}
	li.LosslessLength = dccp.Decode3ByteUint(p[0:3])
	li.LossLength = dccp.Decode3ByteUint(p[3:6])
	li.ECNNonceEcho = (li.LossLength&_24thBit != 0)
	li.LossLength &= ^uint32(_24thBit)
	li.DataLength = dccp.Decode3ByteUint(p[6:9])
	return li
}


// —————
// RFC 4342, Section 8.3
type ReceiveRateOption struct {
	// The rate at which receiver has received data since it last send an Ack; in bytes per second
	Rate uint32 
}

func DecodeReceiveRateOption(opt *dccp.Option) *ReceiveRateOption {
	if opt.Type != OptionReceiveRate || len(opt.Data) != 4 {
		return nil
	}
	return &ReceiveRateOption{Rate: dccp.Decode4ByteUint(opt.Data[0:4])}
}

func (opt *ReceiveRateOption) Encode() (*dccp.Option, os.Error) {
	d := make([]byte, 4)
	dccp.Encode4ByteUint(opt.Rate, d)
	return &dccp.Option{
		Type:      OptionReceiveRate,
		Data:      d,
		Mandatory: false,
	}, nil
}

// —————
// The LossDigest option directly carries, from the receiver to the sender, the types of loss
// information that a CCID3 sender would have to reconstruct from the LossIntervals option.  This is
// an extension to the RFC specification.
type LossDigestOption struct {
	// RateInv is the inverse of the loss event rate, rounded UP, as calculated by the receiver.
	RateInv uint32
	// NewLoss indicates how many new loss events are reported by the feedback packet carrying this option
	NewLossCount uint8
}

func DecodeLossDigestOption(opt *dccp.Option) *LossDigestOption {
	if opt.Type != OptionLossDigest || len(opt.Data) != 5 {
		return nil
	}
	return &LossDigestOption{
		RateInv:      dccp.Decode4ByteUint(opt.Data[0:4]),
		NewLossCount: dccp.Decode1ByteUint(opt.Data[4:5]),
	}
}

func (opt *LossDigestOption) Encode() (*dccp.Option, os.Error) {
	d := make([]byte, 5)
	dccp.Encode4ByteUint(opt.RateInv, d)
	dccp.Encode1ByteUint(opt.NewLossCount, d[4:])
	return &dccp.Option{
		Type:      OptionLossDigest,
		Data:      d,
		Mandatory: false,
	}, nil
}
