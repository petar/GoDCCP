// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"rand"
	"testing"
)

func TestChecksum(t *testing.T) {
	buf := make([]byte, 40+2)
	for i := 0; i < len(buf)-2; i++ {
		buf[i] = byte(rand.Int())
	}
	csumUint16ToBytes(
		csumDone(csumSum(buf)),
		buf[len(buf)-2:],
	)
	if csumDone(csumSum(buf)) != 0 {
		t.Errorf("csum")
	}
}
