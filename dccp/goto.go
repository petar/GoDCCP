// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"log"
)

func (c *Conn) gotoLISTEN() {
	c.AssertLocked()
	c.socket.SetServer(true)
	c.socket.SetState(LISTEN)
	c.logState()
	go func() {
		GetTime().Sleep(REQUEST_BACKOFF_MAX)
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		if state != LISTEN {
			return
		}
		c.abortQuietly()
	}()
}

const RESPOND_TIMEOUT = 30e9 // Timeout in RESPOND state, 30 sec in nanoseconds

func (c *Conn) gotoRESPOND(hServiceCode uint32, hSeqNo int64) {
	c.AssertLocked()
	c.socket.SetState(RESPOND)
	c.logState()
	iss := c.socket.ChooseISS()
	c.socket.SetGAR(iss)
	c.socket.SetISR(hSeqNo)
	c.socket.SetGSR(hSeqNo)
	// TODO: To be more prudent, set service code only if it is currently 0,
	// otherwise check that h.ServiceCode matches socket service code
	c.socket.SetServiceCode(hServiceCode)

	go func() {
		GetTime().Sleep(RESPOND_TIMEOUT)
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		if state == RESPOND {
			c.abortQuietly()
		}
	}()
}

const (
	REQUEST_BACKOFF_FIRST = 1e9 // Initial re-send period for client Request resends is 1 sec, in nanoseconds
	REQUEST_BACKOFF_MAX   = 120e9 // Request re-sends quit after 2 mins, in nanoseconds
	REQUEST_BACKOFF_FREQ  = 10e9 // Back-off Request resend every 10 secs, in nanoseconds
)

func (c *Conn) gotoREQUEST(serviceCode uint32) {
	c.AssertLocked()
	c.socket.SetServer(false)
	c.socket.SetState(REQUEST)
	c.logState()
	c.socket.SetServiceCode(serviceCode)
	iss := c.socket.ChooseISS()
	c.socket.SetGAR(iss)
	c.inject(c.generateRequest(serviceCode))

	// Resend Request using exponential backoff, if no response
	go func() {
		b := newBackOff(REQUEST_BACKOFF_FIRST, REQUEST_BACKOFF_MAX, REQUEST_BACKOFF_FREQ)
		for {
			err, _ := b.Sleep()
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
			log.Printf("resend Request\n")
			c.inject(c.generateRequest(serviceCode))
			c.Unlock()
		}
	}()
}

const (
	PARTOPEN_BACKOFF_FIRST = 200e6    // 200 miliseconds in nanoseconds, Section 8.1.5
	PARTOPEN_BACKOFF_MAX   = 4 * MSL  // 8 mins in nanoseconds, Section 8.1.5
)

func (c *Conn) openCCID() {
	c.AssertLocked()
	if c.ccidOpen {
		return
	}
	c.scc.Open()
	c.rcc.Open()
	c.ccidOpen = true
	c.logEvent("CCID open")
}

func (c *Conn) closeCCID() {
	c.AssertLocked()
	if !c.ccidOpen {
		return
	}
	c.scc.Close()
	c.rcc.Close()
	c.ccidOpen = false
	c.logEvent("CCID close")
}

func (c *Conn) gotoPARTOPEN() {
	c.AssertLocked()
	c.socket.SetState(PARTOPEN)
	c.logState()
	c.openCCID()
	c.inject(nil) // Unblocks the writeLoop select, so it can see the state change

	// Start PARTOPEN timer, according to Section 8.1.5
	go func() {
		b := newBackOff(PARTOPEN_BACKOFF_FIRST, PARTOPEN_BACKOFF_MAX, PARTOPEN_BACKOFF_FIRST)
		c.Logger.Logf("conn", "Event", "PARTOPEN backoff %d start", GetTime().Nanoseconds())
		for {
			err, btm := b.Sleep()
			c.Lock()
			state := c.socket.GetState()
			c.Unlock()
			if state != PARTOPEN {
				c.Logger.Logf("conn", "Event", "PARTOPEN backoff EXIT via state change")
				break
			}
			// If the back-off timer has reached maximum wait. End the connection.
			if err != nil {
				c.abort()
				break
			}
			c.Logger.Logf("conn", "Event", "PARTOPEN backoff %d", btm)
			c.Lock()
			c.inject(c.generateAck())
			// XXX: This is a deviation from the RFC. The Sync packet necessitates a
			// SyncAck response, which moves the client from PARTOPEN to OPEN in the
			// lack of DataAck packets sent from the server to the client.
			c.inject(c.generateSync())
			c.Unlock()
		}
	}()
}

func (c *Conn) gotoOPEN(hSeqNo int64) {
	c.AssertLocked()
	c.socket.SetOSR(hSeqNo)
	c.socket.SetState(OPEN)
	c.logState()
	c.openCCID()
	c.inject(nil) // Unblocks the writeLoop select, so it can see the state change
}

func (c *Conn) gotoTIMEWAIT() {
	c.AssertLocked()
	c.teardownUser()
	c.socket.SetState(TIMEWAIT)
	c.logState()
	c.closeCCID()
	go func() {
		GetTime().Sleep(2 * MSL)
		c.abortQuietly()
	}()
}

func (c *Conn) gotoCLOSING() {
	c.AssertLocked()
	c.teardownUser()
	c.socket.SetState(CLOSING)
	c.logState()
	c.closeCCID()
	go func() {
		c.Lock()
		rtt := c.socket.GetRTT()
		c.Unlock()
		b := newBackOff(2*rtt, CLOSING_BACKOFF_MAX, CLOSING_BACKOFF_FREQ)
		for {
			err, _ := b.Sleep()
			c.Lock()
			state := c.socket.GetState()
			c.Unlock()
			if state != CLOSING {
				break
			}
			if err != nil {
				c.Lock()
				c.gotoTIMEWAIT()
				c.Unlock()
				break
			}
			c.Lock()
			c.inject(c.generateClose())
			c.Unlock()
		}
	}()
}

// gotoCLOSED MUST be idempotent
func (c *Conn) gotoCLOSED() {
	c.AssertLocked()
	c.socket.SetState(CLOSED)
	c.logState()
	c.teardownUser()
	c.teardownWriteLoop()
	c.closeCCID()
}
