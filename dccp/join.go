// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"path"
	goruntime "runtime"
	"strings"
	"sync"
)

// Joiner is an interface to objects that can wait for some event.
type Joiner interface {
	// Join blocks until an underlying event occurs.
	// It returns immediately if called post-event.
	// It is re-entrant.
	Join()
	String() string
}

// GoRoutine represents a running goroutine.
type GoRoutine struct {
	ch   chan int
	file string
	line int
	anno string
}

// Go runs f in a new goroutine and returns a handle object, which can
// then be used for various synchronization mechanisms.
func GoCaller(run Runtime, f func(), skip int, fmt_ string, args_ ...interface{}) *GoRoutine {
	sfile, sline := FetchCaller(1 + skip)
	ch := make(chan int)
	g := &GoRoutine{ 
		ch:   ch,
		file: sfile,
		line: sline,
		anno: fmt.Sprintf(fmt_, args_...),
	}
	run.Go(func() {
		f()
		close(ch)
	})
	return g
}

func Go(run Runtime, f func(), fmt_ string, args_ ...interface{}) *GoRoutine {
	return GoCaller(run, f, 1, fmt_, args_...)
}

// Join blocks until the goroutine completes; otherwise,
// if the goroutine has completed, it returns immediately.
// Join can be called concurrently.
func (g *GoRoutine) Join() {
	_, _ = <-g.ch
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
func FetchCaller(skip int) (sfile string, sline int) {
	_, sfile, sline, _ = goruntime.Caller(1+skip)
	return TrimSourceFile(sfile), sline
}

// TrimSourceFile shortens DCCP source file names for readability
func TrimSourceFile(sfile string) string {
	sdir, sfile := path.Split(sfile)
	if len(sdir) > 0 {
		_, sdir = path.Split(sdir[:len(sdir)-1])
	}
	sfile = path.Join(sdir, sfile)
	return sfile
}

// TrimFuncName shortens DCCP function names for readability and whether this is a DCCP or anonymous function
func TrimFuncName(fname string) (trimmed string, isDCCP bool) {
	const pkgName = "dccp"
	k := strings.Index(fname, pkgName) // Find first occurrence of package name
	if k < 0 {
		return fname, strings.HasPrefix(fname, "_func_") // Whether this is an anonymous func
	}
	k += len(pkgName)
	if len(fname) <= k {
		return fname, false
	}
	switch fname[k] {
	case '.', '/':
		return fname[k+1:], true
	}
	return fname, false
}

// GoJoin waits until a set of GoRoutines all complete. It also allows
// new routines to be added dynamically before the completion event.
type GoJoin struct {
	run        Runtime
	srcFile    string
	srcLine    int
	annotation string

	lk      sync.Mutex	// Locks the fields below
	group   []Joiner	// Slice of joiners included in this conjunction sync
	kdone   int		// Counts the number of Joiners that have already completed
	ch      chan Joiner
	slk     sync.Mutex	// Only one Join can be called at a time
}

// NewGoJoinCaller creates an object capable of waiting until all supplied GoRoutines complete.
func NewGoJoinCaller(run Runtime, skip int, annotation string, group ...Joiner) *GoJoin {
	sfile, sline := FetchCaller(1 + skip)
	var w *GoJoin = &GoJoin{ 
		run:        run,
		srcFile:    sfile,
		srcLine:    sline,
		annotation: annotation,
		kdone:      0, 
		ch:         make(chan Joiner, 10),
	}
	for _, u := range group {
		w.Add(u)
	}
	return w
}

func NewGoJoin(run Runtime, annotation string, group ...Joiner) *GoJoin {
	return NewGoJoinCaller(run, 1, annotation, group...)
}

// String returns a unique, readable string representation of this instance.
func (t *GoJoin) String() string {
	return fmt.Sprintf("%s:%d %s (%x)", t.srcFile, t.srcLine, t.annotation, t)
}

// Add adds a Joiner to the group. It can be called at any time
// as long as the current set of goroutines hasn't completed.
// For instance, as long as you call Add from a GoRoutine which
// is waited on by this object, the condtion will be met.
func (t *GoJoin) Add(u Joiner) {
	t.lk.Lock()
	defer t.lk.Unlock()
	if t.kdone < 0 {
		panic("adding joiners after conjunction event")
	}
	t.group = append(t.group, u)
	ch := t.ch
	t.run.Go(func(){
		u.Join()
		ch <- u
	})
}

// Go is a convenience method which forks f into a new GoRoutine and
// adds the latter to the waiting queue. fmt is a formatted annotation
// with arguments args.
func (t *GoJoin) Go(f func(), fmt_ string, args_ ...interface{}) {
	t.Add(GoCaller(t.run, f, 1, fmt_, args_...))
}

// Join blocks until all goroutines in the group have completed.
// Join can be called concurrently. If called post-completion of the
// goroutine group, Join returns immediately.
func (t *GoJoin) Join() {
	t.slk.Lock()
	defer t.slk.Unlock()

	// Prevent calling Join before any waitees have been added
	t.lk.Lock()
	n := len(t.group)
	ch := t.ch
	t.lk.Unlock()
	if n == 0 {
		panic("waiting on 0 goroutines")
	}

	for t.stillRemain() {
		_, ok := <-ch
		if !ok {
			return
		}
		t.lk.Lock()
		if t.kdone < 0 {
			panic("no joiners should complete past junction event")
		}
		t.kdone++
		if t.kdone == len(t.group) {
			// Ensure future calls to Join return immediately
			t.kdone = -1
			close(ch)
		}
		t.lk.Unlock()
	}
}

func (t* GoJoin) stillRemain() bool {
	t.lk.Lock()
	defer t.lk.Unlock()
	return t.kdone >= 0 && t.kdone < len(t.group)
}
