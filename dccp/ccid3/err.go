// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import "errors"

// TODO: Annotate each error with the circumstances that can cause it
var (
	ErrMissingOption = errors.New("missing option")
	ErrNoAck         = errors.New("packet is not an ack")
)
