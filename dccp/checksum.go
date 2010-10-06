// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

func csum64to16(sum uint64) uint16 {
	// 32+32 to 33
	sum = (sum & 0xffffffff) + (sum >> 32)
	// 17+16 to 17+c
	sum = (sum & 0xffff) + (sum >> 16)
	// (1+c)+16 to 16+c
	sum = (sum & 0xffff) + (sum >> 16)
	// c+16 to 16
	sum = (sum & 0xffff) + (sum >> 16)

	return uint16(sum)
}

func csumBytesToUint16(buf []byte) uint16 {
	return (uint16(buf[0]) << 8) | uint16(buf[1])
}

func csumUint16ToBytes(u uint16, buf []byte) {
	buf[0] = byte(u >> 8)
	buf[1] = byte(u & 0xff)
}

func csumAdd(u,w uint16) uint16 {
	sum := uint32(u) + uint32(w)
	sum = (sum & 0xffff) + (sum >> 16)
	return uint16(sum)
}

// TODO(petar): This method can be optimized significantly
func csumSum(buf []byte) uint16 {
	var sum uint16
	l16 := len(buf) >> 1
	for i := 0; i < l16; i++ {
		sum = csumAdd(sum, csumBytesToUint16(buf[2*i:2*i+2]))
	}
	if (l16 << 2) < len(buf) {
		two := make([]byte, 2)
		two[0] = buf[len(buf)-1]
		two[1] = 0
		sum = csumAdd(sum, csumBytesToUint16(two))
	}
	return sum
}

func csumPartial(sum uint16, buf []byte) uint16 {
	return csumAdd(sum, csumSum(buf))
}

func csumDone(sum uint16) uint16 { return ^sum }

// @dccpLen is the length of the DCCP header with options, plus the length of any data
func csumPseudoIP(sourceIP, destIP []byte, protoNo int, dccpLen int) uint16 {
	if len(sourceIP) == 4 {
		return csumPseudoIPv4(sourceIP, destIP, protoNo, dccpLen)
	}
	return csumPseudoIPv6(sourceIP, destIP, protoNo, dccpLen)
}

func csumPseudoIPv4(sourceIP, destIP []byte, protoNo int, dccpLen int) uint16 {
	if len(sourceIP) != 4 || len(destIP) != 4 {
		panic("size")
	}
	if uint(protoNo) >> 8 != 0 {
		panic("proto no")
	}
	if uint16(dccpLen) >> 16 != 0 {
		panic("len")
	}
	sum := csumSum(sourceIP)
	sum = csumPartial(sum, destIP)
	sum = csumAdd(sum, uint16(protoNo))
	sum = csumAdd(sum, uint16(dccpLen))
	return sum
}

func csumPseudoIPv6(sourceIP, destIP []byte, protoNo int, dccpLen int) uint16 {
	if len(sourceIP) != 16 || len(destIP) != 16 {
		panic("size")
	}
	if uint(protoNo) >> 8 != 0 {
		panic("proto no")
	}
	sum := csumSum(sourceIP)
	sum = csumPartial(sum, destIP)
	sum = csumAdd(sum, uint16(uint32(dccpLen) >> 16))
	sum = csumAdd(sum, uint16(uint32(dccpLen) & 0xffff))
	sum = csumAdd(sum, uint16(protoNo))
	return sum
}
