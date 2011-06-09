// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

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

	cc.Start()

	return c
}

func newConnServer(hc HeaderConn, cc CongestionControl) *Conn {
	c := newConn(hc, cc)

	c.Lock()
	c.gotoLISTEN()
	c.Unlock()

	go c.writeLoop()
	go c.readLoop()
	return c
}

func newConnClient(hc HeaderConn, cc CongestionControl, serviceCode uint32) *Conn {
	c := newConn(hc, cc)

	c.Lock()
	c.gotoREQUEST(serviceCode)
	c.Unlock()

	go c.writeLoop()
	go c.readLoop()
	return c
}
