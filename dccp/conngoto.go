// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "time"

func (c *Conn) gotoLISTEN() {
	c.AssertLocked()
	c.socket.SetServer(true)
	c.socket.SetState(LISTEN)
	go func() {
		time.Sleep(REQUEST_BACKOFF_MAX)
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		if state != LISTEN {
			return
		}
		c.abortQuietly()
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
	c.socket.SetServiceCode(serviceCode)
	iss := c.socket.ChooseISS()
	c.socket.SetGAR(iss)
	c.inject(c.generateRequest(serviceCode))

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
}

func (c *Conn) gotoTIMEWAIT() {
	c.AssertLocked()
	c.socket.SetState(TIMEWAIT)
	go func() {
		time.Sleep(2 * MSL)
		c.abortQuietly()
	}()
}

func (c *Conn) gotoCLOSING() {
	c.AssertLocked()
	c.socket.SetState(CLOSING)
	go func() {
		c.Lock()
		rtt := c.socket.GetRTT()
		c.Unlock()
		b := newBackOff(2*rtt, CLOSING_BACKOFF_MAX, CLOSING_BACKOFF_FREQ)
		for {
			err := b.Sleep()
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
