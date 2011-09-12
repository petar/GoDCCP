// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

// abortWith() resets the connection with Reset Code resetCode
func (c *Conn) abortWith(resetCode byte) {
	c.Lock()
	c.gotoCLOSED()
	c.inject(c.generateReset(resetCode))
	c.Unlock()
	c.teardownUser()
	c.teardownWriteLoop()
}

func (c *Conn) abortWithUnderLock(resetCode byte) {
	c.AssertLocked()
	c.gotoCLOSED()
	c.inject(c.generateReset(resetCode))
	c.teardownUser()
	c.teardownWriteLoop()
}

// abort() resets the connection with Reset Code 2, "Aborted"
func (c *Conn) abort() { c.abortWith(ResetAborted) }

// abortQuietly() aborts the connection immediately without sending Reset packets
func (c *Conn) abortQuietly() {
	c.Lock()
	c.gotoCLOSED()
	c.Unlock()
	c.teardownUser()
	c.teardownWriteLoop()
}

// teardownUser MUST be idempotent
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

// teardownWriteLoop MUST be idempotent
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
