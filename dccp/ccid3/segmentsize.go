// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

// senderSegmentSize keeps an up-to-date estimate of the Segment Size (SS).
// The current implementation simply uses the MPS (maximum packet size) as SS.
// TODO: Compute SS as the average SS over a few most recent loss intervals, see Section 5.3.
type senderSegmentSize struct {
	mps int
}

const FixedSegmentSize = 2*1500

// Init resets the object for new use
func (t *senderSegmentSize) Init() {
	t.mps = 0
}

// Sender calls SetMPS to notify this object if the maximum packet size in use
func (t *senderSegmentSize) SetMPS(mps int) { t.mps = mps }

// SS returns the current estimate of the segment size
func (t *senderSegmentSize) SS() int { 
	if t.mps <= 0 {
		panic("not ready with SS")
	}
	// TODO: Segment Size should equal maximum packet size minus DCCP header size.
	// In other words, it is supposed to reflect the app data size.
	// Since we are in user space, we tend to use MPS to mean app data as well. 
	// It would be nice to set these straight eventually and use more uniform terminology.
	return t.mps
}
