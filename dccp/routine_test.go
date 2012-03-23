// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"testing"
	"time"
)

func TestGoConjunction(t *testing.T) {
	var hello, world bool
	NewGoConjunction("hello+world", 
		Go(func() { 
			hello = true
			time.Sleep(time.Second)
		}, "hello"), 
		Go(func() { 
			world = true
			time.Sleep(time.Second/2)
		}, "world"), 
	).Wait()
	if !hello || !world {
		t.Errorf("go routines did not complete")
	}
}
