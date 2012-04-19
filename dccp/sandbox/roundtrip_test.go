// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

const (
	roundtripDuration = 10e9                       // Duration of the experiment = 10 sec
	roundtripInterval = 100e6                      // How often we perform heartbeat writes to avoid idle periods = 100 ms
	roundtripRate     = 1e9 / roundtripInterval    // Fixed send rate for both endpoints in packets per second = 10 pps
	roundtripLatency  = 50e6                       // Latency of 50ms
	roundtripComputationalLatency = 1e6            // With no latency injection, computational effects in the test inject about 1 ms latency
)

// TestRoundtripEstimation checks that round-trip times are estimated accurately.
func TestRoundtripEstimation(t *testing.T) {

	reducer := newRoundtripMeasure(t)
	run, plex := NewRuntime("rtt")
	plex.Add(reducer)
	plex.Add(newRoundtripCheckpoint(run, t))

	clientConn, serverConn, clientToServer, _ := NewClientServerPipe(run)

	// Roundtrip estimates might be imprecise during long idle periods,
	// as a product of the CCID3 design, since during such period precise
	// estimates are not necessary. Therefore, to focus on roundtrip time
	// estimation without saturating the link, we generate sufficiently 
	// regular transmissions.

	cargo := []byte{1, 2, 3}
	buf := make([]byte, len(cargo))

	// In order to isolate roundtrip measurement testing from the complexities
	// of the send rate calculation mechanism, we fix the send rate of both
	// endpoints using the debug flag FixRate.
	clientConn.Amb().Flags().SetUint32("FixRate", roundtripRate)
	serverConn.Amb().Flags().SetUint32("FixRate", roundtripRate)

	// Increase the client—>server latency from 0 to latency at half time
	go func() {
		run.Sleep(roundtripDuration / 2)
		clientToServer.SetWriteLatency(roundtripLatency)
	}()

	cchan := make(chan int, 1)
	go func() {
		t0 := run.Now()
		for run.Now() - t0 < roundtripDuration {
			err := clientConn.WriteSegment(buf)
			if err != nil {
				break
			}
			run.Sleep(roundtripInterval)
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
