// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"bytes"
	"fmt"
	"testing"
	"time"
	"github.com/petar/GoDCCP/dccp"
)

// roundtripReducer is a dccp.Guzzle which listens to the logs
// emitted from the RTT test and performs various checks.
type roundtripReducer struct {
	t *testing.T
	leftClient map[int64]int64  // SeqNo —> Time left client
	leftServer map[int64]int64  // SeqNo —> Time left server
	clientToServer Moment
	serverToClient Moment
}

func newRoundtripReducer(t *testing.T) *roundtripReducer {
	x := &roundtripReducer{
		t: t,
		leftClient: make(map[int64]int64),
		leftServer: make(map[int64]int64),
	}
	x.clientToServer.Init()
	x.serverToClient.Init()
	return x
}

func (x *roundtripReducer) Write(r *dccp.LogRecord) {
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

func (x *roundtripReducer) Sync() error { 
	return nil 
}

func (x *roundtripReducer) String() string {
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

func (x *roundtripReducer) Close() error { 
	fmt.Println(x.String())
	return nil 
}
