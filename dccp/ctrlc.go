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
	"time"
)

// InstallTimeout panics the current process in ns time
func InstallTimeout(ns int64) {
	go func() {
		time.Sleep(time.Duration(ns))
		panic("process timeout")
	}()
}

// InstallCtrlCPanic installs a Ctrl-C signal handler that panics
func InstallCtrlCPanic() {
	go func() {
		defer SavePanicTrace()
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt)
		for s := range ch {
			log.Printf("ctrl-c interruption: %s\n", s)
			panic("ctrl-c")
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
	syscall.Dup2(int(file.Fd()), int(os.Stderr.Fd()))
	// TRY: defer func() { file.Close() }()
	panic("dumper " + r.(string))
}
