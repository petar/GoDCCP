// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// inject() blocks until the block can be sent while respecting
// whatever rate-limiting policy is in use
func (c *Conn) inject(block []byte) os.Error {
	??
}
