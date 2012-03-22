// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/petar/GoDCCP/dccp"
)

func main() {
	dccp.NewGoConjunction("hello+world", 
		dccp.Go(func() { 
			fmt.Printf("Hello\n") 
		}, "hello"), 
	).Wait()
}
