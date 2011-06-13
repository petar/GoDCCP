// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"log"
	// "rand"
	"time"
	. "github.com/petar/GoDCCP/dccp"
)

func main() {
	//rand.Seed(time.Nanoseconds())

	InstallCtrlCPanic()
	defer SavePanicTrace()
	defer time.Sleep(2e9) // Sleep for 1 min after end

	// Install stacks
	linka, linkb := NewChanPipe()
	newcc := NewConstRateControlFunc(12)
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
	p := []byte("hello world!")

	// Write and read the block
	for t := 0; t < 30; t++ {
		err = ca.WriteBlock(p)
		if err != nil {
			log.Printf("side a write: %s", err)
		}
		q, err := cb.ReadBlock()
		if err != nil {
			log.Printf("side b read: %s", err)
		}
		log.Printf("<--%d--|%s|\n", t, string(q))

		// Compare
		if !byteSlicesEqual(p, q) {
			log.Printf("read and write blocks differ")
		}
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
