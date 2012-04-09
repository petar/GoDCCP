// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"fmt"
	//"os"
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

// rttReducer is a dccp.Guzzle which listens to the logs
// emitted from the RTT test and performs various checks.
type rttReducer struct {
	t *testing.T
}

func (t *rttReducer) Write(r *dccp.LogRecord) {
	/*
	switch r.Event {
	case "rrtt": 
		rtt, _ := r.Args.Int64("rtt")
		fmt.Printf("%s rRTT: %d\n", r.Module, rtt)
	case "srtt":
		rtt, _ := r.Args.Int64("rtt")
		est, _ := r.Args.Bool("est")
		fmt.Printf("%s sRTT: %d %v\n", r.Module, rtt, est)
	}
	*/
}

func (t *rttReducer) Sync() error { 
	return nil 
}

func (t *rttReducer) Close() error { 
	return nil 
}

// TestRoundtripEstimation checks that round-trip times are estimated accurately.
func TestRoundtripEstimation(t *testing.T) {
	reducer := &rttReducer{t}
	clientConn, serverConn, run := NewClientServerPipeDup("rtt", reducer)

	// Roundtrip estimates might be imprecise during long idle periods,
	// as a product of the CCID3 design, since during such period precise
	// estimates are not necessary. Therefore, to focus on roundtrip time
	// estimation without saturating the link, we generate sufficiently 
	// regular transmissions.

	cargo := []byte{1, 2, 3}
	buf := make([]byte, len(cargo))
	const (
		duration = 20e9
		interval = 100e6
	)

	cchan := make(chan int, 1)
	go func() {
		t0 := run.Nanoseconds()
		for run.Nanoseconds() - t0 < duration {
			err := clientConn.WriteSegment(buf)
			if err != nil {
				break
			}
			run.Sleep(interval)
		}
		close(cchan)
	}()

	schan := make(chan int, 1)
	go func() {
		for {
			_, err := serverConn.ReadSegment()
			if err != nil {
				break
			}
		}
		close(schan)
	}()

	_, _ = <-cchan
	_, _ = <-schan

	// Shutdown the connections properly
	clientConn.Abort()
	serverConn.Abort()
	dccp.NewGoConjunction("end-of-test", clientConn.Waiter(), serverConn.Waiter()).Wait()
	dccp.NewAmb("line", run).E(dccp.EventMatch, "Server and client done.")
	if err := run.Close(); err != nil {
		t.Errorf("error closing runtime (%s)", err)
	}
}

func TestConverge(t *testing.T) {

	dccp.InstallCtrlCPanic()
	clientConn, serverConn, run := NewClientServerPipe("converge")

	cchan := make(chan int, 1)
	mtu := clientConn.GetMTU()
	buf := make([]byte, mtu)
	go func() {
		t0 := run.Nanoseconds()
		for run.Nanoseconds() - t0 < 10e9 {
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
