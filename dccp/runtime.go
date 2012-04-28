package dccp

import (
	"time"
)

// Runtime represents an environment of execution, and in particular the notion of time.
// Its methods provide access to that environment both in way of querying (e.g.
// "What time is it?") as well as in way of asking for actions (e.g. "Fork a
// new goroutine!") being taken.
type Runtime interface {

	// Sleep blocks for ns nanoseconds 
	Sleep(nsec int64)

	// Now returns the current time in nanoseconds since an abstract 0-moment
	Now() int64
}

// realRuntime is an implementation of Runtime that represents real time
type realRuntime struct {}

var RealRuntime realRuntime

func (realRuntime) Now() int64 {
	return time.Now().UnixNano()
}

func (realRuntime) Sleep(ns int64) {
	time.Sleep(time.Duration(ns))
}
