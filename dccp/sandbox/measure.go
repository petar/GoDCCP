// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"bytes"
	"fmt"
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

// Measure is a dccp.TraceWriter which listens to the logs emitted from the
// Roundtrip. It measures the real roundtrip time between the sender and
// receiver, based on read and write logs and prints out this information.
type Measure struct {
	t                      *testing.T
	env                    *dccp.Env
	leftClient             map[int64]int64  // SeqNo —> Time left client
	leftServer             map[int64]int64  // SeqNo —> Time left server

	clientToServerTransmit int64
	serverToClientTransmit int64

	clientToServerDrop     int64
	serverToClientDrop     int64

	clientToServerTriptime Moment
	serverToClientTriptime Moment
}

func NewMeasure(env *dccp.Env, t *testing.T) *Measure {
	x := &Measure{
		env: env,
		t:   t,
		leftClient: make(map[int64]int64),
		leftServer: make(map[int64]int64),
	}
	x.clientToServerTriptime.Init()
	x.serverToClientTriptime.Init()
	return x
}

func (x *Measure) Write(r *dccp.Trace) {
	now := x.env.Now()
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
				x.serverToClientTriptime.Add(float64(now - left))
				delete(x.leftServer, r.SeqNo)
				x.serverToClientTransmit++
			}
		case "server":
			left, ok := x.leftClient[r.SeqNo]
			if !ok {
				fmt.Printf("server read, no client write, seqno=%06x\n", r.SeqNo)
			} else {
				x.clientToServerTriptime.Add(float64(now - left))
				delete(x.leftClient, r.SeqNo)
				x.clientToServerTransmit++
			}
		}
	case dccp.EventDrop:
		if _, ok := x.leftClient[r.SeqNo]; ok {
			x.clientToServerDrop++
		}
		delete(x.leftClient, r.SeqNo)
		if _, ok := x.leftServer[r.SeqNo]; ok {
			x.serverToClientDrop++
		}
		delete(x.leftServer, r.SeqNo)
	}
}

func (x *Measure) Sync() error { 
	return nil 
}

func (x *Measure) Loss() (cs float64, csLoss, csTotal int64, sc float64, scLoss, scTotal int64) {
	csTotal = x.clientToServerTransmit + x.clientToServerDrop
	csLoss = x.clientToServerDrop
	if csTotal <= 0 {
		cs = 0
	} else {
		cs = float64(csLoss) / float64(csTotal)
	}
	scTotal = x.serverToClientTransmit + x.serverToClientDrop
	scLoss = x.serverToClientDrop
	if scTotal <= 0 {
		sc = 0
	} else {
		sc = float64(scLoss) / float64(scTotal)
	}
	return
}

func (x *Measure) String() string {
	var w bytes.Buffer
	cs, csLoss, csTotal, sc, scLoss, scTotal := x.Loss()
	fmt.Fprintf(&w, "     Loss: c—>s %0.1f%% (%d/%d), c<—s %0.1f%% (%d/%d)\n", 
		100*cs, csLoss, csTotal, 100*sc, scLoss, scTotal)
	fmt.Fprintf(&w, "Trip time: c—>s %0.1f/%0.1f ms, c<—s %0.1f/%0.1f\n", 
		x.clientToServerTriptime.Average()/1e6, x.clientToServerTriptime.StdDev()/1e6,
		x.serverToClientTriptime.Average()/1e6, x.serverToClientTriptime.StdDev()/1e6,
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

func (x *Measure) Close() error { 
	//fmt.Println(x.String())
	return nil 
}
