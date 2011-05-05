// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"log"
	"os/signal"
)

// InstallCtrlC() installs a Ctrl-C signal handler that panics
func InstallCtrlC() {
	go func() {
		for s := range signal.Incoming {
			if s == signal.Signal(signal.SIGINT) {
				log.Printf("signal interruption: %s\n", s)
				panic("ctrl-c")
			}
		}
	}()
}
