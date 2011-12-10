// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"path"
	goruntime "runtime"
	//"os"
	"sync"
)

// GoRoutine represents a running goroutine.
type GoRoutine struct {
	onEnd chan int
	file  string
	line  int
}

// Go runs f in a new goroutine and returns a handle object, which can
// then be used for various synchronization mechanisms.
func GoCaller(f func(), level int) *GoRoutine {
	sfile, sline := FetchCaller(2 + level)
	g := &GoRoutine{ 
		onEnd: make(chan int, 0),
		file:  sfile,
		line:  sline,
	}
	go func() {
		f()
		close(g.onEnd)
	}()
	return g
}

func Go(f func()) *GoRoutine {
	return GoCaller(f, 1)
}

// Wait blocks until the goroutine completes; otherwise,
// if the goroutine has completed, it returns immediately.
// Wait can be called concurrently.
func (g *GoRoutine) Wait() {
	u := g.String()
	fmt.Printf(" go: + %s\n", u)
	_, _ = <-g.onEnd
	fmt.Printf(" go: - %s\n", u)
}

// Source returns the file and line where the goroutine was forked.
func (g *GoRoutine) Source() (sfile string, sline int) {
	return g.file, g.line
}

// Strings returns a unique and readable string representation of this instance.
func (g *GoRoutine) String() string {
	sfile, sline := g.Source()
	return fmt.Sprintf("%s:%d (%p)", sfile, sline, g)
}

// FetchCaller returns a shortened (more readable) version of the
// source file name, as well as the source line number of the caller.
func FetchCaller(level int) (sfile string, sline int) {
	_, sfile, sline, _ = goruntime.Caller(1+level)
	sdir, sfile := path.Split(sfile)
	if len(sdir) > 0 {
		_, sdir = path.Split(sdir[:len(sdir)-1])
	}
	sfile = path.Join(sdir, sfile)
	return sfile, sline
}

// Waiter is an interface to objects that can wait for some event.
// Wait should be re-entrant and return immediately if called post-event.
type Waiter interface {
	Wait()
	String() string
}

// GoGroup waits until a set of GoRoutines all complete. It also allows
// new routines to be added dynamically before the completion event.
type GoGroup struct {
	srcFile string
	srcLine int
	lk      sync.Mutex
	group   []Waiter
	ndone   int
	onEnd   chan Waiter
}

// NewGoGroupCaller creates an object capable of waiting until all supplied GoRoutines complete.
func NewGoGroupCaller(level int, group ...Waiter) *GoGroup {
	sfile, sline := FetchCaller(1 + level)
	var w *GoGroup = &GoGroup{ 
		srcFile: sfile,
		srcLine: sline,
		ndone: 0, 
		onEnd: make(chan Waiter),
	}
	for _, u := range group {
		w.Add(u)
	}
	return w
}

func NewGoGroup(group ...Waiter) *GoGroup {
	return NewGoGroupCaller(1, group...)
}

// String returns a unique, readable string representation of this instance.
func (t *GoGroup) String() string {
	return fmt.Sprintf("%s:%d (%x)", t.srcFile, t.srcLine, t)
}

// Add adds a Waiter to the group. It can be called at any time
// as long as the current set of goroutines hasn't completed.
// For instance, as long as you call Add from a GoRoutine which
// is waited on by this object, the condtion will be met.
func (t *GoGroup) Add(u Waiter) {
	t.lk.Lock()
	defer t.lk.Unlock()
	fmt.Printf("woa: + %s (%d)\n", u.String(), t.n)
	t.group = append(t.group, u)
}

// Go is a convenience method which forks f into a new GoRoutine and
// adds the latter to the waiting queue.
func (t *GoGroup) Go(f func()) {
	t.Add(GoCaller(f, 1))
}

// Wait blocks until all goroutines in the group have completed.
// Wait can be called concurrently. If called post-completion of the
// goroutine group, Wait returns immediately.
func (t *GoGroup) Wait() {

	// XXX ensure re-entrant

	// Start waiting
	t.lk.Lock()
	for _, u := range t.group {
		go func(){
			u.Wait()
			t.onEnd <- u
		}()
	}
	?
	n := len(t.group)
	t.lk.Unlock()
	if n == 0 {
		panic("waiting on 0 goroutines")
	}
	for t.stillRemain() {
		u, ok := <-t.onEnd
		if !ok {
			return
		}
		t.lk.Lock()
		t.n--
		fmt.Printf("woa: - %s (%d)\n", u.String(), t.n)
		if t.n == 0 {
			t.n = -1
			close(t.onEnd)
			fmt.Printf("woa: wait done\n")
		}
		t.lk.Unlock()
	}
}

func (t* GoGroup) stillRemain() bool {
	t.lk.Lock()
	defer t.lk.Unlock()
	return t.ndone < len(t.group)
}
