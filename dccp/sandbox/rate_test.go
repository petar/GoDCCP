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
	hca, hcb, _ := NewLine("line", "client", "server", 10)
	ccid := ccid3.CCID3{}
	gauge.Select("client", "server", /* "line",*/ "conn", "s", "s-x", "s-strober", "s-tracker", "r")
	/* cc := */ dccp.NewConnClient("client", hca, ccid.NewSender(), ccid.NewReceiver(), 0)
	/* cs := */ dccp.NewConnServer("server", hcb, ccid.NewSender(), ccid.NewReceiver())
	time.Sleep(30e9)
}
