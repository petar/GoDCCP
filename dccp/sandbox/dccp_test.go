// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"os"
	"testing"
	"github.com/petar/GoGauge/gauge"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

func makeEnds(logname string) (clientConn, serverConn *dccp.Conn, log *dccp.FileLogWriter) {
	logf := dccp.NewFileLogWriter(os.Getenv("DCCPLOG")+"-"+logname)
	dccp.SetLogWriter(logf)
	gauge.Select("client", "server", "line", "conn", "s", "s-x", "s-strober", "s-tracker", "r")

	dccp.SetTime(dccp.RealTime)

	hca, hcb, _ := NewLine("line", "client", "server", 1e9, 100)  // 100 packets per second
	ccid := ccid3.CCID3{}

	clog := dccp.Logger("client")
	clientConn = dccp.NewConnClient(clog, hca, ccid.NewSender(clog), ccid.NewReceiver(clog), 0)

	slog := dccp.Logger("server")
	serverConn = dccp.NewConnServer(slog, hcb, ccid.NewSender(slog), ccid.NewReceiver(slog))

	return clientConn, serverConn, logf
}

func TestOpenClose(t *testing.T) {

	dccp.InstallCtrlCPanic()
	clientConn, serverConn, logf := makeEnds("openclose")

	cchan := make(chan int, 1)
	go func() {
		dccp.Sleep(2e9)
		_, err := clientConn.ReadSegment()
		if err != os.EBADF {
			t.Errorf("client read error (%s), expected EBADF", err)
		}
		cchan <- 1
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		dccp.Sleep(1e9)
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
	if err := logf.Close(); err != nil {
		t.Errorf("Error closing log (%s)", err)
	}
}

/*
func TestIdle(t *testing.T) {

	clientConn, serverConn, logf := makeEnds("idle")

	cchan := make(chan int, 1)
	go func() {
		dccp.Sleep(5e9) // Stay idle for 5sec
		cchan <- 1
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		dccp.Sleep(5e9) // Stay idle for 5sec
		schan <- 1
		close(schan)
	}()

	<-cchan
	<-schan
	clientConn.Abort()
	serverConn.Abort()
	if err := logf.Close(); err != nil {
		t.Errorf("Error closing log (%s)", err)
	}
}
*/
