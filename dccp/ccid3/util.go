// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

// —————
// The following are custom types intended to distinguish the different units of measurement
// that might otherwise use similar primitive types.

type UnitNS    int64   // Duration expressed in nanoseconds
type UnitNSUTC int64   // Time relative to UTC zero in nanoseconds
type UnitBPS   uint32  // Bytes per second
type UnitPPS   uint32  // Packets (segments) per second

// In CCID3 and related RFCs, the term 'rate' means fraction (of a whole), and thus assumes
// values in the real interval [0,1]. It is not to be confused with 'frequency' which
// usually means volume-over-time and can assume non-negative reals as values.
//
// The type UnitRate represents a discretized rate with 0 corresponding to 0.0 and 1e9
// corresponding to 1.0. Thus a UnitRate variable assumes values in [0, 1e9].
//
type UnitRate int64

// —————
// Below are some basic utility functions

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func minu32(x, y uint32) uint32 {
	if x < y {
		return x
	}
	return y
}

func min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func maxu32(x, y uint32) uint32 {
	if x > y {
		return x
	}
	return y
}

func max64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}
