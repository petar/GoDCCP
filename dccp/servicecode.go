// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// After '8.1.2. Service Codes'

func isValidServiceCode(sc uint32) bool { return sc != 4294967295 }

// isASCIIServiceCodeChar() returns true if c@ is a ServiceCode character
// that can be displayed in ASCII
func isASCIIServiceCodeChar(c byte) bool {
	if c == 32 {
		return true
	}
	if c >= 42 && c <= 43 {
		return true
	}
	if c >= 45 && c <= 57 {
		return true
	}
	if c >= 63 && c <= 90 {
		return true
	}
	if c == 95 {
		return true
	}
	if c >= 97 && c <= 122 {
		return true
	}
	return false
}

func serviceCodeToSlice(u uint32) []byte {
	p := make([]byte, 4)
	p[0] = byte(u >> 3 * 8)
	p[1] = byte((u >> 2 * 8) & 0xff)
	p[2] = byte((u >> 1 * 8) & 0xff)
	p[3] = byte(u & 0xff)
	return p
}

func sliceToServiceCode(p []byte) uint32 {
	var s uint32
	s = uint32(p[0])
	s <<= 8
	s |= uint32(p[1])
	s <<= 8
	s |= uint32(p[2])
	s <<= 8
	s |= uint32(p[3])
	return s
}

// TODO: Implement additional string representations according to '8.1.2. Service Codes'

func ServiceCodeString(code uint32) string {
	return "SC:" + string(serviceCodeToSlice(code))
}

func ParseServiceCode(p []byte) (uint32, error) {
	if len(p) != 7 {
		return 0, ErrSyntax
	}
	if string(p[:3]) != "SC:" {
		return 0, ErrSyntax
	}
	return sliceToServiceCode(p[3:7]), nil
}
