// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"rand"
	"testing"
)

func TestWireEncodeDecode(t *testing.T) {

	buf := make([]byte, 8)
	var u8 uint8
	var u16 uint16
	var u32 uint32
	var u64 uint64

	// 1 byte
	u8 = uint8(rand.Int31())
	EncodeUint8(u8, buf[0:1])
	if DecodeUint8(buf[0:1]) != u8 {
		t.Errorf("E/D 1 byte")
	}

	// 2 byte
	u16 = uint16(rand.Int31())
	EncodeUint16(u16, buf[0:2])
	if DecodeUint16(buf[0:2]) != u16 {
		t.Errorf("E/D 2 byte")
	}

	// 3 byte
	u32 = uint32(rand.Int31())
	u32 = (u32 << 8) >> 8
	EncodeUint24(u32, buf[0:3])
	if DecodeUint24(buf[0:3]) != u32 {
		t.Errorf("E/D 3 byte")
	}

	// 4 byte
	u32 = uint32(rand.Int31()) << 1
	EncodeUint32(u32, buf[0:4])
	if DecodeUint32(buf[0:4]) != u32 {
		t.Errorf("E/D 4 byte")
	}

	// 6 byte
	u64 = uint64(rand.Int63())
	u64 = (u64 << (2 * 8)) >> (2 * 8)
	EncodeUint48(u64, buf[0:6])
	if DecodeUint48(buf[0:6]) != u64 {
		t.Errorf("E/D 6 byte")
	}
}
