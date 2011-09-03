// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math"
)

// TODO: Implement a compressed version of the table whereby 
// repeating values are omitted.

func main() {

	// Generate qTable
	fmt.Printf(
`// THIS FILE IS AUTO-GENERATED

package ccid3

var qTable = []struct{ RateInvLo uint32; Q int64 }{
`)
	// j is the loss rate inverse
	for j := 1; j < 3000; j++ {
		p := 1 / (1+float64(j)) // loss rate
		q := (math.Sqrt(2*p/3) + 12*math.Sqrt(3*p/8)*p*(1+32*p*p))
		fmt.Printf("\t{ % 4d, % 5d\t},\n", j, int64(q*1e3))
	}
	fmt.Printf(
`}
`)

}
