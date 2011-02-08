// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"


// SC:	Indicates a Service Code representable using a subset of the
//      ASCII characters.  The colon is followed by one to four
//      characters taken from the following set: letters, digits, and
//      the characters in "-_+.*/?@" (not including quotes).
//      Numerically, these characters have values in {42-43, 45-57,
//      63-90, 95, 97-122}.  The Service Code is calculated by
//      padding the string on the right with spaces (value 32) and
//	intepreting the four-character result as a 32-bit big-endian
//      number.

func MarshalServiceCode(code uint32) string { 
	? 
}

func UnmarshalServiceCode(p []byte) (uint32, os.Error) { 
	? 
}
