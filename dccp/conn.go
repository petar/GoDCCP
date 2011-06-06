// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import ()

// Conn 
type Conn struct {
	hc    HeaderConn
	cc    CongestionControl
	Mutex // Protects access to socket
	socket
	readApp      chan []byte  // readLoop() sends application data to Read()
	writeData    chan []byte  // Write() sends application data to writeLoop()
	writeNonData chan *Header // inject() sends wire-format non-Data packets (higher priority) to writeLoop()
}

func newConn(hc HeaderConn, cc CongestionControl) *Conn {
	c := &Conn{
		hc:           hc,
		cc:           cc,
		readApp:      make(chan []byte, 3),
		writeData:    make(chan []byte),
		writeNonData: make(chan *Header, 3),
	}

	c.Lock()
	// Currently, CCID is not negotiated, rather both sides use the same
	c.socket.SetCCIDA(cc.GetID())
	c.socket.SetCCIDB(cc.GetID())
	c.updateSocketLink()
	c.updateSocketCongestionControl()
	c.Unlock()

	return c
}

func newConnServer(hc HeaderConn, cc CongestionControl) *Conn {
	c := newConn(hc, cc)

	c.Lock()
	c.socket.SetState(LISTEN)
	c.Unlock()

	go c.writeLoop()
	go c.readLoop()
	return c
}

func newConnClient(hc HeaderConn, cc CongestionControl, serviceCode uint32) *Conn {
	c := newConn(hc, cc)

	c.Lock()
	c.socket.SetState(REQUEST)
	c.socket.SetServiceCode(serviceCode)
	iss := c.socket.ChooseISS()
	c.socket.SetGAR(iss)  // ???
	c.inject(c.generateRequest(serviceCode))
	c.Unlock()

	// Exponential backoff if not response

	// A client in the REQUEST state SHOULD use an exponential-backoff timer to send new
	// DCCP-Request packets if no response is received.  The first retransmission should occur
	// after approximately one second, backing off to not less than one packet every 64 seconds;
	// Each new DCCP-Request MUST increment the Sequence Number by one and MUST contain the same
	// Service Code and application data as the original DCCP-Request.

	// A client MAY give up on its DCCP-Requests after some time (3 minutes, for example).  When
	// it does, it SHOULD send a DCCP-Reset packet to the server with Reset Code 2, "Aborted",
	// to clean up state in case one or more of the Requests actually arrived.  A client in
	// REQUEST state has never received an initial sequence number from its peer, so the
	// DCCP-Reset's Acknowledgement Number MUST be set to zero.

	go c.writeLoop()
	go c.readLoop()
	return c
}
