// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp


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
// if the goroutine has completed, it returns immediately
func (g *GoRoutine) Wait() {
	_, _ = <-g.onEnd
}

// WaitAll blocks until all goroutines complete
func WaitAll(goroutines ...*GoRoutine) {
	var x = make(chan int)
	for _, g := range goroutines {
		go func() {
			g.Wait()
			x <- 1
		}()
	}
	for i := 0; i < len(goroutines); i++ {
		<-x
	}
}
