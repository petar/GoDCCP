// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"fmt"
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

func TestConverge(t *testing.T) {

	dccp.InstallCtrlCPanic()
	clientConn, serverConn, run, _, _ := NewClientServerPipe("converge")

	cchan := make(chan int, 1)
	mtu := clientConn.GetMTU()
	buf := make([]byte, mtu)
	go func() {
		t0 := run.Now()
		for run.Now() - t0 < 10e9 {
			err := clientConn.WriteSegment(buf)
			if err != nil {
				t.Errorf("error writing (%s)", err)
				break
			}
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
			if err == dccp.ErrEOF {
				break 
			} else if err != nil {
				t.Errorf("error reading (%s)", err)
				break
			}
		}
		close(schan)
	}()

	_, _ = <-cchan
	_, _ = <-schan
	dccp.NewGoConjunction("end-of-test", clientConn.Waiter(), serverConn.Waiter()).Wait()
	dccp.NewAmb("line", run).E(dccp.EventMatch, "Server and client done.")
	if err := run.Close(); err != nil {
		t.Errorf("error closing runtime (%s)", err)
	}
}
