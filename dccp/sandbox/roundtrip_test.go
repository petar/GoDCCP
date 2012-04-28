// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"fmt"
	"math"
	"testing"
	"os"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
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
	dccp.InstallCtrlCPanic()

	env, plex := NewEnv("rtt")
	reducer := NewMeasure(env, t)
	plex.Add(reducer)
	plex.Add(newRoundtripCheckpoint(env, t))
	plex.HighlightSamples(ccid3.RoundtripElapsedSample, ccid3.RoundtripReportSample)

	clientConn, serverConn, clientToServer, _ := NewClientServerPipe(env)

	// Roundtrip estimates might be imprecise during long idle periods,
	// as a product of the CCID3 design, since during such period precise
	// estimates are not necessary. Therefore, to focus on roundtrip time
	// estimation without saturating the link, we generate sufficiently 
	// regular transmissions.

	payload := []byte{1, 2, 3}
	buf := make([]byte, len(payload))

	// In order to isolate roundtrip measurement testing from the complexities
	// of the send rate calculation mechanism, we fix the send rate of both
	// endpoints using the debug flag FixRate.
	clientConn.Amb().Flags().SetUint32("FixRate", roundtripRate)
	serverConn.Amb().Flags().SetUint32("FixRate", roundtripRate)

	// Increase the client—>server latency from 0 to latency at half time
	env.Go(func() {
		env.Sleep(roundtripDuration / 2)
		clientToServer.SetWriteLatency(roundtripLatency)
	}, "test controller")

	cchan := make(chan int, 1)
	env.Go(func() {
		t0 := env.Now()
		for env.Now() - t0 < roundtripDuration {
			err := clientConn.Write(buf)
			if err != nil {
				break
			}
		}
		// Close is necessary because otherwise, if no read timeout is in place, the
		// server sides hangs forever on Read
		clientConn.Close()
		close(cchan)
	}, "test client")

	schan := make(chan int, 1)
	env.Go(func() {
		for {
			_, err := serverConn.Read()
			if err != nil {
				break
			}
		}
		close(schan)
	}, "test server")

	_, _ = <-cchan
	_, _ = <-schan

	// Shutdown the connections properly
	clientConn.Abort()
	serverConn.Abort()
	env.NewGoJoin("end-of-test", clientConn.Joiner(), serverConn.Joiner()).Join()
	dccp.NewAmb("line", env).E(dccp.EventMatch, "Server and client done.")
	if err := env.Close(); err != nil {
		t.Errorf("error closing runtime (%s)", err)
	}
}

// roundtripCheckpoint verifies that roundtrip estimates are within expected at
// different point in time in the Roundtrip test.
type roundtripCheckpoint struct {
	env   *dccp.Env
	t     *testing.T

	checkTimes    []int64		// Times when test conditions are checked
	expected      []float64		// Approximate expected value of variables at respective times
	tolerance     []float64         // Multiplicative error tolerance with respect to expected

	clientElapsed []float64		// Readings for test variables from client of roundtrip estimate after "elapsed" event
	clientReport  []float64
	serverElapsed []float64
	serverReport  []float64

}

func newRoundtripCheckpoint(env *dccp.Env, t *testing.T) *roundtripCheckpoint {
	return &roundtripCheckpoint{
		env: env,
		t:   t,
		checkTimes:    []int64{roundtripDuration / 2, roundtripDuration},
		expected:      []float64{NanoToMilli(roundtripComputationalLatency), NanoToMilli(roundtripLatency)},
		tolerance:     []float64{1.0, 0.15},
		clientElapsed: make([]float64, 2),
		clientReport:  make([]float64, 2),
		serverElapsed: make([]float64, 2),
		serverReport:  make([]float64, 2),
	}
}

func (x *roundtripCheckpoint) Write(r *dccp.LogRecord) {
	reading, ok := r.Sample()
	if !ok {
		return
	}

	var slot []float64
	switch {
	case r.ArgOfType(ccid3.RoundtripElapsedCheckpoint) != nil:
		endpoint := r.Labels[0]
		switch endpoint {
		case "client":
			slot = x.clientElapsed
		case "server":
			slot = x.serverElapsed
		}
	case r.ArgOfType(ccid3.RoundtripReportCheckpoint) != nil:
		endpoint := r.Labels[0]
		switch endpoint {
		case "client":
			slot = x.clientReport
		case "server":
			slot = x.serverReport
		}
	}
	if slot == nil {
		return
	}
	for i, checkTime := range x.checkTimes {
		if r.Time < checkTime {
			slot[i] = reading.Value
		}
	}
}

func (x *roundtripCheckpoint) Sync() error { 
	return nil 
}

func (x *roundtripCheckpoint) Close() error { 
	checkDeviation(x.t, "client-elapsed", x.clientElapsed, x.expected, x.tolerance)
	checkDeviation(x.t, "client-report", x.clientReport, x.expected, x.tolerance)
	checkDeviation(x.t, "server-elapsed", x.serverElapsed, x.expected, x.tolerance)
	checkDeviation(x.t, "server-report", x.serverReport, x.expected, x.tolerance)
	return nil 
}

func checkDeviation(t *testing.T, name string, actual []float64, expected []float64, tolerance []float64) {
	for i, _ := range actual {
		dev := math.Abs(actual[i]-expected[i]) / expected[i]
		if dev > tolerance[i] {
			fmt.Fprintf(os.Stderr, "%s=%v, expected=%v, tolerance=%v\n", name, actual, expected, tolerance)
			t.Errorf("%s deviates by %0.2f%% in %d-th term", name, dev * 100, i)
		}
	}
}
