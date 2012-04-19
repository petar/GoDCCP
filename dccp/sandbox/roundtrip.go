// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"testing"
	"time"
	"github.com/petar/GoDCCP/dccp"
	"github.com/petar/GoDCCP/dccp/ccid3"
)

// roundtripMeasure is a dccp.Guzzle which listens to the logs emitted from the
// Roundtrip. It measures the real roundtrip time between the sender and
// receiver, based on read and write logs and prints out this information.
type roundtripMeasure struct {
	t              *testing.T
	leftClient     map[int64]int64  // SeqNo —> Time left client
	leftServer     map[int64]int64  // SeqNo —> Time left server
	clientToServer Moment
	serverToClient Moment
}

func newRoundtripMeasure(t *testing.T) *roundtripMeasure {
	x := &roundtripMeasure{
		t: t,
		leftClient: make(map[int64]int64),
		leftServer: make(map[int64]int64),
	}
	x.clientToServer.Init()
	x.serverToClient.Init()
	return x
}

func (x *roundtripMeasure) Write(r *dccp.LogRecord) {
	now := time.Now().UnixNano()
	switch r.Event {
	case dccp.EventWrite:
		switch r.Labels[0] {
		case "client":
			x.leftClient[r.SeqNo] = now
		case "server":
			x.leftServer[r.SeqNo] = now
		}
	case dccp.EventRead:
		switch r.Labels[0] {
		case "client":
			left, ok := x.leftServer[r.SeqNo]
			if !ok {
				fmt.Printf("client read, no server write, seqno=%06x\n", r.SeqNo)
			} else {
				x.serverToClient.Add(float64(now - left))
				delete(x.leftServer, r.SeqNo)
			}
		case "server":
			left, ok := x.leftClient[r.SeqNo]
			if !ok {
				fmt.Printf("server read, no client write, seqno=%06x\n", r.SeqNo)
			} else {
				x.clientToServer.Add(float64(now - left))
				delete(x.leftClient, r.SeqNo)
			}
		}
	case dccp.EventDrop:
		delete(x.leftClient, r.SeqNo)
		delete(x.leftServer, r.SeqNo)
	}
}

func (x *roundtripMeasure) Sync() error { 
	return nil 
}

func (x *roundtripMeasure) String() string {
	var w bytes.Buffer
	fmt.Fprintf(&w, "c—>s %0.1f/%0.1f ms, c<—s %0.1f/%0.1f\n", 
		x.clientToServer.Average()/1e6, x.clientToServer.StdDev()/1e6,
		x.serverToClient.Average()/1e6, x.serverToClient.StdDev()/1e6,
	)
	if len(x.leftClient) > 0 {
		fmt.Fprintf(&w, "Left client and unclaimed:\n")
	}
	for s, _ := range x.leftClient {
		fmt.Fprintf(&w, "  %06x\n", s)
	}
	if len(x.leftServer) > 0 {
		fmt.Fprintf(&w, "Left server and unclaimed:\n")
	}
	for s, _ := range x.leftServer {
		fmt.Fprintf(&w, "  %06x\n", s)
	}
	return string(w.Bytes())
}

func (x *roundtripMeasure) Close() error { 
	//fmt.Println(x.String())
	return nil 
}

// roundtripCheckpoint verifies that roundtrip estimates are within expected at
// different point in time in the Roundtrip test.
type roundtripCheckpoint struct {
	run   *dccp.Runtime
	t     *testing.T

	checkTimes    []int64		// Times when test conditions are checked
	expected      []float64		// Approximate expected value of variables at respective times
	tolerance     []float64         // Multiplicative error tolerance with respect to expected

	clientElapsed []float64		// Readings for test variables from client of roundtrip estimate after "elapsed" event
	clientReport  []float64
	serverElapsed []float64
	serverReport  []float64

}

func newRoundtripCheckpoint(run *dccp.Runtime, t *testing.T) *roundtripCheckpoint {
	return &roundtripCheckpoint{
		run: run,
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
			slot[i] = reading
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
			t.Errorf("%s deviates by %0.2f%% in i-th term", name, dev * 100, i)
		}
	}
}
