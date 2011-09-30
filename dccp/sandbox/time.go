// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

type Time struct {
}

func NewTime() *Time {
	panic("a")
}

func (t *Time) Nanoseconds() int64 {
	panic("a")
}

func (t *Time) Sleep(ns int64) {
	panic("a")
}
