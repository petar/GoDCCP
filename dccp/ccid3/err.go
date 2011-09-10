// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import "os"

// TODO: Annotate each error with the circumstances that can cause it
var (
	ErrMissingOption = os.NewError("missing option")
	ErrNoAck         = os.NewError("packet is not an ack")
)
