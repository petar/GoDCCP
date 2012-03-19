// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// All multi-byte numerical quantities in DCCP, such as port numbers, Sequence Numbers, and
// arguments to options, are transmitted in network byte order (most significant byte
// first).

// Wire format to integers

func DecodeUint8(w []byte) uint8 {
	if len(w) != 1 {
		panic("size")
	}
	return uint8(w[0])
}

func DecodeUint16(w []byte) uint16 {
	if len(w) != 2 {
		panic("size")
	}
	var u uint16
	u |= uint16(w[1])
	u |= uint16(w[0]) << 8
	return u
}

func DecodeUint24(w []byte) uint32 {
	if len(w) != 3 {
		panic("size")
	}
	var u uint32
	u |= uint32(w[2])
	u |= uint32(w[1]) << (8 * 1)
	u |= uint32(w[0]) << (8 * 2)
	return u
}

func DecodeUint32(w []byte) uint32 {
	if len(w) != 4 {
		panic("size")
	}
	var u uint32
	u |= uint32(w[3])
	u |= uint32(w[2]) << (8 * 1)
	u |= uint32(w[1]) << (8 * 2)
	u |= uint32(w[0]) << (8 * 3)
	return u
}

func DecodeUint48(w []byte) uint64 {
	if len(w) != 6 {
		panic("size")
	}
	var u uint64
	u |= uint64(w[5])
	u |= uint64(w[4]) << (8 * 1)
	u |= uint64(w[3]) << (8 * 2)
	u |= uint64(w[2]) << (8 * 3)
	u |= uint64(w[1]) << (8 * 4)
	u |= uint64(w[0]) << (8 * 5)
	return u
}

// Integers to wire format

func EncodeUint8(u uint8, w []byte) {
	if len(w) != 1 {
		panic("size")
	}
	w[0] = u
}

func EncodeUint16(u uint16, w []byte) {
	if len(w) != 2 {
		panic("size")
	}
	w[1] = uint8(u & 0xff)
	w[0] = uint8((u >> 8) & 0xff)
}

func EncodeUint24(u uint32, w []byte) {
	if len(w) != 3 {
		panic("size")
	}
	w[2] = uint8(u & 0xff)
	w[1] = uint8((u >> (8 * 1)) & 0xff)
	w[0] = uint8((u >> (8 * 2)) & 0xff)
	if (u >> (8 * 3)) != 0 {
		panic("overflow")
	}
}

func EncodeUint32(u uint32, w []byte) {
	if len(w) != 4 {
		panic("size")
	}
	w[3] = uint8(u & 0xff)
	w[2] = uint8((u >> (8 * 1)) & 0xff)
	w[1] = uint8((u >> (8 * 2)) & 0xff)
	w[0] = uint8((u >> (8 * 3)) & 0xff)
}

func EncodeUint48(u uint64, w []byte) {
	if len(w) != 6 {
		panic("size")
	}
	w[5] = uint8(u & 0xff)
	w[4] = uint8((u >> (8 * 1)) & 0xff)
	w[3] = uint8((u >> (8 * 2)) & 0xff)
	w[2] = uint8((u >> (8 * 3)) & 0xff)
	w[1] = uint8((u >> (8 * 4)) & 0xff)
	w[0] = uint8((u >> (8 * 5)) & 0xff)
	if (u >> (8 * 6)) != 0 {
		panic("overflow")
	}
}

// Assertions

func FitsIn16Bits(x uint64) bool { return x>>16 == 0 }

func FitsIn24Bits(x uint64) bool { return x>>24 == 0 }

func FitsIn32Bits(x uint64) bool { return x>>32 == 0 }

func FitsIn23Bits(x uint64) bool { return x>>23 == 0 }

func assertFitsIn16Bits(x uint64) {
	if !FitsIn16Bits(x) {
		panic("width overflow, 2 bytes")
	}
}

func assertFitsIn32Bits(x uint64) {
	if !FitsIn32Bits(x) {
		panic("width overflow, 4 bytes")
	}
}
