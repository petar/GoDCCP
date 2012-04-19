// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

// TestNop checks that no panics occur in the first 5 seconds of connection establishment
func TestNop(t *testing.T) {
	// dccp.InstallCtrlCPanic()
	// dccp.InstallTimeout(10e9)
	run, _ := NewRuntime("nop")
	NewClientServerPipe(run)
	run.Sleep(5e9)
}

// TestOpenClose verifies that connect and close handshakes function correctly
func TestOpenClose(t *testing.T) {
	run, _ := NewRuntime("openclose")
	clientConn, serverConn, _, _ := NewClientServerPipe(run)

	cchan := make(chan int, 1)
	go func() {
		run.Sleep(2e9)
		_, err := clientConn.ReadSegment()
		if err != dccp.ErrEOF {
			t.Errorf("client read error (%s), expected EBADF", err)
		}
		cchan <- 1
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		run.Sleep(1e9)
		if err := serverConn.Close(); err != nil {
			t.Errorf("server close error (%s)", err)
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
	dccp.NewGoConjunction("end-of-test", clientConn.Waiter(), serverConn.Waiter()).Wait()

	dccp.NewAmb("line", run).E(dccp.EventMatch, "Server and client done.")
	if err := run.Close(); err != nil {
		t.Errorf("Error closing runtime (%s)", err)
	}
}

// Idle keeps the connection between a client and server idle for a few seconds and makes sure that
// no unusual behavior occurs.
func TestIdle(t *testing.T) {

	run, _ := NewRuntime("idle")
	clientConn, serverConn, _, _ := NewClientServerPipe(run)
	cargo := []byte{1, 2, 3}

	cchan := make(chan int, 1)
	go func() {
		if err := clientConn.WriteSegment(cargo); err != nil {
			t.Errorf("client write (%s)", err)
		}
		run.Sleep(10e9) // Stay idle for 10 sec
		if err := clientConn.Close(); err != nil && err != dccp.ErrEOF {
			t.Errorf("client close (%s)", err)
		}
		cchan <- 1
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		if err := serverConn.WriteSegment(cargo); err != nil {
			t.Errorf("server write (%s)", err)
		}
		run.Sleep(10e9) // Stay idle for 10 sec
		if err := serverConn.Close(); err != nil && err != dccp.ErrEOF {
			// XXX why not EOF
			t.Logf("server close (%s)", err)
		}
		schan <- 1
		close(schan)
	}()

	<-cchan
	<-schan
	clientConn.Abort()
	serverConn.Abort()
	dccp.NewGoConjunction("end-of-test", clientConn.Waiter(), serverConn.Waiter()).Wait()

	dccp.NewAmb("line", run).E(dccp.EventMatch, "Server and client done.")
	if err := run.Close(); err != nil {
		t.Errorf("Error closing runtime (%s)", err)
	}
}
