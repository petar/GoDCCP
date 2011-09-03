// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math"
)

func main() {
	// j is the loss rate inverse
	for j := 1; j <= 100; j++ {
		p := 1 / float64(j) // loss rate
		q := (math.Sqrt(2*p/3) + 12*math.Sqrt(3*p/8)*p*(1+32*p*p))
		fmt.Printf("%d\t%1.uint32\t%1.3g\n", j, p, q)
	}
}
