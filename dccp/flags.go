// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
)

// Flags is a general purpose key-value map, which is used inside Amb to allow
// an Amb and all of its refinements to share a collection of debug flags.
type Flags struct {
	sync.Mutex
	flags map[string]interface{}
}

// NewFlags creates and initializes a new Flags instance
func NewFlags() *Flags {
	x := &Flags{}
	x.Init()
	return x
}

// Init clears a Flags instance for fresh use
func (x *Flags) Init() {
	x.Lock()
	defer x.Unlock()
	x.flags = make(map[string]interface{})
}

func (x *Flags) Set(key string, value interface{}) {
	x.Lock()
	defer x.Unlock()
	x.flags[key] = value
}

func (x *Flags) SetUint32(key string, value uint32) {
	x.Lock()
	defer x.Unlock()
	x.flags[key] = value
}

func (x *Flags) Has(key string) bool {
	x.Lock()
	defer x.Unlock()
	_, ok := x.flags[key]
	return ok
}

func (x *Flags) GetInt64(key string) (value int64, present bool) {
	x.Lock()
	defer x.Unlock()
	v, ok := x.flags[key]
	if !ok {
		return 0, false
	}
	return v.(int64), true
}

func (x *Flags) GetUint32(key string) (value uint32, present bool) {
	x.Lock()
	defer x.Unlock()
	v, ok := x.flags[key]
	if !ok {
		return 0, false
	}
	return v.(uint32), true
}
