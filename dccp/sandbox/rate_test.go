// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"testing"
	"time"
	"github.com/petar/GoGauge/gauge"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
	dgauge "github.com/petar/GoDCCP/dccp/gauge"
)

func TestDropRate(t *testing.T) {

	logEmitter := dgauge.NewD3() // dgauge.NewLogReducer()
	dccp.SetLogEmitter(logEmitter)

	gauge.Select("client", "server", "line", "conn", "s", "s-x", "s-strober", "s-tracker", "r")

	dccp.SetTime(dccp.RealTime)

	hca, hcb, _ := NewLine("line", "client", "server", 1e9, 10)
	ccid := ccid3.CCID3{}

	clog := dccp.Logger("client")
	/* cc := */ dccp.NewConnClient(clog, hca, ccid.NewSender(clog), ccid.NewReceiver(clog), 0)

	slog := dccp.Logger("server")
	/* cs := */ dccp.NewConnServer(slog, hcb, ccid.NewSender(slog), ccid.NewReceiver(slog))

	time.Sleep(10e9)

	logData := logEmitter.Close()
	dgauge.OutToFile("DCCPD3", logData)
}
