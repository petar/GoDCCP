// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// abortWith() resets the connection with Reset Code resetCode
func (c *Conn) abortWith(resetCode byte) {
	c.Lock()
	c.setError(ErrAbort)
	c.gotoCLOSED()
	c.inject(c.generateReset(resetCode))
	c.Unlock()
	c.teardownUser()
	c.teardownWriteLoop()
}

// abort() resets the connection with Reset Code 2, "Aborted"
func (c *Conn) abort() { c.abortWith(ResetAborted) }

func (c *Conn) setError(err error) {
	c.AssertLocked()
	if c.err != nil {
		return
	}
	c.err = err
}

func (c *Conn) reset(resetCode byte, err error) {
	c.AssertLocked()
	c.setError(err)
	c.gotoCLOSED()
	c.inject(c.generateReset(resetCode))
	c.teardownUser()
	c.teardownWriteLoop()
}

// abortQuietly() aborts the connection immediately without sending Reset packets
func (c *Conn) abortQuietly() {
	c.Lock()
	c.setError(ErrAbort)
	c.gotoCLOSED()
	c.Unlock()
	c.teardownUser()
	c.teardownWriteLoop()
}

// teardownUser MUST be idempotent. It may be called with or without lock on c.
func (c *Conn) teardownUser() {
	c.readAppLk.Lock()
	if c.readApp != nil {
		close(c.readApp)
		c.readApp = nil
	}
	c.readAppLk.Unlock()
	c.writeDataLk.Lock()
	if c.writeData != nil {
		close(c.writeData)
		c.writeData = nil
	}
	c.writeDataLk.Unlock()
}

// teardownWriteLoop MUST be idempotent. It may be called with or without lock on c.
func (c *Conn) teardownWriteLoop() {
	c.writeNonDataLk.Lock()
	defer c.writeNonDataLk.Unlock()
	if c.writeNonData != nil {
		close(c.writeNonData)
		c.writeNonData = nil
	}
	c.scc.Close()
	c.rcc.Close()
}
