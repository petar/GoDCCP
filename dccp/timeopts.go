// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
	"time"
)

const (
	TenMicroInNano = 1e4 // 10 microseconds in nanoseconds
	OneInTenMicro  = 1e5 // 1 seconds in ten microsecond units
)

// TimestampOption, Section 13.1
// Time values are based on a circular uint32 value at 10 microseconds granularity
type TimestampOption Option

func NewTimestampOption(t int64) *TimestampOption {
	return &TimestampOption{
		Type:      TimestampOption,
		Data:      encodeTimestamp(t, make([]byte, 4)),
		Mandatory: false,
	}
}

// d must be a 4-byte slice
func encodeTimestamp(t int64, d []byte) []byte {
	var t0 uint32 = uint32(t / TenMicroInNano) 
	encode4ByteUint(t0, d)
	return d[0:4]
}

func ValidateTimestampOption(opt *Option) *TimestampOption {
	if opt.Type != TimestampOption || len(opt.Data) != 4 {
		return nil
	}
	return (*TimestampOption)(opt)
}

// GetTimestamp returns the timestamp option value in 10 microsecond circular units
func (opt *TimestampOption) GetTimestamp() uint32 {
	return decodeTimestamp(opt.Data)
}

func decodeTimestamp(d []byte) uint32 {
	return decode4ByteUint(d)
}

// ElapsedTimeOption, Section 13.2
// This option is permitted in any DCCP packet that contains an Acknowledgement Number; such
// options received on other packet types MUST be ignored.  It indicates how much time has
// elapsed since the packet being acknowledged -- the packet with the given Acknowledgement
// Number -- was received.
//
// The option data, Elapsed Time, represents an estimated lower bound on the amount of time
// elapsed since the packet being acknowledged was received, with units of hundredths of
// milliseconds (10 microseconds granularity).
type ElapsedTimeOption Option

const MaxElapsedTime = 4294967295 * TenMicroInNano // Maximum distinguishable elapsed time in nanoseconds

// The argument elapsed measures time in nanoseconds
func NewElapsedTimeOption(elapsed int64) *ElapsedTimeOption {
	return &ElapsedTimeOption{
		Type:      ElapsedTimeOption,
		Data:      encodeElapsed(elapsed, make([]byte, 4)),
		Mandatory: false,
	}
}

// d must be a 4-byte slice
func encodeElapsed(elapsed int64, d []byte) []byte {
	if elapsed >= MaxElapsedTime {
		elapsed = MaxElapsedTime
	}
	if elapsed < 1e9/2 {
		assert2Byte(elapsed / TenMicroInNano)
		e := uint16(elapsed / TenMicroInNano)
		encode2ByteUint(e, d)
		return d[0:2]
	} else {
		assert4Byte(elapsed / TenMicroInNano)
		e := uint32(elapsed / TenMicroInNano)
		encode4ByteUint(e, d)
		return d[0:4]
	}
	panic("unreach")
}

func ValidateElapsedTimeOption(opt *Option) *ElapsedTimeOption {
	if opt.Type != ElapsedTimeOption || (len(opt.Data) != 2 && len(opt.Data) != 4) {
		return nil
	}
	return (*ElapsedTimeOption)(opt)
}

// GetElapsed() returns the elapsed time value in nanoseconds
func (opt *ElapsedTimeOption) GetElapsed() int64 {
	return decodeElapsed(opt.Data)
}

func decodeElapsed(d []byte) int64 {
	var t int64
	switch len(d) {
	case 2:
		t = int64(decode2ByteUint(d))
	case 4:
		t = int64(decode4ByteUint(d))
	default:
		panic("unreach")
	}
	return t * TenMicroInNano
}

// TimestampEchoOption, Section 13.3
// Time values are based on a circular uint32 value at 10 microseconds granularity
type TimestampEchoOption Option

// The argument elapsed is in nanoseconds.
func NewTimestampEcho(timestampOpt *Option, elapsed int64) *TimestampEchoOption {
	d := make([]byte, 8)
	copy(d[0:4], timestampOpt.Data[0:4])
	if elapsed == 0 {
		d = d[0:4]
	} else {
		l := len(encodeElapsed(elapsed, d))
		d = d[0:4+l]
	}
	// The size of d can be 4, 6 or 8
	return &TimestampEcho{
		Type:      TimestampEchoOption,
		Data:      d,
		Mandatory: false,
	}
}

func ValidateTimestampEchoOption(opt *Option) *TimestampEchoOption {
	if opt.Type != TimestampEchoOption || 
		(len(opt.Data) != 4 && len(opt.Data) != 6 && len(opt.Data) != 8) {

		return nil
	}
	return (*TimestampEchoOption)(opt)
}

// GetTimestamp returns the timestamp echo option value in 10 microsecond circular units
func (opt *TimestampEchoOption) GetTimestamp() uint32 {
	return decodeTimestamp(opt.Data[0:4])
}

func (opt *TimestampEchoOption) GetElapsed() int64 {
	if len(opt.Data) == 4 {
		return 0
	}
	return decodeElapsed(opt.Data[4:])
}

// GetTimestampDiff() returns the smaller circular difference between t0 an t1
// in nanoseconds. While note that t0 and t1 are given in 10 microsecond circular units
func GetTimestampDiff(t0, t1 uint32) int64 {
	return int64(minUint32(t0-t1, t1-t0)) * TenMicroInNano
}

func minUint32(x, y uint32) uint32 {
	if x < y {
		return x
	}
	return y
}
