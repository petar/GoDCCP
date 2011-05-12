// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// CongestionControl abstracts away the congestion control logic of a 
// DCCP connection.
type CongestionControl interface {
	GetCCMPS() uint32 // Returns the Congestion Control Maximum Packet Size, CCMPS. Generally, PMTU <= CCMPS
}

const (
)
