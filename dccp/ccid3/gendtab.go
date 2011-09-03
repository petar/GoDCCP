// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math"
)

func main() {
	for j := 0; j <= 100; j++ {
		p := float64(j) / 100
		q := (math.Sqrt(2*p/3) + 12*math.Sqrt(3*p/8)*p*(1+32*p*p))
		fmt.Printf("%g\t%g\n", p, q)
	}
}
