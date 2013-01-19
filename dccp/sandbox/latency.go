// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"sort"
	"github.com/petar/GoDCCP/dccp"
)

type latencyQueue struct {
	env         *dccp.Env
	amb         *dccp.Amb
	queue       []*pipeHeader
}

// Init initializes the queue for initial use
func (x *latencyQueue) Init(env *dccp.Env, amb *dccp.Amb) {
	x.env = env
	x.amb = amb
	x.queue = make([]*pipeHeader, 0)
}

// Add adds a new item to the queue
func (x *latencyQueue) Add(ph *pipeHeader) {
	x.queue = append(x.queue, ph)
	sort.Sort(pipeHeaderTimeSort(x.queue))
}

// DeleteMin removes the item with lowest timestamp from the queue
func (x *latencyQueue) DeleteMin() *pipeHeader {
	if len(x.queue) == 0 {
		return nil
	}
	ph := x.queue[0]
	x.queue = x.queue[1:]
	return ph
}

// TimeToMin returns the duration of time from now until the timestamp of the
// earliest item in the queue
func (x *latencyQueue) TimeToMin() (dur int64, present bool) {
	if len(x.queue) == 0 {
		return 0, false
	}
	now := x.env.Now()
	return max64(0, x.queue[0].DeliverTime - now), true
}

// pipeHeaderTimeSort sorts a slice of *pipeHeader by timestamp
type pipeHeaderTimeSort []*pipeHeader

func (t pipeHeaderTimeSort) Len() int {
	return len(t)
}

func (t pipeHeaderTimeSort) Less(i, j int) bool {
	return t[i].DeliverTime < t[j].DeliverTime
}

func (t pipeHeaderTimeSort) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
