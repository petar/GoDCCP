// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
)

// Conn 
type Conn struct {
	id
	hc HeaderConn
	sync.Mutex // Protects access to socket
	socket
	readApp      chan []byte // readLoop() sends application data to Read()
	writeData    chan []byte // Write() sends wire-format Data packets to injectLoop()
	writeNonData chan []byte // inject() sends wire-format non-Data packets (higher priority) to injectLoop()
}

type id struct {
	SourcePort, DestPort uint16
	SourceAddr, DestAddr []byte
}

func newConnServer() *Conn {
	c := &Conn{
		readApp:      make(chan []byte, 3),
		writeData:    make(chan []byte),
		writeNonData: make(chan []byte, 3),
	}
	c.socket.SetRTT(??)
	go c.injectLoop()
	return c
}

func newConnClient() *Conn {
	XXX
}
