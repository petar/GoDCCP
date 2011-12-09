// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
	"github.com/petar/GoDCCP/dccp/sandbox"
)

func makeEnds(logname string) (clientConn, serverConn *dccp.Conn, run *dccp.Runtime) {

	logwriter := dccp.NewFileLogWriter(path.Join(os.Getenv("DCCPLOG"), logname+"_test.emit"))
	run = dccp.NewRuntime(dccp.RealTime, logwriter)
	run.Filter().Select(
		"client", "server", "end", "line", "conn", "s", 
		"s-x", "s-strober", "s-tracker", 
		"r", "r-evolver",
	)

	llog := dccp.NewLogger("line", run)
	hca, hcb, _ := sandbox.NewLine(run, llog, "client", "server", 1e9, 100)  // 100 packets per second
	ccid := ccid3.CCID3{}

	clog := dccp.NewLogger("client", run)
	clientConn = dccp.NewConnClient(run, clog, hca, 
		ccid.NewSender(run, clog), ccid.NewReceiver(run, clog), 0)

	slog := dccp.NewLogger("server", run)
	serverConn = dccp.NewConnServer(run, slog, hcb, 
		ccid.NewSender(run, slog), ccid.NewReceiver(run, slog))

	return clientConn, serverConn, run
}

func main() {
	dccp.InstallCtrlCPanic()
	clientConn, serverConn, run := makeEnds("converge")
	fmt.Printf("post-make\n")

	cchan := make(chan int, 1)
	mtu := clientConn.GetMTU()
	buf := make([]byte, mtu)
	go func() {
		t0 := run.Nanoseconds()
		for run.Nanoseconds() - t0 < 10e9 {
			fmt.Printf("pre-write\n")
			err := clientConn.WriteSegment(buf)
			if err != nil {
				fmt.Printf("error writing (%s)", err)
			}
			fmt.Printf("post-write\n")
		}
		clientConn.Close()
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		for {
			fmt.Printf("pre-read\n")
			_, err := serverConn.ReadSegment()
			fmt.Printf("post-read\n")
			if err == io.EOF {
				break 
			} else if err != nil {
				fmt.Printf("error reading (%s)", err)
			}
		}
		close(schan)
	}()

	_, _ = <-cchan
	_, _ = <-schan
	dccp.WaitOnAll(clientConn.Waiter(), serverConn.Waiter()).Wait()
	dccp.NewLogger("line", run).Emit("end", "end", nil, "Server and client done.")
	if err := run.Close(); err != nil {
		fmt.Printf("error closing runtime (%s)", err)
	}
}
