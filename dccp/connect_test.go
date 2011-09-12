// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"rand"
	"testing"
)

func TestConnect(t *testing.T) {

	// Install stacks
	linka, linkb := NewChanPipe()
	newscc := NewFixedRateSenderControlFunc(10)
	newrcc := NewFixedRateReceiverControlFunc()
	stacka, stackb := NewStack(linka, newscc, newrcc), NewStack(linkb, newscc, newrcc)

	// Establish connection
	ca, err := stacka.Dial(nil, 1)
	if err != nil {
		t.Fatalf("side a dial: %s", err)
	}
	cb, err := stackb.Accept()
	if err != nil {
		t.Fatalf("side b accept: %s", err)
	}

	// Prepare random block
	p := make([]byte, 10)
	for i, _ := range p {
		p[i] = byte(rand.Int())
	}

	// Write and read the block
	err = ca.WriteBlock(p)
	if err != nil {
		t.Errorf("side a write: %s", err)
	}
	q, err := cb.ReadBlock()
	if err != nil {
		t.Errorf("side b read: %s", err)
	}

	// Compare
	if !byteSlicesEqual(p, q) {
		t.Errorf("read and write blocks differ")
	}

	// Close connection
	err = ca.Close()
	if err != nil {
		t.Errorf("side a close: %s", err)
	}
	err = cb.Close()
	if err != nil {
		t.Errorf("side b close: %s", err)
	}
}

func byteSlicesEqual(a,b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		if b[i] != x {
			return false
		}
	}
	return true
}
