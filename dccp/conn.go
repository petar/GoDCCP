// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
)

// Conn 
type Conn struct {
	id
	hc HeaderConn
	CongestionControl
	Mutex // Protects access to socket
	socket
	readApp      chan []byte  // readLoop() sends application data to Read()
	writeData    chan []byte  // Write() sends application data to writeLoop()
	writeNonData chan *Header // inject() sends wire-format non-Data packets (higher priority) to writeLoop()
}

type id struct {
	SourcePort, DestPort uint16
	SourceAddr, DestAddr []byte
}

func newConnServer() *Conn {
	c := &Conn{
		readApp:      make(chan []byte, 3),
		writeData:    make(chan []byte),
		writeNonData: make(chan *Header, 3),
	}
	// Currently, CCID is not negotiated, rather both sides use the same
	c.socket.SetCCIDA(CCID_PETAR)
	c.socket.SetCCIDB(CCID_PETAR)
	c.socket.SetRTT(RTT_DEFAULT)
	go c.writeLoop()
	go c.readLoop()
	return c
}

func newConnClient() *Conn {
	XXX
}
