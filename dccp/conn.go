// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
)

// Conn 
type Conn struct {
	id
	hc HeaderConn
	slk sync.Mutex // Protects access to socket
	socket
}

type id struct {
	SourcePort, DestPort uint16
	SourceAddr, DestAddr []byte
}
