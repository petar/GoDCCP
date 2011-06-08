// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"rand"
	. "github.com/petar/GoDCCP/dccp"
)

func main() {
	InstallCtrlCPanic()
	defer func() { chan int(nil) <- 1 }()

	// Install stacks
	linka, linkb := NewChanPipe()
	newcc := NewConstRateControlFunc(10)
	stacka, stackb := NewStack(linka, newcc), NewStack(linkb, newcc)

	// Establish connection
	ca, err := stacka.Dial(nil, 1)
	if err != nil {
		log.Printf("side a dial: %s", err)
	}
	cb, err := stackb.Accept()
	if err != nil {
		log.Printf("side b accept: %s", err)
	}

	// Prepare random block
	p := make([]byte, 10)
	for i, _ := range p {
		p[i] = byte(rand.Int())
	}

	// Write and read the block
	err = ca.WriteBlock(p)
	if err != nil {
		log.Printf("side a write: %s", err)
	}
	q, err := cb.ReadBlock()
	if err != nil {
		log.Printf("side b read: %s", err)
	}

	// Compare
	if !byteSlicesEqual(p, q) {
		log.Printf("read and write blocks differ")
	}

	// Close connection
	err = ca.Close()
	if err != nil {
		log.Printf("side a close: %s", err)
	}
	err = cb.Close()
	if err != nil {
		log.Printf("side b close: %s", err)
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
