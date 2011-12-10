// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"os"
	"path"
	"testing"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

func makeEnds(logname string) (clientConn, serverConn *dccp.Conn, run *dccp.Runtime) {

	logwriter := dccp.NewFileLogWriter(path.Join(os.Getenv("DCCPLOG"), logname+"_test.emit"))
	run = dccp.NewRuntime(dccp.RealTime, logwriter)
	run.Filter().Select(
		"client", "server", "end", "line", "conn", "s", 
		"s-x", "s-strober", "s-tracker", 
		"r", "r-evolver",
	)

	llog := dccp.NewLogger("line", run)
	hca, hcb, _ := NewLine(run, llog, "client", "server", 1e9, 100)  // 100 packets per second
	ccid := ccid3.CCID3{}

	clog := dccp.NewLogger("client", run)
	clientConn = dccp.NewConnClient(run, clog, hca, 
		ccid.NewSender(run, clog), ccid.NewReceiver(run, clog), 0)

	slog := dccp.NewLogger("server", run)
	serverConn = dccp.NewConnServer(run, slog, hcb, 
		ccid.NewSender(run, slog), ccid.NewReceiver(run, slog))

	return clientConn, serverConn, run
}

func TestNop(t *testing.T) {
	dccp.InstallCtrlCPanic()
	dccp.InstallTimeout(10e9)
	_, _, run := makeEnds("openclose")
	run.Sleep(5e9)
}

func TestOpenClose(t *testing.T) {

	dccp.InstallCtrlCPanic()
	dccp.InstallTimeout(20e9)
	clientConn, serverConn, run := makeEnds("openclose")

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
	clientConn.Abort()
	serverConn.Abort()
	dccp.NewGoGroup(clientConn.Waiter(), serverConn.Waiter()).Wait() // XXX causes hang
	dccp.NewLogger("line", run).Emit("end", "end", nil, "Server and client done.")
	if err := run.Close(); err != nil {
		t.Errorf("Error closing runtime (%s)", err)
	}
	t.Logf("normal exit\n")
}

func TestIdle(t *testing.T) {

	clientConn, serverConn, run := makeEnds("idle")

	cchan := make(chan int, 1)
	go func() {
		run.Sleep(5e9) // Stay idle for 5sec
		if err := clientConn.Close(); err != nil {
			t.Errorf("client close error (%s)", err)
		}
		cchan <- 1
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		run.Sleep(7e9) // Stay idle for 5sec
		if err := serverConn.Close(); err != nil {
			t.Errorf("server close error (%s)", err)
		}
		schan <- 1
		close(schan)
	}()

	<-cchan
	<-schan
	clientConn.Abort()
	serverConn.Abort()
	dccp.NewGoGroup(clientConn.Waiter(), serverConn.Waiter()).Wait()
	dccp.NewLogger("line", run).Emit("end", "end", nil, "Server and client done.")
	if err := run.Close(); err != nil {
		t.Errorf("Error closing runtime (%s)", err)
	}
}
