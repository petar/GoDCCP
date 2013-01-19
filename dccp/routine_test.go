// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"testing"
	"time"
)

func TestGoJoin(t *testing.T) {
	var hello, world bool
	NewGoJoin("hello+world", 
		Go(func() { 
			hello = true
			time.Sleep(time.Second)
		}, "hello"), 
		Go(func() { 
			world = true
			time.Sleep(time.Second/2)
		}, "world"), 
	).Join()
	if !hello || !world {
		t.Errorf("goroutines did not complete")
	}
}
