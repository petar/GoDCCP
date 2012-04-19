// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

// TestRoundtripEstimation checks that round-trip times are estimated accurately.
func TestRoundtripEstimation(t *testing.T) {
	reducer := newRoundtripMeasure(t)
	clientConn, serverConn, run, clientToServer, _ := NewClientServerPipeDup(
		"rtt", NewGuzzlePlex(reducer, ??),
	)

	// Roundtrip estimates might be imprecise during long idle periods,
	// as a product of the CCID3 design, since during such period precise
	// estimates are not necessary. Therefore, to focus on roundtrip time
	// estimation without saturating the link, we generate sufficiently 
	// regular transmissions.

	cargo := []byte{1, 2, 3}
	buf := make([]byte, len(cargo))
	const (
		duration = 10e9              // Duration of the experiment = 10 sec
		interval = 100e6             // How often we perform heartbeat writes to avoid idle periods = 100 ms
		rate     = 1e9 / interval    // Fixed send rate for both endpoints in packets per second = 10 pps
		latency  = 50e6              // Latency of 50ms
	)

	// In order to isolate roundtrip measurement testing from the complexities
	// of the send rate calculation mechanism, we fix the send rate of both
	// endpoints using the debug flag FixRate.
	clientConn.Amb().Flags().SetUint32("FixRate", rate)
	serverConn.Amb().Flags().SetUint32("FixRate", rate)

	// Increase the client—>server latency from 0 to latency at half time
	go func() {
		run.Sleep(duration / 2)
		clientToServer.SetWriteLatency(latency)
	}()

	cchan := make(chan int, 1)
	go func() {
		t0 := run.Now()
		for run.Now() - t0 < duration {
			err := clientConn.WriteSegment(buf)
			if err != nil {
				break
			}
			run.Sleep(interval)
		}
		// Close is necessary because otherwise, if no read timeout is in place, the
		// server sides hangs forever on ReadSegment
		clientConn.Close()
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
