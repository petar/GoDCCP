// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"testing"
)

func TestGoConjunction(t *testing.T) {
	NewGoConjunction("hello+world", 
		Go(func() { 
			fmt.Printf("Hello\n") 
		}, "hello"), 
	).Wait()
}
