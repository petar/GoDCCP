// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// Close closes the connection, Section 8.3
func (c *Conn) Close() os.Error {
	c.slk.Lock()
	defer c.slk.Unlock()
	// Check if connection already closed
	state := c.socket.GetState()
	if state == CLOSEREQ || state == CLOSING || state == TIMEWAIT {
		return nil
	}
	if state != OPEN {
		return os.EBADF
	}
	// Transition to CLOSING
	c.teardown()
	c.inject(c.generateClose())
	c.gotoCLOSING()
	return nil
}
