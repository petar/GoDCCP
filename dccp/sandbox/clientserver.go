// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"os"
	"path"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

// NewEnv creates a dccp.Env for test purposes, whose dccp.TraceWriter writes to a file
// and duplicates all emits to any number of additional guzzles, which are usually used to check
// test conditions. The TraceWriterPlex is returned to facilitate adding further guzzles.
func NewEnv(guzzleFilename string) (env *dccp.Env, plex *TraceWriterPlex) {
	fileTraceWriter := dccp.NewFileTraceWriter(path.Join(os.Getenv("DCCPLOG"), guzzleFilename + ".emit"))
	plex = NewTraceWriterPlex(fileTraceWriter)
	plex = NewSyncTraceWriter(plex)
	return dccp.NewEnv(plex), plex
}

// NewClientServerPipe creates a sandbox communication pipe and attaches a DCCP client and a DCCP
// server to its endpoints. In addition to sending all emits to a standard DCCP log file, it sends a
// copy of all emits to the dup TraceWriter.
func NewClientServerPipe(env *dccp.Env) (clientConn, serverConn *dccp.Conn, clientToServer, serverToClient *headerHalfPipe) {
	llog := dccp.NewAmb("line", env)
	hca, hcb, _ := NewPipe(env, llog, "client", "server")
	ccid := ccid3.CCID3{}

	clog := dccp.NewAmb("client", env)
	clientConn = dccp.NewConnClient(env, clog, hca, ccid.NewSender(env, clog), ccid.NewReceiver(env, clog), 0)

	slog := dccp.NewAmb("server", env)
	serverConn = dccp.NewConnServer(env, slog, hcb, ccid.NewSender(env, slog), ccid.NewReceiver(env, slog))

	return clientConn, serverConn, hca, hcb
}
