// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "sync"

// Go runs f in a new goroutine and returns a handle object, which can
// then be used for various synchronization mechanisms.
func Go(f func()) *GoRoutine {
	g := &GoRoutine{ onEnd: make(chan int, 0) }
	go func() {
		f()
		close(g.onEnd)
	}()
	return g
}

// GoRoutine represents a running goroutine.
type GoRoutine struct {
	onEnd chan int
}

// Wait blocks until the goroutine completes; otherwise,
// if the goroutine has completed, it returns immediately.
// Wait can be called concurrently.
func (g *GoRoutine) Wait() {
	_, _ = <-g.onEnd
}

// Waiter is an interface to objects that can wait for some event.
// Wait should be re-entrant and return immediately if called post-event.
type Waiter interface {
	Wait()
}

// ConjWaiter waits until a set of GoRoutines all complete.
type ConjWaiter struct {
	lk    sync.Mutex
	n     int
	onEnd chan int
}

// MakeWaitConj creates an object capable of waiting until all supplied GoRoutines complete.
func MakeConjWaiter(group ...Waiter) *ConjWaiter {
	var w *ConjWaiter = &ConjWaiter{ n: 0, onEnd: make(chan int) }
	for _, u := range group {
		w.Add(u)
	}
	return w
}

// Add adds a Waiter to the group. It can be called at any time
// as long as the current set of goroutines hasn't completed.
// For instance, as long as you call Add from a GoRoutine which
// is waited on by this object, the condtion will be met.
func (t *ConjWaiter) Add(u Waiter) {
	t.lk.Lock()
	defer t.lk.Unlock()
	if t.n < 0 {
		panic("adding goroutine after completion")
	}
	t.n++
	go func() {
		u.Wait()
		t.onEnd <- 1
	}()
}

// Go is a convenience method which forks f into a new GoRoutine and
// adds the latter to the waiting queue.
func (t *ConjWaiter) Go(f func()) {
	t.Add(Go(f))
}

// Wait blocks until all goroutines in the group have completed.
// Wait can be called concurrently. If called post-completion of the
// goroutine group, Wait returns immediately.
func (t *ConjWaiter) Wait() {
	for t.stillRemain() {
		_, ok := <-t.onEnd
		if !ok {
			return
		}
		t.lk.Lock()
		t.n--
		if t.n == 0 {
			t.n = -1
			close(t.onEnd)
		}
		t.lk.Unlock()
	}
}

func (t* ConjWaiter) stillRemain() bool {
	t.lk.Lock()
	defer t.lk.Unlock()
	return t.n > 0
}
