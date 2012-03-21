// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	t "log"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/sandbox"
	"time"
)

func main() {
	dccp.InstallCtrlCPanic()
	dccp.InstallTimeout(10e9)
	clientConn, serverConn, run := sandbox.NewClientServerPipe("openclose")

	cchan := make(chan int, 1)
	go func() {
		run.Sleep(2e9)
		_, err := clientConn.ReadSegment()
		if err != dccp.ErrEOF {
			t.Fatalf("client read error (%s), expected EBADF", err)
		}
		cchan <- 1
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		run.Sleep(1e9)
		if err := serverConn.Close(); err != nil {
			t.Fatalf("server close error (%s)", err)
		}
		schan <- 1
		close(schan)
	}()

	<-cchan
	<-schan

	// Abort casuses both connection to wrap up the connection quickly
	clientConn.Abort()
	serverConn.Abort()
	// However, even aborting leaves various connection go-routines lingering for a short while.
	// The next line ensures that we wait until all go routines are done.
	// dccp.NewGoConjunction("end-of-test", clientConn.Waiter(), serverConn.Waiter()).Wait() // XXX causes hang
	time.Sleep(time.Second*10)

	dccp.NewLogger("line", run).E("end", "end", "Server and client done.")
	if err := run.Close(); err != nil {
		t.Fatalf("Error closing runtime (%s)", err)
	}
}
