// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"github.com/petar/GoGauge/gauge"
	"github.com/petar/GoDCCP/dccp/sandbox"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

func main() {
	dccp.InstallCtrlCPanic()
	dccp.SetLogWriter(dccp.NewFileLogWriter(os.Getenv("DCCPLOG")))

	gauge.Select("client", "server", "line", "conn", "s", "s-x", "s-strober", "s-tracker", "r")

	dccp.SetTime(dccp.RealTime)

	hca, hcb, _ := sandbox.NewLine("line", "client", "server", 1e9, 10)  // 10 packets per second
	ccid := ccid3.CCID3{}

	clog := dccp.Logger("client")
	clientConn := dccp.NewConnClient(clog, hca, ccid.NewSender(clog), ccid.NewReceiver(clog), 0)
	cchan := make(chan int, 1)
	go func() {
		dccp.Sleep(1e9)
		_, err := clientConn.ReadSegment()
		fmt.Printf("client/read err = %v\n", err)
		cchan <- 1
		close(cchan)
	}()

	slog := dccp.Logger("server")
	serverConn := dccp.NewConnServer(slog, hcb, ccid.NewSender(slog), ccid.NewReceiver(slog))
	schan := make(chan int, 1)
	go func() {
		dccp.Sleep(1e9)
		if err := serverConn.Close(); err != nil {
			fmt.Printf("server close error (%s)\n", err)
		}
		schan <- 1
		close(schan)
	}()

	<-cchan
	<-schan
}
