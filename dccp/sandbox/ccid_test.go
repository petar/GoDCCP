// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"io"
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

func TestRateConvergence(t *testing.T) {

	dccp.InstallCtrlCPanic()
	clientConn, serverConn, run := makeEnds("converge")

	cchan := make(chan int, 1)
	mtu := clientConn.GetMTU()
	buf := make([]byte, mtu)
	go func() {
		t0 := run.Nanoseconds()
		for run.Nanoseconds() - t0 < 10e9 {
			err := clientConn.WriteSegment(buf)
			if err != nil {
				t.Errorf("error writing (%s)", err)
			}
		}
		clientConn.Close()
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		for {
			_, err := serverConn.ReadSegment()
			if err == io.EOF {
				break 
			} else if err != nil {
				t.Errorf("error reading (%s)", err)
			}
		}
		close(schan)
	}()

	_, _ = <-cchan
	_, _ = <-schan
	dccp.MakeConjWaiter(clientConn.Waiter(), serverConn.Waiter()).Wait()
	dccp.NewLogger("line", run).Emit("end", "end", nil, "Server and client done.")
	if err := run.Close(); err != nil {
		t.Errorf("error closing runtime (%s)", err)
	}
}
