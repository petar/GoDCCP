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
)

func TestDropRate(t *testing.T) {

	gauge.Select("client", "server", "line", "conn", "s", "s-x", "s-strober", "s-tracker", "r")

	var tt dccp.Time = dccp.RealTime{}

	hca, hcb, _ := NewLine(dccp.NewLogger(tt, "line"), "client", "server", 1e9, 10)
	ccid := ccid3.CCID3{}

	clog := dccp.NewLogger(tt, "client")
	/* cc := */ dccp.NewConnClient(tt, clog, hca, ccid.NewSender(tt, clog), ccid.NewReceiver(tt, clog), 0)

	slog := dccp.NewLogger(tt, "server")
	/* cs := */ dccp.NewConnServer(tt, slog, hcb, ccid.NewSender(tt, slog), ccid.NewReceiver(tt, slog))

	time.Sleep(10e9)
}
