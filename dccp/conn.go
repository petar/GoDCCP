// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// Conn 
type Conn struct {
	env   *Env
	amb   *Amb

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
	writeNonData   chan *writeHeader // inject() sends wire-format non-Data packets (higher priority) to writeLoop()

	writeTime      monotoneTime
}

// Joiner returns a Joiner instance that can wait until all goroutines
// associated with the connection have completed.
func (c *Conn) Joiner() Joiner {
	return c.env.Joiner()
}

// Amb returns the Amb instance associated with this connection
func (c *Conn) Amb() *Amb {
	return c.amb
}

func newConn(env *Env, amb *Amb, hc HeaderConn, scc SenderCongestionControl, rcc ReceiverCongestionControl) *Conn {
	c := &Conn{
		env:          env,
		amb:          amb,
		hc:           hc,
		scc:          scc,
		rcc:          rcc,
		ccidOpen:     false,
		readApp:      make(chan []byte, 5),
		writeData:    make(chan []byte),
		writeNonData: make(chan *writeHeader, 5),
	}
	c.writeTime.Init(env)

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

func NewConnServer(env *Env, amb *Amb, hc HeaderConn, 
	scc SenderCongestionControl, rcc ReceiverCongestionControl) *Conn {

	c := newConn(env, amb, hc, scc, rcc)

	c.Lock()
	c.gotoLISTEN()
	c.Unlock()

	c.env.Go(func() { c.writeLoop(c.writeNonData, c.writeData) }, "ConnServer·writLoop")
	c.env.Go(func() { c.readLoop() }, "ConnServer·readLoop")
	c.env.Go(func() { c.idleLoop() }, "ConnServer·idleLoop")
	return c
}

func NewConnClient(env *Env, amb *Amb, hc HeaderConn, 
	scc SenderCongestionControl, rcc ReceiverCongestionControl, serviceCode uint32) *Conn {

	c := newConn(env, amb, hc, scc, rcc)

	c.Lock()
	c.gotoREQUEST(serviceCode)
	c.Unlock()

	c.env.Go(func() { c.writeLoop(c.writeNonData, c.writeData) }, "ConnClient·writeLoop")
	c.env.Go(func() { c.readLoop() }, "ConnClient·readLoop")
	c.env.Go(func() { c.idleLoop() }, "ConnClient·idleLoop")
	return c
}
