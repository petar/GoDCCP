// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"time"
)

type Time interface {
	// Nanoseconds returns the current time in nanoseconds since UTC zero
	Nanoseconds() int64

	// Sleep blocks for ns nanoseconds 
	Sleep(ns int64)
}

type RealTime struct {}

func (RealTime) Nanoseconds() int64 {
	return time.Nanoseconds()
}

func (RealTime) Sleep(ns int64) {
	<-time.NewTimer(ns).C
}
