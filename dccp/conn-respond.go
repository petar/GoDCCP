// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// If socket is in RESPOND, 
// Implements Step 11, Section 8.5
func (c *Conn) processRESPOND(h *Header) os.Error {
	panic("impl?")
}
