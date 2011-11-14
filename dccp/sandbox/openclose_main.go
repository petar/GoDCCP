// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/petar/GoGauge/gauge"
	"github.com/petar/GoDCCP/dccp/sandbox"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

func main() {

	gauge.Select("client", "server", "line", "conn", "s", "s-x", "s-strober", "s-tracker", "r")

	dccp.SetTime(dccp.RealTime)

	hca, hcb, _ := sandbox.NewLine("line", "client", "server", 1e9, 10)  // 10 packets per second
	ccid := ccid3.CCID3{}

	clog := dccp.Logger("client")
	clientConn := dccp.NewConnClient(clog, hca, ccid.NewSender(clog), ccid.NewReceiver(clog), 0)
	go func() {
		dccp.Sleep(1e9)
		_, err := clientConn.ReadSegment()
		fmt.Printf("client/read err = %v\n", err)
	}()

	slog := dccp.Logger("server")
	serverConn := dccp.NewConnServer(slog, hcb, ccid.NewSender(slog), ccid.NewReceiver(slog))
	go func() {
		dccp.Sleep(1e9)
		if err := serverConn.Close(); err != nil {
			fmt.Printf("server close error (%s)\n", err)
		}
	}()

	dccp.Sleep(40e9)
}
