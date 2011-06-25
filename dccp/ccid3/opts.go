// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import "os"

// CCID3-specific options
const (
	OptionLossEventRate = 192
	OptionLossIntervals = 193
	OptionReceiveRate   = 194
)

// RFC 4342, Section 8.5
type LossEventRateOption struct {
	// RateInv is the inverse of the loss event rate, rounded UP, as calculated by the receiver
	RateInv uint32
}

const UnknownLossEventRate = 2 ^ 32 - 1

func DecodeLossEventRateOption(opt *Option) *LossEventRateOption {
	if opt.Type != OptionLossEventRate || len(opt.Data) != 4 {
		return nil
	}
	return &LossEventRateOption{RateInv: Decode4ByteUint(opt.Data[0:4])}
}

func (opt *LossEventRateOption) Encode() (*Option, os.Error) {
	d := make([]byte, 4)
	Encode4ByteUint(opt.RateInv, d)
	return &Option{
		Type:      OptionLossEventRate,
		Data:      d,
		Mandatory: false,
	}, nil
}


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

func DecodeLossIntervalsOption(opt *Option) *LossIntervalsOption {
	if opt.Type != OptionLossIntervals || len(opt.Data) < 1 {
		return nil
	}
	k, r := (len(opt.Data)-1)/lossIntervalFootprint, (len(opt.Data)-1)%lossIntervalFootprint
	if k > MaxLossIntervals || r != 0 {
		return nil
	}
	skip := Decode1ByteUint(opt.Data[0:1])
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

func (opt *LossIntervalsOption) Encode() (*Option, os.Error) {
	if opt.SkipLength > NDUPACK {
		return nil, ErrOverflow
	}
	if len(opt.LossIntervals) > MaxLossIntervals {
		return nil, ErrOverflow
	}
	d := make([]byte, 1+lossIntervalFootprint*len(opt.LossIntervals))
	Encode1ByteUint(opt.SkipLength, d[0:1])
	for i, lossInterval := range opt.LossIntervals {
		j := 1 + i*lossIntervalFootprint
		if lossInterval.encode(d[j:j+lossIntervalFootprint]) != nil {
			return nil, nil
		}
	}
	return &Option{
		Type:      OptionLossIntervals,
		Data:      d,
		Mandatory: false,
	}, nil
}

// LossInterval describes an individual loss interval, RFC 4342, Section 8.6.2
type LossInterval struct {
	LosslessLength uint32 // Lossless Length, a 24-bit number, RFC 4342, Section 8.6.1
	LossLength     uint32 // Loss Length, a 23-bit number, RFC 4342, Section 8.6.1
	DataLength     uint32 // Data Length, a 24-bit number, RFC 4342, Section 8.6.1
	ECNNonceEcho   bool   // ECN Nonce Echo, RFC 4342, Section 8.6.1
}

const _24thBit = 1 << 23

func (li *LossInterval) encode(p []byte) os.Error {
	if len(p) != 9 {
		return ErrSize
	}
	if !FitsIn3Bytes(uint64(li.LosslessLength)) ||
		!fitsIn23Bits(uint64(li.LossLength)) ||
		!FitsIn3Bytes(uint64(li.DataLength)) {
		return ErrOverflow
	}

	Encode3ByteUint(li.LosslessLength, p[0:3])
	l := li.LossLength
	if li.ECNNonceEcho {
		l |= _24thBit
	}
	Encode3ByteUint(l, p[3:6])
	Encode3ByteUint(li.DataLength, p[6:9])

	return nil
}

func decodeLossInterval(p []byte) *LossInterval {
	li := &LossInterval{}
	li.LosslessLength = Decode3ByteUint(p[0:3])
	li.LossLength = Decode3ByteUint(p[3:6])
	li.ECNNonceEcho = (li.LossLength&_24thBit != 0)
	li.LossLength &= ^uint32(_24thBit)
	li.DataLength = Decode3ByteUint(p[6:9])
	return li
}


// RFC 4342, Section 8.3
type ReceiveRateOption struct {
	Rate uint32 // in bytes per second
}

func DecodeReceiveRateOption(opt *Option) *ReceiveRateOption {
	if opt.Type != OptionReceiveRate || len(opt.Data) != 4 {
		return nil
	}
	return &ReceiveRateOption{Rate: Decode4ByteUint(opt.Data[0:4])}
}

func (opt *ReceiveRateOption) Encode() (*Option, os.Error) {
	d := make([]byte, 4)
	Encode4ByteUint(opt.Rate, d)
	return &Option{
		Type:      OptionReceiveRate,
		Data:      d,
		Mandatory: false,
	}, nil
}
