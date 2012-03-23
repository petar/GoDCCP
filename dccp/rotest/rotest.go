// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)
/*
type Waiter interface {
	Wait()
	String() string
}

type GoRoutine struct {
	ch   chan int
	file string
	line int
	anno string
}
*/
func main() {
	dccp.NewGoConjunction("hello+world", 
		dccp.Go(func() { 
			fmt.Printf("Hello\n") 
		}, "hello"), 
	).Wait()
}
