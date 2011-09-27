// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"fmt"
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

func TestStrober(t *testing.T) {
	var clog dccp.CLog
	clog.Init("x")
	var s strober
	s.Init(clog, 1024, 1024)
	for {
		s.Strobe()
		fmt.Printf("*")
	}
}
