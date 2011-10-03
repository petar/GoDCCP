// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"time"
)

func SetTime(time Time) {
	runtime.Time = time
}

func GetTime() Time {
	return runtime.Time
}

var runtime struct {
	Time
}

// Time is an interface for interacting time
type Time interface {
	// Nanoseconds returns the current time in nanoseconds since UTC zero
	Nanoseconds() int64

	// Sleep blocks for ns nanoseconds 
	Sleep(ns int64)
}

type realTime struct {}

var RealTime realTime

func (realTime) Nanoseconds() int64 {
	return time.Nanoseconds()
}

func (realTime) Sleep(ns int64) {
	<-time.NewTimer(ns).C
}
