// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"fmt"
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

// TestRate tests whether a single connection's one-way client-to-server rate converges to
// limit imposed by connection in that the send rate has to:
//	(1) converge and stabilize, and
//	(2) the stable rate should 
//		(2.a) either be closely below the connection limit,
//		(2.b) or be closely above the connection limit (and maintain a drop rate below some threshold)
// A two-way test is not necessary as the congestion mechanisms in either direction are completely independent.
func TestRate(t *testing.T) {

	run, _ := NewRuntime("rate")
	clientConn, serverConn, _, _ := NewClientServerPipe(run)

	cchan := make(chan int, 1)
	mtu := clientConn.GetMTU()
	buf := make([]byte, mtu)
	go func() {
		t0 := run.Now()
		for run.Now() - t0 < 10e9 {
			err := clientConn.Write(buf)
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
			_, err := serverConn.Read()
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
