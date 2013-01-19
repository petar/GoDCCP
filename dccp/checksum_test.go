// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"math/rand"
	"testing"
)

func TestChecksum(t *testing.T) {
	// Test even-size checksum
	buf := make([]byte, 2+40)
	for i := 2; i < len(buf); i++ {
		buf[i] = byte(rand.Int())
	}
	csumUint16ToBytes(
		csumDone(csumSum(buf)),
		buf[0:2],
	)
	if csumDone(csumSum(buf)) != 0 {
		t.Errorf("csum")
	}

	// Test odd-size checksum
	buf = make([]byte, 41+2)
	for i := 2; i < len(buf); i++ {
		buf[i] = byte(rand.Int())
	}
	csumUint16ToBytes(
		csumDone(csumSum(buf)),
		buf[0:2],
	)
	if csumDone(csumSum(buf)) != 0 {
		t.Errorf("csum")
	}

}
