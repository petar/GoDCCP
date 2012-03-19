// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// TimestampOption, Section 13.1
// Time values are based on a circular uint32 value at 10 microseconds granularity
type TimestampOption struct {
	// Timestamp is given in 10 microsecond circular units
	Timestamp uint32
}

func (opt *TimestampOption) Encode() (*Option, error) {
	return &Option{
		Type:      OptionTimestamp,
		Data:      encodeTimestamp(opt.Timestamp, make([]byte, 4)),
		Mandatory: false,
	}, nil
}

// d must be a 4-byte slice
func encodeTimestamp(t uint32, d []byte) []byte {
	EncodeUint32(t, d)
	return d[0:4]
}

func DecodeTimestampOption(opt *Option) *TimestampOption {
	if opt.Type != OptionTimestamp || len(opt.Data) != 4 {
		return nil
	}
	return &TimestampOption{Timestamp: decodeTimestamp(opt.Data[0:4])}
}

func decodeTimestamp(d []byte) uint32 {
	return DecodeUint32(d)
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
type ElapsedTimeOption struct {
	Elapsed uint32
}

const (
	TenMicroInNano       = 1e4		// 10 microseconds in nanoseconds
	OneSecInTenMicro     = 1e5		// 1 seconds in ten microsecond units
	MaxTenMicro          = 4294967295	// Maximum allowed time in ten microsecond units
	MaxElapsedInTenMicro = MaxTenMicro	// Maximum elapsed time in ten microsecond units
)

func (opt *ElapsedTimeOption) Encode() (*Option, error) {
	return &Option{
		Type:      OptionElapsedTime,
		Data:      encodeElapsed(opt.Elapsed, make([]byte, 4)),
		Mandatory: false,
	}, nil
}

// d must be a 4-byte slice
func encodeElapsed(elapsed uint32, d []byte) []byte {
	if elapsed >= MaxElapsedInTenMicro {
		elapsed = MaxElapsedInTenMicro
	}
	if elapsed < OneSecInTenMicro / 2 {
		assertFitsIn16Bits(uint64(elapsed))
		EncodeUint16(uint16(elapsed), d[0:2])
		return d[0:2]
	} else {
		EncodeUint32(elapsed, d)
		return d[0:4]
	}
	panic("unreach")
}

func DecodeElapsedTimeOption(opt *Option) *ElapsedTimeOption {
	if opt.Type != OptionElapsedTime || (len(opt.Data) != 2 && len(opt.Data) != 4) {
		return nil
	}
	elapsed, err := decodeElapsed(opt.Data)
	if err != nil {
		return nil
	}
	return &ElapsedTimeOption{
		Elapsed: elapsed,
	}
}

func decodeElapsed(d []byte) (uint32, error) {
	var t uint32
	switch len(d) {
	case 2:
		t = uint32(DecodeUint16(d))
	case 4:
		t = DecodeUint32(d)
	default:
		return 0, ErrSize
	}
	return t, nil
}

// —————
// TimestampEchoOption, Section 13.3
// Time values are based on a circular uint32 value at 10 microseconds granularity
type TimestampEchoOption struct {
	// The timestamp echo option value in 10 microsecond circular units
	Timestamp uint32
	// The elapsed time in nanoseconds
	Elapsed uint32
}

func (opt *TimestampEchoOption) Encode() (*Option, error) {
	d := make([]byte, 8)
	encodeTimestamp(opt.Timestamp, d[0:4])
	if opt.Elapsed == 0 {
		d = d[0:4]
	} else {
		l := len(encodeElapsed(opt.Elapsed, d[4:]))
		d = d[0 : 4+l]
	}
	// The size of d can be 4, 6 or 8
	return &Option{
		Type:      OptionTimestampEcho,
		Data:      d,
		Mandatory: false,
	}, nil
}

func DecodeTimestampEchoOption(opt *Option) *TimestampEchoOption {
	if opt.Type != OptionTimestampEcho ||
		(len(opt.Data) != 4 && len(opt.Data) != 6 && len(opt.Data) != 8) {

		return nil
	}
	var elapsed uint32
	if len(opt.Data) > 4 {
		var err error
		elapsed, err = decodeElapsed(opt.Data[4:])
		if err != nil {
			return nil
		}
	}
	return &TimestampEchoOption{
		Timestamp: decodeTimestamp(opt.Data[0:4]),
		Elapsed:   elapsed,
	}
}

// TenMicroDiff returns the circular difference between t0 an t1.
// The arguments and the result are in ten microsecond units.
func TenMicroDiff(t0, t1 uint32) uint32 { return minu32(t0-t1, t1-t0) }

// TenMicroFromNano converts a time length given in nanoseconds into 
// units of 10 microseconds, capped by MaxTenMicro.
func TenMicroFromNano(ns int64) uint32 {
	if ns < 0 {
		panic("negative time")
	}
	return uint32(min64(ns/TenMicroInNano, MaxTenMicro))
}

// NanoFromTenMicro converts a time length given in ten microsecond units into
// nanoseconds, without exceeding the maximum allowed time limit of MaxTenMicro.
func NanoFromTenMicro(tenmicro uint32) int64 {
	return min64(int64(tenmicro)*TenMicroInNano, MaxTenMicro*TenMicroInNano)
}
