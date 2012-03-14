// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
)

type Mutex struct {
	sync.Mutex
}

func (m *Mutex) Lock() {
	m.Mutex.Lock()
}

func (m *Mutex) AssertLocked() {
	// TODO: Implement
}

func (m *Mutex) Unlock() {
	m.Mutex.Unlock()
}
