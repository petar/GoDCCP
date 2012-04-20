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

// NewRuntime creates a dccp.Runtime for test purposes, whose dccp.Guzzle writes to a file
// and duplicates all emits to any number of additional guzzles, which are usually used to check
// test conditions. The GuzzlePlex is returned to facilitate adding further guzzles.
func NewRuntime(guzzleFilename string, guzzles ...dccp.Guzzle) (run *dccp.Runtime, plex *GuzzlePlex) {
	fileGuzzle := dccp.NewFileGuzzle(path.Join(os.Getenv("DCCPLOG"), guzzleFilename + ".emit"))
	plex = NewGuzzlePlex(append(guzzles, fileGuzzle)...)
	return dccp.NewRuntime(dccp.RealTime, plex), plex
}

// NewClientServerPipe creates a sandbox communication pipe and attaches a DCCP client and a DCCP
// server to its endpoints. In addition to sending all emits to a standard DCCP log file, it sends a
// copy of all emits to the dup Guzzle.
func NewClientServerPipe(run *dccp.Runtime) (clientConn, serverConn *dccp.Conn, clientToServer, serverToClient *headerHalfPipe) {
	llog := dccp.NewAmb("line", run)
	hca, hcb, _ := NewPipe(run, llog, "client", "server")
	ccid := ccid3.CCID3{}

	clog := dccp.NewAmb("client", run)
	clientConn = dccp.NewConnClient(run, clog, hca, ccid.NewSender(run, clog), ccid.NewReceiver(run, clog), 0)

	slog := dccp.NewAmb("server", run)
	serverConn = dccp.NewConnServer(run, slog, hcb, ccid.NewSender(run, slog), ccid.NewReceiver(run, slog))

	return clientConn, serverConn, hca, hcb
}
