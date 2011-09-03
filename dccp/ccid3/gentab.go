// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math"
)

var qTable = []struct{ rateInvLo uint32; q int64 }{}

func main() {
	fmt.Printf(
`// THIS FILE IS AUTO-GENERATED

package ccid3

var qTable = []struct{ rateInvLo uint32; q int64 }{
`)
	// j is the loss rate inverse
	for j := 1; j <= 100; j++ {
		p := 1 / float64(j) // loss rate
		q := (math.Sqrt(2*p/3) + 12*math.Sqrt(3*p/8)*p*(1+32*p*p))
		fmt.Printf("\t{ %d, %g\t},\n", j, q)
	}
	fmt.Printf(
`
}
`)
}
