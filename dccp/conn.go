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

	cc.Start()

	return c
}

func newConnServer(hc HeaderConn, cc CongestionControl) *Conn {
	c := newConn(hc, cc)

	c.Lock()
	c.socket.SetServer(true)
	c.socket.SetState(LISTEN)
	c.Unlock()

	go c.writeLoop()
	go c.readLoop()
	return c
}

const (
	REQUEST_BACKOFF_FIRST = 1e9 // Initial re-send period for client Request resends, in nanoseconds
	REQUEST_BACKOFF_MAX   = 2*60e9 // Request re-sends quit after 2 mins, in nanoseconds
	REQUEST_BACKOFF_FREQ  = 20e9 // Back-off Request resend every 20 mins, in nanoseconds
)

func newConnClient(hc HeaderConn, cc CongestionControl, serviceCode uint32) *Conn {
	c := newConn(hc, cc)

	c.Lock()
	c.socket.SetServer(false)
	c.socket.SetState(REQUEST)
	c.socket.SetServiceCode(serviceCode)
	iss := c.socket.ChooseISS()
	c.socket.SetGAR(iss)
	c.inject(c.generateRequest(serviceCode))
	c.Unlock()

	// Resend Request using exponential backoff, if no response
	go func() {
		b := newBackOff(REQUEST_BACKOFF_FIRST, REQUEST_BACKOFF_MAX, REQUEST_BACKOFF_FREQ)
		for {
			err := b.Sleep()
			c.Lock()
			state := c.socket.GetState()
			c.Unlock()
			if state != REQUEST {
				break
			}
			// If the back-off timer has reached maximum wait, quit trying
			if err != nil {
				c.abort()
				break
			}
			c.Lock()
			c.inject(c.generateRequest(serviceCode))
			c.Unlock()
		}
	}()

	go c.writeLoop()
	go c.readLoop()
	return c
}
