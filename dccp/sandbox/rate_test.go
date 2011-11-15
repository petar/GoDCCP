// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"fmt"
	"os"
	"testing"
	"time"
	"github.com/petar/GoGauge/gauge"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

func TestOpenClose(t *testing.T) {

	dccp.SetLogWriter(dccp.NewFileLogWriter(os.Getenv("DCCPLOG")))
	gauge.Select("client", "server", "line", "conn", "s", "s-x", "s-strober", "s-tracker", "r")

	dccp.SetTime(dccp.RealTime)

	hca, hcb, _ := NewLine("line", "client", "server", 1e9, 10)  // 10 packets per second
	ccid := ccid3.CCID3{}

	clog := dccp.Logger("client")
	clientConn := dccp.NewConnClient(clog, hca, ccid.NewSender(clog), ccid.NewReceiver(clog), 0)
	go func() {
		time.Sleep(1e9)
		_, err := clientConn.ReadSegment()
		fmt.Printf("client/read err = %v\n", err)
	}()

	slog := dccp.Logger("server")
	serverConn := dccp.NewConnServer(slog, hcb, ccid.NewSender(slog), ccid.NewReceiver(slog))
	go func() {
		time.Sleep(1e9)
		if err := serverConn.Close(); err != nil {
			t.Errorf("server close error (%s)", err)
		}
	}()

	time.Sleep(40e9)
}
