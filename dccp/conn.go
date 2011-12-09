// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"io"
	"os"
)

// Conn 
type Conn struct {
	run      *Runtime
	logger   *Logger

	hc    HeaderConn
	scc   SenderCongestionControl
	rcc   ReceiverCongestionControl

	Mutex                       // Protects access to socket, ccidOpen and err
	socket
	ccidOpen       bool         // True if the sender and receiver CCID's have been opened
	err            error        // Reason for connection tear down

	readAppLk      Mutex
	readApp        chan []byte  // readLoop() sends application data to Read()
	writeDataLk    Mutex
	writeData      chan []byte  // Write() sends application data to writeLoop()
	writeNonDataLk Mutex
	writeNonData   chan *Header // inject() sends wire-format non-Data packets (higher priority) to writeLoop()
}

var (
	ErrEOF   = io.EOF
	ErrAbort = os.EIO
)

// Waiter returns a Waiter instance that can wait until all goroutines
// associated with the connection have completed.
func (c *Conn) Waiter() Waiter {
	return c.run.Waiter()
}

func newConn(run *Runtime, logger *Logger, hc HeaderConn, scc SenderCongestionControl, rcc ReceiverCongestionControl) *Conn {
	c := &Conn{
		run:          run,
		logger:       logger,
		hc:           hc,
		scc:          scc,
		rcc:          rcc,
		ccidOpen:     false,
		readApp:      make(chan []byte, 5),
		writeData:    make(chan []byte),
		writeNonData: make(chan *Header, 5),
	}

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

func NewConnServer(run *Runtime, logger *Logger, hc HeaderConn, 
	scc SenderCongestionControl, rcc ReceiverCongestionControl) *Conn {

	c := newConn(run, logger, hc, scc, rcc)

	c.Lock()
	c.gotoLISTEN()
	c.Unlock()

	c.run.Go(func() { c.writeLoop(c.writeNonData, c.writeData) })
	c.run.Go(func() { c.readLoop() })
	c.run.Go(func() { c.idleLoop() })
	return c
}

func NewConnClient(run *Runtime, logger *Logger, hc HeaderConn, 
	scc SenderCongestionControl, rcc ReceiverCongestionControl, serviceCode uint32) *Conn {

	c := newConn(run, logger, hc, scc, rcc)

	c.Lock()
	c.gotoREQUEST(serviceCode)
	c.Unlock()

	c.run.Go(func() { c.writeLoop(c.writeNonData, c.writeData) })
	c.run.Go(func() { c.readLoop() })
	c.run.Go(func() { c.idleLoop() })
	return c
}
