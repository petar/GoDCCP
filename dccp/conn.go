// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "github.com/petar/GoGauge/context"

// Conn 
type Conn struct {
	log   logger

	hc    HeaderConn
	scc   SenderCongestionControl
	rcc   ReceiverCongestionControl

	Mutex // Protects access to socket
	socket

	readAppLk      Mutex
	readApp        chan []byte  // readLoop() sends application data to Read()
	writeDataLk    Mutex
	writeData      chan []byte  // Write() sends application data to writeLoop()
	writeNonDataLk Mutex
	writeNonData   chan *Header // inject() sends wire-format non-Data packets (higher priority) to writeLoop()
}

func newConn(name string, hc HeaderConn, scc SenderCongestionControl, rcc ReceiverCongestionControl) *Conn {
	c := &Conn{
		hc:           hc,
		scc:          newSenderCCActuator(scc),
		rcc:          newReceiverCCActuator(rcc),
		readApp:      make(chan []byte, 5),
		writeData:    make(chan []byte),
		writeNonData: make(chan *Header, 5),
	}
	c.log.Init(context.NewContext(name))

	c.Lock()
	// Currently, CCID is not negotiated, rather both sides use the same
	c.socket.SetCCIDA(scc.GetID())
	c.socket.SetCCIDB(rcc.GetID())

	// REMARK: SWAF/SWBF are currently not implemented. 
	// Instead, we use wide enough fixed-size windows
	c.socket.SetSWAF(SEQWIN_FIXED)
	c.socket.SetSWBF(SEQWIN_FIXED)

	c.syncWithLink()
	c.syncWithCongestionControl()
	c.Unlock()

	return c
}

func NewConnServer(name string, hc HeaderConn, scc SenderCongestionControl, rcc ReceiverCongestionControl) *Conn {
	c := newConn(name, hc, scc, rcc)

	c.Lock()
	c.gotoLISTEN()
	c.Unlock()

	go c.writeLoop(c.writeNonData, c.writeData)
	go c.readLoop()
	return c
}

func NewConnClient(name string, hc HeaderConn, scc SenderCongestionControl, rcc ReceiverCongestionControl, serviceCode uint32) *Conn {
	c := newConn(name, hc, scc, rcc)

	c.Lock()
	c.gotoREQUEST(serviceCode)
	c.Unlock()

	go c.writeLoop(c.writeNonData, c.writeData)
	go c.readLoop()
	return c
}
