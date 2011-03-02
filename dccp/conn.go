// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
)

// Conn wraps the two Endpoints (for each Half-connection) 
type Conn struct {
	flowid	*FlowID
	write	Endpoint
	read	Endpoint
}
