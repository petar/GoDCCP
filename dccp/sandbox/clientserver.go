// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"os"
	"path"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

func NewClientServerPipe(logname string) (clientConn, serverConn *dccp.Conn, run *dccp.Runtime) {
	return NewClientServerPipeDup(logname, nil)
}

// NewClientServerPipeDup creates a sandbox communication pipe and attaches a DCCP client and a DCCP
// server to its endpoints. In addition to sending all emits to a standard DCCP log file, it sends a
// copy of all emits to the dup LogWriter.
func NewClientServerPipeDup(logname string, dup dccp.LogWriter) (clientConn, serverConn *dccp.Conn, run *dccp.Runtime) {

	logwriter := dccp.NewFileLogWriterDup(path.Join(os.Getenv("DCCPLOG"), logname+".emit"), dup)
	run = dccp.NewRuntime(dccp.RealTime, logwriter)
	run.Filter().Select(
		"client", "server", "end", "line", "conn", "s", 
		"s-x", "s-strober", "s-tracker", 
		"r", "r-evolver",
	)

	llog := dccp.NewLogger("line", run)
	hca, hcb, _ := NewPipe(run, llog, "client", "server")
	ccid := ccid3.CCID3{}

	clog := dccp.NewLogger("client", run)
	clientConn = dccp.NewConnClient(run, clog, hca, 
		ccid.NewSender(run, clog), ccid.NewReceiver(run, clog), 0)

	slog := dccp.NewLogger("server", run)
	serverConn = dccp.NewConnServer(run, slog, hcb, 
		ccid.NewSender(run, slog), ccid.NewReceiver(run, slog))

	return clientConn, serverConn, run
}
