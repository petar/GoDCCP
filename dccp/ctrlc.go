// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// InstallCtrlCPanic installs a Ctrl-C signal handler that panics
func InstallCtrlCPanic() {
	go func() {
		defer SavePanicTrace()
		for s := range signal.Incoming {
			//if s == os.Signal(syscall.SIGINT) {
				log.Printf("ctrl-c interruption: %s\n", s)
				panic("ctrl-c")
			//}
		}
	}()
}

func SavePanicTrace() {
	r := recover()
	if r == nil {
		return
	}
	// Redirect stderr
	file, err := os.Create("panic")
	if err != nil {
		panic("dumper (no file) " + r.(fmt.Stringer).String())
	}
	syscall.Dup2(file.Fd(), os.Stderr.Fd())
	panic("dumper " + r.(string))
}
