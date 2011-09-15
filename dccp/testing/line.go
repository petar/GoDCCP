// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package testing

import (
	"os"
	"sync"
	"github.com/petar/GoDCCP/dccp"
)

//   headerHalfPipe <---> Line <---> headerHalfPipe
type Line struct {
	SideA, SideB dccp.HeaderConn
}

func NewLine() (sidea, sideb dccp.HeaderConn, line *Line) {
}

func (t *Line) SetRate(segmentsPerSecond uint32) {
	// code
}
