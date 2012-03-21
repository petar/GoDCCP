// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"path"
	goruntime "runtime"
	"sync"
)

// GoRoutine represents a running goroutine.
type GoRoutine struct {
	ch   chan int
	file string
	line int
	anno string
}

// Go runs f in a new goroutine and returns a handle object, which can
// then be used for various synchronization mechanisms.
func GoCaller(f func(), level int, afmt string, aargs ...interface{}) *GoRoutine {
	sfile, sline := FetchCaller(2 + level)
	g := &GoRoutine{ 
		ch:   make(chan int, 0),
		file: sfile,
		line: sline,
		anno: fmt.Sprintf(afmt, aargs...),
	}
	go func() {
		f()
		close(g.ch)
	}()
	return g
}

func Go(f func(), afmt string, aargs ...interface{}) *GoRoutine {
	return GoCaller(f, 1, afmt, aargs...)
}

// Wait blocks until the goroutine completes; otherwise,
// if the goroutine has completed, it returns immediately.
// Wait can be called concurrently.
func (g *GoRoutine) Wait() {
	_, _ = <-g.ch
	fmt.Printf("GoRoutine %s: done\n", g.String())
}

// Source returns the file and line where the goroutine was forked.
func (g *GoRoutine) Source() (sfile string, sline int) {
	return g.file, g.line
}

// Strings returns a unique and readable string representation of this instance.
func (g *GoRoutine) String() string {
	sfile, sline := g.Source()
	return fmt.Sprintf("%s:%d %s (%p)", sfile, sline, g.anno, g)
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
type Waiter interface {
	// Wait blocks until an underlying event occurs.
	// It returns immediately if called post-event.
	// It is re-entrant.
	Wait()
	String() string
}

// GoConjunction waits until a set of GoRoutines all complete. It also allows
// new routines to be added dynamically before the completion event.
type GoConjunction struct {
	srcFile    string
	srcLine    int
	annotation string

	lk      sync.Mutex	// Locks the fields below
	group   []Waiter	// Slice of waiters included in this conjunction sync
	kdone   int		// Counts the number of Waiters that have already completed
	ch      chan Waiter
	slk     sync.Mutex	// Only one Wait can be called at a time
}

// NewGoConjunctionCaller creates an object capable of waiting until all supplied GoRoutines complete.
func NewGoConjunctionCaller(level int, annotation string, group ...Waiter) *GoConjunction {
	sfile, sline := FetchCaller(1 + level)
	var w *GoConjunction = &GoConjunction{ 
		srcFile:    sfile,
		srcLine:    sline,
		annotation: annotation,
		kdone:      0, 
		ch:         make(chan Waiter, 10),
	}
	for _, u := range group {
		w.Add(u)
	}
	return w
}

func NewGoConjunction(annotation string, group ...Waiter) *GoConjunction {
	return NewGoConjunctionCaller(1, annotation, group...)
}

// String returns a unique, readable string representation of this instance.
func (t *GoConjunction) String() string {
	return fmt.Sprintf("%s:%d %s (%x)", t.srcFile, t.srcLine, t.annotation, t)
}

// Add adds a Waiter to the group. It can be called at any time
// as long as the current set of goroutines hasn't completed.
// For instance, as long as you call Add from a GoRoutine which
// is waited on by this object, the condtion will be met.
func (t *GoConjunction) Add(u Waiter) {
	t.lk.Lock()
	defer t.lk.Unlock()
	if t.kdone < 0 {
		panic("adding waiters after conjunction event")
	}
	t.group = append(t.group, u)
	ch := t.ch
	go func(){
		u.Wait()
		ch <- u
	}()
}

// Go is a convenience method which forks f into a new GoRoutine and
// adds the latter to the waiting queue. fmt is a formatted annotation
// with arguments args.
func (t *GoConjunction) Go(f func(), afmt string, aargs ...interface{}) {
	t.Add(GoCaller(f, 1, afmt, aargs...))
}

// Wait blocks until all goroutines in the group have completed.
// Wait can be called concurrently. If called post-completion of the
// goroutine group, Wait returns immediately.
func (t *GoConjunction) Wait() {
	t.slk.Lock()
	defer t.slk.Unlock()

	// Prevent calling Wait before any waitees have been added
	t.lk.Lock()
	n := len(t.group)
	t.lk.Unlock()
	if n == 0 {
		panic("waiting on 0 goroutines")
	}

	for t.stillRemain() {
		_, ok := <-t.ch
		if !ok {
			fmt.Printf("Wait (%s) returns on closed chan\n", t.String())
			return
		}
		t.lk.Lock()
		if t.kdone < 0 {
			panic("no waiters should complete past junction event")
		}
		t.kdone++
		fmt.Printf("GoConj %s: %d/%d done\n", t.String(), t.kdone, len(t.group))
		if t.kdone == len(t.group) {
			// Ensure future calls to Wait return immediately
			t.kdone = -1
			close(t.ch)
		}
		t.lk.Unlock()
	}
	fmt.Printf("Wait (%s) returns on no more remain\n", t.String())
}

func (t* GoConjunction) stillRemain() bool {
	t.lk.Lock()
	defer t.lk.Unlock()
	return t.kdone > 0 && t.kdone < len(t.group)
}
