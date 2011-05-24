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

func NewTimestampOptionNow() *TimestampOption {
	d := make([]byte, 4)
	// Time inside timestamps is measured in 10 microseconds granularity
	var t uint32 = uint32(time.Nanoseconds() / TenMicroInNano) 
	encode4ByteUint(t, d)
	return &TimestampOption{
		Type:      TimestampOption,
		Data:      d,
		Mandatory: false,
	}
}

func ValidateTimestampOption(opt *Option) *TimestampOption {
	if opt.Type != TimestampOption || len(opt.Data) != 4 {
		return nil
	}
	return (*TimestampOption)(opt)
}

// GetTimestamp returns the timestamp option value in 10 microsecond circular units
func (opt *TimestampOption) GetTimestamp() uint32 {
	return decode4ByteUint(opt.Data)
}

// TimestampEchoOption, Section 13.3
// Time values are based on a circular uint32 value at 10 microseconds granularity
type TimestampEchoOption Option

func NewTimestampEcho(timestampOpt *Option) *TimestampEchoOption {
	return &TimestampEcho{
		Type:      TimestampEchoOption,
		Data:      timestampOpt.Data,
		Mandatory: false,
	}
}

func ValidateTimestampEchoOption(opt *Option) *TimestampEchoOption {
	if opt.Type != TimestampEchoOption || len(opt.Data) != 4 {
		return nil
	}
	return (*TimestampEchoOption)(opt)
}

// GetTimestamp returns the timestamp echo option value in 10 microsecond circular units
func (opt *TimestampEchoOption) GetTimestamp() uint32 {
	return decode4ByteUint(opt.Data)
}

// GetOptionTimestampDifference() returns the smaller circular difference between t0 an t1
// in nanoseconds. While note that t0 and t1 are given in 10 microsecond circular units
func GetOptionTimestampDifference(t0, t1 uint32) int64 {
	return int64(minUint32(t0-t1, t1-t0)) * TenMicroInNano
}

func minUint32(x, y uint32) uint32 {
	if x < y {
		return x
	}
	return y
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
	var d []byte
	if elapsed >= MaxElapsedTime {
		elapsed = MaxElapsedTime
	}
	if elapsed < 1e9/2 {
		d = make([]byte, 2)
		assert2Byte(elapsed / TenMicroInNano)
		e := uint16(elapsed / TenMicroInNano)
		encode2ByteUint(e, d)
	} else {
		d = make([]byte, 4)
		assert4Byte(elapsed / TenMicroInNano)
		e := uint32(elapsed / TenMicroInNano)
		encode4ByteUint(e, d)
	}
	return &ElapsedTimeOption{
		Type:      ElapsedTimeOption,
		Data:      d,
		Mandatory: false,
	}
}

func ValidateElapsedTimeOption(opt *Option) *ElapsedTimeOption {
	if opt.Type != ElapsedTimeOption || (len(opt.Data) != 2 && len(opt.Data) != 4) {
		return nil
	}
	return (*ElapsedTimeOption)(opt)
}

// GetElapsed() returns the elapsed time value in nanoseconds
func (opt *ElapsedTimeOption) GetElapsed() int64 {
	var t int64
	switch len(opt.Data) {
	case 2:
		t = int64(decode2ByteUint(opt.Data))
	case 4:
		t = int64(decode4ByteUint(opt.Data))
	default:
		panic("unreach")
	}
	return t * TenMicroInNano
}
